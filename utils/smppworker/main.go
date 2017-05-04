package main

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/message"
	"bitbucket.org/codefreak/hsmpp/smpp/db/sphinx"
	"bitbucket.org/codefreak/hsmpp/smpp/influx"
	"bitbucket.org/codefreak/hsmpp/smpp/license"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"flag"
	log "github.com/Sirupsen/logrus"
	fiorix "github.com/fiorix/go-smpp/smpp"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	c        *smpp.Config
	s        smpp.Sender
	r        queue.MQ
	sconn    *smpp.Conn
	connid   = flag.String("cid", "", "Pass smpp connection id of connection this worker is going to send sms to.")
	group    = flag.String("group", "", "Group name of connection.")
	dlvTick  *time.Ticker
	sendTick *time.Ticker
	bucket   chan int
)

const (
	//ThrottlingError is 0x00000058 status
	ThrottlingError = "throttling error"
	//RetryCount is number of times we should retry sending throttling error messsages
	RetryCount = 30
)

// Handler is called by rabbitmq library after a queue has been bound/
// deliveries channel gets data when a new job is to be consumed by worker
// This function should wait for done channel before terminating so that all
// pending jobs should be finished and rabbitmq should be notified about disconnect
func handler(d queue.QueueDelivery) {
	var i queue.Item
	err := i.FromJSON(d.Body())
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"body": d.Body,
		}).Error("Failed in parsing json.")
		d.Nack(false, true)
		return
	}
	<-dlvTick.C
	for c := 1; c < i.Total; c++ {
		<-dlvTick.C
	}
	go send(i)
	d.Ack(false)
}

// This function also increments count by ceil of number of characters divided by number of characters per message.
// When count reaches a certain number defined per connection, worker waits for time t defined in configuration before resuming operations.
func send(i queue.Item) {
	m, err := message.Get(i.MsgID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
			"id":  i.MsgID,
		}).Error("Failed in fetching message from db.")
		return
	}
	if m.Status != message.Queued {
		log.Info("Message is not queued skipping.")
		return
	}
	if m.SendAfter != "" && m.SendBefore != "" {
		afterParts := strings.Split(m.SendAfter, ":")
		beforeParts := strings.Split(m.SendBefore, ":")

		hour, _ := strconv.ParseInt(afterParts[0], 10, 32)
		minute, _ := strconv.ParseInt(afterParts[1], 10, 32)
		now := time.Now().UTC()
		//7  or 23
		afterTime := time.Date(now.Year(), now.Month(), now.Day(), int(hour), int(minute), 0, 0, now.Location())
		hour, _ = strconv.ParseInt(beforeParts[0], 10, 32)
		minute, _ = strconv.ParseInt(beforeParts[1], 10, 32)
		//19 or 1
		beforeTime := time.Date(now.Year(), now.Month(), now.Day(), int(hour), int(minute), 0, 0, now.Location())
		//if 1 is less than 23
		// then 1 is on next day
		if beforeTime.Unix() < afterTime.Unix() {
			beforeTime = beforeTime.AddDate(0, 0, 1)
		}
		// if 2 is greater than 23 and 2 is lesser than 01 the next day //false, schedule it
		// if 00:01 is greater than 23 and 00:01 is lesser than 01 the next day // true, send it now
		// if 16 is greater than 7 and 16 is lesser than 19 // true, send it now
		// if 20 is greater than 7 and 20 is lesser than 19// false, schedule it next day at 7:01
		if !(now.Unix() > afterTime.Unix() && now.Unix() < beforeTime.Unix()) {
			//don't send msg here
			scheduledTime := afterTime.Add(time.Second * 1)
			if now.Unix() > beforeTime.Unix() {
				scheduledTime = scheduledTime.AddDate(0, 0, 1)
			}
			log.WithField("time", scheduledTime.String()).Info("Scheduling message.")
			m.ScheduledAt = scheduledTime.Unix()
			m.Status = message.Scheduled
			m.Update()
			return
		}
	}
	var respID string

	inf, err := influx.GetClient()
	if err != nil {
		log.WithError(err).Error("Couldn't get influxdb client")
		os.Exit(2)
	}
	sent := int64(0)
	if i.Total == 1 {
		for j := 1; j <= RetryCount; j++ {
			bucket <- 1
			if sent == 0 {
				sent = time.Now().UTC().Unix()
			}
			<-sendTick.C
			start := time.Now()
			respID, err = s.Send(m.Src, m.Dst, m.Enc, m.RealMsg, m.IsFlash)
			go inf.AddPoint(&influx.Point{
				Measurement: "message",
				Tags: influx.Tags{
					"Connection":      sconn.ID,
					"ConnectionGroup": m.ConnectionGroup,
					"User":            m.Username,
					"Src":             m.Src,
				},
				Fields: influx.Fields{
					"total":    1.0,
					"duration": time.Now().Sub(start).Seconds(),
				},
				Time: time.Now(),
			})
			<-bucket
			if err == nil || (err != nil && err.Error() != ThrottlingError) {
				break
			}
			log.WithError(err).Infof("Error occured, retrying.")
		}
	} else {
		sm, parts := s.SplitLong(m.Src, m.Dst, m.Enc, m.RealMsg, m.IsFlash)
		for i, p := range parts {
			for {
				bucket <- 1
				if sent == 0 {
					sent = time.Now().UTC().Unix()
				}
				<-sendTick.C
				start := time.Now()
				respID, err = s.SendPart(sm, p)
				go inf.AddPoint(&influx.Point{
					Measurement: "message",
					Tags: influx.Tags{
						"Connection":      sconn.ID,
						"ConnectionGroup": m.ConnectionGroup,
						"User":            m.Username,
						"Src":             m.Src,
					},
					Fields: influx.Fields{
						"total":    1.0,
						"duration": time.Now().Sub(start).Seconds(),
					},
					Time: time.Now(),
				})
				<-bucket
				log.WithField("part", i+1).Info("Sent part")
				if err == nil || (err != nil && err.Error() != ThrottlingError) {
					break
				}
				log.WithError(err).Infof("Error occured, retrying.")
			}
		}
	}
	if err != nil {
		log.WithFields(log.Fields{
			"Src":    m.Src,
			"Dst":    m.Dst,
			"err":    err,
			"Enc":    m.Enc,
			"Fields": s.GetFields(),
		}).Error("Couldn't send message.")
		if err == fiorix.ErrNotConnected {
			log.Error("SMPP not connected. Aborting worker.")
			//exit code 2, because supervisord wont restart this
			os.Exit(2)
		}
		go updateMessage(m, respID, sconn.ID, err.Error(), sent)
	} else {
		log.WithFields(log.Fields{
			"Src":    m.Src,
			"Dst":    m.Dst,
			"Enc":    m.Enc,
			"Fields": s.GetFields(),
		}).Info("Sent message.")
		go updateMessage(m, respID, sconn.ID, "", sent)
	}
	log.WithField("RespID", respID).Info("response id")
}

// When SIGTERM or SIGINT is received, this routine will make sure we shutdown our queues and finish in progress jobs
func gracefulShutdown() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)
	go func() {
		<-sig
		log.Print("Sutting down gracefully.")
		s.Close()
		os.Exit(0)
	}()
}

// Binds to rabbitmq queue and listens for all numbers starting with supplied prefixes.
// This function calls handler when a connection is succesfully established
func bind() {
	var err error
	sconn = &smpp.Conn{}
	*sconn, err = c.GetConn(*group, *connid)
	if err != nil {
		log.WithField("connid", connid).Fatalf("Couldn't get connection from settings. Check your settings and passed connection id parameter.")
	}
	err = smpp.ConnectFiorix(&fiorix.Transceiver{
		Addr:    sconn.URL,
		User:    sconn.User,
		Passwd:  sconn.Passwd,
		Handler: receiver,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"Addr":   sconn.URL,
			"User":   sconn.User,
			"Passwd": sconn.Passwd,
			"Error":  err,
		}).Error("Aborting due to connection error.")
		os.Exit(2)
	}
	s = smpp.GetSender()
	go s.ConnectOrDie()
	defer s.Close()
	s.SetFields(sconn.Fields)
	r, err = queue.ConnectRabbitMQ(viper.GetString("RABBITMQ_URL"), viper.GetString("RABBITMQ_EXCHANGE"), 1)
	if err != nil {
		log.WithError(err).Error("Couldn't get queue")
		os.Exit(2)
	}
	cl, err := influx.Connect(viper.GetString("INFLUXDB_ADDR"), viper.GetString("INFLUXDB_USERNAME"), viper.GetString("INFLUXDB_PASSWORD"))
	if err != nil {
		log.WithError(err).Error("Couldn't connect to influxdb")
		os.Exit(2)
	}
	defer cl.Close()
	go writeInfluxBatch()
	rate := time.Second / time.Duration(sconn.Size)
	dlvTick = time.NewTicker(rate)
	defer dlvTick.Stop()
	sendTick = time.NewTicker(rate)
	defer sendTick.Stop()
	//bucket helps in keeping at max Size concurrent network requests at a time
	bucket = make(chan int, sconn.Size)
	defer close(bucket)
	log.WithField("Pfxs", sconn.Pfxs).Info("Binding to routing keys")
	for i := range sconn.Pfxs {
		sconn.Pfxs[i] = *group + "-" + sconn.Pfxs[i]
	}
	err = r.Bind(sconn.Pfxs, handler)
	defer r.Close()
	if err != nil {
		os.Exit(2)
	}
	//Listen for termination signals from OS
	go gracefulShutdown()

	forever := make(<-chan int)
	<-forever
}

func main() {
	go license.CheckExpiry()
	flag.Parse()
	if *connid == "" {
		flag.Usage()
		os.Exit(2)
	}
	log.Info("Connecting database.")
	conn, err := db.Connect(viper.GetString("MYSQL_HOST"), viper.GetInt("MYSQL_PORT"), viper.GetString("MYSQL_DBNAME"), viper.GetString("MYSQL_USER"), viper.GetString("MYSQL_PASSWORD"))
	if err != nil {
		log.WithError(err).Fatal("Couldn't setup database connection.")
	}
	defer conn.Db.Close()
	c = &smpp.Config{}
	*c, err = smpp.GetConfig()
	if err != nil {
		log.Fatal("Can't continue without settings. Exiting.")
	}
	spconn, err := sphinx.Connect(viper.GetString("SPHINX_HOST"), viper.GetInt("SPHINX_PORT"))
	if err != nil {
		log.WithError(err).Fatalf("Error in connecting to sphinx.")
	}
	defer spconn.Db.Close()
	bind()
}
