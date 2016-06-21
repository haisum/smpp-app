package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/license"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	log "github.com/Sirupsen/logrus"
	smppstatus "github.com/fiorix/go-smpp/smpp"
	"github.com/streadway/amqp"
)

var (
	c       *smpp.Config
	s       *smpp.Sender
	sconn   *smpp.Conn
	connid  = flag.String("cid", "", "Pass smpp connection id of connection this worker is going to send sms to.")
	group   = flag.String("group", "", "Group name of connection.")
	tick    *time.Ticker
	errTick *time.Ticker
	bucket  chan int
)

const (
	//ThrottlingError is 0x00000058 status
	ThrottlingError = "throttling error"
)

// Handler is called by rabbitmq library after a queue has been bound/
// deliveries channel gets data when a new job is to be consumed by worker
// This function should wait for done channel before terminating so that all
// pending jobs should be finished and rabbitmq should be notified about disconnect
func handler(deliveries <-chan amqp.Delivery, done chan error) {
	for d := range deliveries {
		var i queue.Item
		err := i.FromJSON(d.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"err":  err,
				"body": d.Body,
			}).Error("Failed in parsing json.")
			d.Nack(false, true)
			return
		}
		<-tick.C
		for c := 1; c < i.Total; c++ {
			<-tick.C
		}
		go send(i)
		d.Ack(false)
	}
	log.Printf("handle: deliveries channel closed")
	done <- nil
}

// This is called per job and as a separate go routing
// This function is responsible for acknowledging the job completion to rabbitmq
// This function also increments count by ceil of number of characters divided by number of characters per message.
// When count reaches a certain number defined per connection, worker waits for time t defined in configuration before resuming operations.
func send(i queue.Item) {
	m, err := models.GetMessage(i.MsgID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
			"id":  i.MsgID,
		}).Error("Failed in fetching message from db.")
		return
	}
	if m.Status == models.MsgStopped {
		log.Info("Message is stopped skipping.")
		return
	}
	log.WithField("Status", m.Status).Info("Message Status")

	if m.SendAfter != "" && m.SendBefore != "" {
		afterParts := strings.Split(m.SendAfter, ":")
		beforeParts := strings.Split(m.SendBefore, ":")

		hour, _ := strconv.ParseInt(afterParts[0], 10, 32)
		minute, _ := strconv.ParseInt(afterParts[1], 10, 32)
		now := time.Now().UTC()
		afterTime := time.Date(now.Year(), now.Month(), now.Day(), int(hour), int(minute), 0, 0, now.Location())
		hour, _ = strconv.ParseInt(beforeParts[0], 10, 32)
		minute, _ = strconv.ParseInt(beforeParts[1], 10, 32)
		beforeTime := time.Date(now.Year(), now.Month(), now.Day(), int(hour), int(minute), 0, 0, now.Location())
		fmt.Println(now.String())
		fmt.Println(beforeTime.String())
		fmt.Println(afterTime.String())
		if beforeTime.Unix() < afterTime.Unix() {
			beforeTime.AddDate(0, 0, 1)
		}
		log.WithFields(log.Fields{"after": afterTime.String(), "before": beforeTime.String(), "now": now.String()}).Info("Scheduling")
		if !(now.After(afterTime) && now.Before(beforeTime)) {
			//don't send msg here
			scheduledTime := afterTime.AddDate(0, 0, 1).Add(time.Minute)
			log.WithField("time", scheduledTime.String()).Info("Scheduling message.")
			m.ScheduledAt = scheduledTime.Unix()
			m.Status = models.MsgScheduled
			m.Update()
			return
		}
	}
	var respID string
	sent := time.Now().UTC().Unix()
	if i.Total == 1 {
		for j := 1; j <= 10; j++ {
			bucket <- 1
			respID, err = s.Send(m.Src, m.Dst, m.Enc, i.Msg)
			<-bucket
			if err != nil {
				if err.Error() != ThrottlingError {
					break
				} else {
					log.WithError(err).Infof("Error occured, retrying for %d time.", j)
					<-errTick.C
				}
			} else {
				break
			}
		}
	} else {
		sm, parts := s.SplitLong(m.Src, m.Dst, m.Enc, i.Msg)
		for i, p := range parts {
			for j := 1; j <= 10; j++ {
				bucket <- 1
				respID, err = s.SendPart(sm, p)
				<-bucket
				log.WithField("part", i+1).Info("Sent part")
				if err != nil {
					if err.Error() != ThrottlingError {
						break
					} else {
						log.WithError(err).Infof("Error occured, retrying for %d time.", j)
						<-errTick.C
					}
				} else {
					break
				}
			}
		}
	}
	if err != nil {
		log.WithFields(log.Fields{
			"Src":    m.Src,
			"Dst":    m.Dst,
			"err":    err,
			"Enc":    m.Enc,
			"Fields": s.Fields,
		}).Error("Couldn't send message.")
		if err == smppstatus.ErrNotConnected {
			log.Error("SMPP not connected. Aborting worker.")
			//exit code 2, because supervisord wont restart this
			os.Exit(2)
		}
		go updateMessage(m, respID, sconn.ID, err.Error(), s.Fields, sent)
	} else {
		log.WithFields(log.Fields{
			"Src":    m.Src,
			"Dst":    m.Dst,
			"Enc":    m.Enc,
			"Fields": s.Fields,
		}).Info("Sent message.")
		go updateMessage(m, respID, sconn.ID, "", s.Fields, sent)
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
		os.Exit(0)
	}()
}

// Binds to rabbitmq queue and listens for all numbers starting with supplied prefixes.
// This function calls handler when a connection is succesfully established
func bind() {
	var err error
	sconn = &smpp.Conn{}
	*sconn, err = c.GetConn(*group, *connid)
	log.WithFields(log.Fields{
		"connid":   *connid,
		"username": sconn.URL,
	}).Info("Connection id")
	log.WithFields(log.Fields{
		"URL":    sconn.URL,
		"User":   sconn.User,
		"Passwd": sconn.Passwd,
		"Conn":   sconn,
		"c":      c,
	}).Info("Dialing")
	s = &smpp.Sender{}
	s.Connect(sconn.URL, sconn.User, sconn.Passwd, receiver)
	defer s.Tx.Close()
	s.Fields = sconn.Fields
	log.Info("Waiting for smpp connection")
	select {
	case <-s.Connected:
	case <-time.After(time.Duration(time.Second * 5)):
		log.Error("Timed out waiting for smpp connection. Exiting.")
		os.Exit(2)
	}
	log.WithField("conn", sconn).Info("Binding")
	if err != nil {
		log.WithField("connid", connid).Fatalf("Couldn't get connection from settings. Check your settings and passed connection id parameter.")
	}
	r, err := queue.GetQueue("amqp://guest:guest@localhost:5672/", "smppworker-exchange", 1)
	if err != nil {
		os.Exit(2)
	}
	rate := time.Second / time.Duration(sconn.Size)
	tick = time.NewTicker(rate)
	defer tick.Stop()
	errTick = time.NewTicker(rate * 2)
	defer errTick.Stop()
	//bucket helps in keeping at max Size concurrent network requests at a time
	bucket = make(chan int, sconn.Size)
	defer close(bucket)
	log.WithField("Pfxs", sconn.Pfxs).Info("Binding to routing keys")
	err = r.Bind(*group, sconn.Pfxs, handler)
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
	var err error
	c = &smpp.Config{}
	*c, err = models.GetConfig()
	if err != nil {
		log.Fatal("Can't continue without settings. Exiting.")
	}
	bind()
}
