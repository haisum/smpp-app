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
	"sync/atomic"
	"syscall"
	"time"
)

var c smpp.Config
var conn smpp.Conn

func handler(deliveries <-chan amqp.Delivery, done chan error) {
	var s smpp.Sender
	s.Connect(conn.Url, conn.User, conn.Passwd)
	log.Info("Waiting for smpp connection")
	<-s.Connected
	var count int32
	count = 0
	for d := range deliveries {
		cur := atomic.LoadInt32(&count)
		if cur >= conn.Size {
			log.Info("Waiting one second before proceeding")
			time.Sleep(time.Second * time.Duration(conn.Time))
			log.Info("Resuming messages")
			atomic.SwapInt32(&count, 0)
		}
		go send(&s, d, &count)
	}
	log.Printf("handle: deliveries channel closed")
	done <- nil
}

func send(s *smpp.Sender, d amqp.Delivery, count *int32) {
	var i queue.QueueItem
	err := i.FromJSON(d.Body)
	if err != nil {
		log.Printf("Failed in parsing json %s. Error: %s", string(d.Body[:]), err)
		d.Nack(false, true)
		return
	}
	charLimit := 160
	if i.Enc == "UCS" {
		charLimit = 60
	}
	res := float64(float64(len(i.Msg)) / float64(charLimit))
	total := math.Ceil(res)
	atomic.AddInt32(count, int32(total))
	_, err = s.Send(i.Src, i.Dst, i.Enc, i.Msg)
	if err != nil {
		log.Printf("Couldn't send message from %s to %s", i.Src, i.Dst)
		if err != smppstatus.ErrNotConnected {
			d.Reject(false)
		} else {
			d.Nack(false, true)
		}
	} else {
		log.WithFields(log.Fields{
			"Src": i.Src,
			"Dst": i.Dst,
			"Enc": i.Enc,
		}).Info("Sent message.")
		d.Ack(false)
	}
}

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

func main() {
	connid := flag.String("cid", "", "Pass smpp connection id of connection this worker is going to send sms to.")
	flag.Parse()
	if *connid == "" {
		flag.Usage()
		os.Exit(1)
	}

	err := c.LoadFile("settings.json")
	if err != nil {
		log.Fatal("Can't continue without settings. Exiting.")
	}

	conn, err = c.GetConn(*connid)
	if err != nil {
		log.WithField("connid", *connid).Fatalf("Couldn't get connection from settings. Check your settings and passed connection id parameter.")
	}

	var r queue.Rabbit
	err = r.Init(c.AmqpUrl, "smppworker-exchange", 1)
	if err != nil {
		os.Exit(1)
	}
	log.WithField("Pfxs", conn.Pfxs).Info("Binding to routing keys")
	err = r.Bind(conn.Pfxs, handler)
	if err != nil {
		os.Exit(1)
	}

	//Listen for termination signals from OS
	go gracefulShutdown(&r)

	forever := make(<-chan int)
	<-forever
}
