package main

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/queue"
	smppstatus "github.com/fiorix/go-smpp/smpp"
	"github.com/streadway/amqp"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var c smpp.Config
var conn smpp.Conn

func handler(deliveries <-chan amqp.Delivery, done chan error) {

	var s smpp.Sender
	s.Connect(conn.Url, conn.User, conn.Passwd)
	count := 0
	for d := range deliveries {
		if count == conn.Size {
			<-time.Tick(time.Second * time.Duration(conn.Time))
			count = 0
		}
		go send(&s, d)
		count++
	}
	log.Printf("handle: deliveries channel closed")
	done <- nil
}

func send(s *smpp.Sender, d amqp.Delivery) {
	var i queue.QueueItem
	err := i.FromJSON(d.Body)
	if err != nil {
		log.Printf("Failed in parsing json %s. Error: %s", string(d.Body[:]), err)
		d.Nack(false, true)
		return
	}
	_, err = s.Send(i.Src, i.Dst, i.Msg)
	if err != nil {
		log.Printf("Couldn't send message from %s to %s", i.Src, i.Dst)
		if err != smppstatus.ErrNotConnected {
			d.Reject(false)
		} else {
			d.Nack(false, true)
		}
	}
	m, _ := time.Now().MarshalText()
	log.Printf("%s\n", m)
	d.Ack(false)
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
	err := c.LoadFile("settings.json")
	if err != nil {
		log.Fatal("Can't continue without settings. Exiting.")
	}
	connid := ""

	conn, err = c.GetConn(connid)
	if err != nil {
		log.Fatalf("Couldn't get connection %s from settings.", connid)
	}

	var r queue.Rabbit
	err = r.Init(c.AmqpUrl, "smppworker-exchange", 5)
	if err != nil {
		os.Exit(1)
	}
	err = r.Bind("smppworker-queue", conn.Pfxs, handler)
	if err != nil {
		os.Exit(1)
	}

	//Listen for termination signals from OS
	go gracefulShutdown(&r)

	forever := make(<-chan int)
	<-forever
}
