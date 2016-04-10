package main

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/queue"
	"flag"
	log "github.com/Sirupsen/logrus"
	smppstatus "github.com/fiorix/go-smpp/smpp"
	"github.com/streadway/amqp"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	c      *smpp.Config
	s      *smpp.Sender
	sconn  *smpp.Conn
	connid = flag.String("cid", "", "Pass smpp connection id of connection this worker is going to send sms to.")
	cmutex smpp.CountMutex
)

// Handler is called by rabbitmq library after a queue has been bound/
// deliveries channel gets data when a new job is to be consumed by worker
// This function should wait for done channel before terminating so that all
// pending jobs should be finished and rabbitmq should be notified about disconnect
func handler(deliveries <-chan amqp.Delivery, done chan error) {
	for d := range deliveries {
		// multiple threads may race to read/write this, so we need atomic operation
		cmutex.Lock()
		cur := cmutex.Count
		if cur >= sconn.Size {
			log.Info("Waiting one second before proceeding")
			time.Sleep(time.Second * time.Duration(sconn.Time))
			log.Info("Resuming messages")
			cmutex.Count = 0
		}
		cmutex.Unlock()
		go send(d)
	}
	log.Printf("handle: deliveries channel closed")
	done <- nil
}

// This is called per job and as a separate go routing
// This function is responsible for acknowledging the job completion to rabbitmq
// This function also increments count by ceil of number of characters divided by number of characters per message.
// When count reaches a certain number defined per connection, worker waits for time t defined in configuration before resuming operations.
func send(d amqp.Delivery) {
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
	charLimit := smpp.MaxLatinChars
	if i.Enc == "UCS" {
		charLimit = smpp.MaxUCSChars
	}
	res := float64(float64(len(i.Msg)) / float64(charLimit))
	total := math.Ceil(res)
	cmutex.Lock()
	cmutex.Count = cmutex.Count + int32(total)
	cmutex.Unlock()
	_, err = s.Send(i.Src, i.Dst, i.Enc, i.Msg)
	if err != nil {
		log.WithFields(log.Fields{
			"Src":    i.Src,
			"Dst":    i.Dst,
			"err":    err,
			"Enc":    i.Enc,
			"Fields": s.Fields,
		}).Error("Couldn't send message.")
		if err != smppstatus.ErrNotConnected {
			d.Reject(false)
		} else {
			d.Nack(false, true)
		}
	} else {
		log.WithFields(log.Fields{
			"Src":    i.Src,
			"Dst":    i.Dst,
			"Enc":    i.Enc,
			"Fields": s.Fields,
		}).Info("Sent message.")
		d.Ack(false)
	}
}

// When SIGTERM or SIGINT is received, this routine will make sure we shutdown our queues and finish in progress jobs
func gracefulShutdown(r *queue.Rabbit) {
	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)
	signal.Notify(s, syscall.SIGTERM)
	go func() {
		<-s
		log.Print("Sutting down gracefully.")
		r.Close()
		os.Exit(0)
	}()
}

// Binds to rabbitmq queue and listens for all numbers starting with supplied prefixes.
// This function calls handler when a connection is succesfully established
func bind() {
	var err error
	sconn = &smpp.Conn{}
	*sconn, err = c.GetConn(*connid)
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
	s.Connect(sconn.URL, sconn.User, sconn.Passwd)
	s.Fields = sconn.Fields
	log.Info("Waiting for smpp connection")
	<-s.Connected
	cmutex = smpp.CountMutex{}
	log.WithField("conn", sconn).Info("Binding")
	if err != nil {
		log.WithField("connid", connid).Fatalf("Couldn't get connection from settings. Check your settings and passed connection id parameter.")
	}

	r := queue.Rabbit{
		Conn: &amqp.Connection{},
		Ch:   &amqp.Channel{},
		Dial: amqp.Dial,
	}
	err = r.Init(c.AmqpURL, "smppworker-exchange", 1)
	if err != nil {
		os.Exit(1)
	}
	log.WithField("Pfxs", sconn.Pfxs).Info("Binding to routing keys")
	err = r.Bind(sconn.Pfxs, handler)
	if err != nil {
		os.Exit(1)
	}
	//Listen for termination signals from OS
	go gracefulShutdown(&r)

}

func main() {
	flag.Parse()
	if *connid == "" {
		flag.Usage()
		os.Exit(1)
	}
	c = &smpp.Config{}
	err := c.LoadFile("settings.json")
	if err != nil {
		log.Fatal("Can't continue without settings. Exiting.")
	}
	bind()

	forever := make(<-chan int)
	<-forever
}
