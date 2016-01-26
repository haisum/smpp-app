package main

import (
	smppstatus "github.com/fiorix/go-smpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/queue"
	"github.com/streadway/amqp"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func handler(deliveries <-chan amqp.Delivery, done chan error) {

	var s smpp.Sender
	s.Connect("192.168.0.105:2775", "smppclient1", "password")
	for d := range deliveries {
		go send(&s, d)
	}
	log.Printf("handle: deliveries channel closed")
	done <- nil
}

func send(s *smpp.Sender, d amqp.Delivery) {
	var i smpp.QueueItem
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
	var r queue.Rabbit
	err := r.Init("amqp://guest:guest@localhost:5672/", "TestExchange")
	if err != nil {
		os.Exit(1)
	}
	routingKeys := []string{"firstroutingkey"}
	err = r.BindQueues(routingKeys, handler)
	if err != nil {
		os.Exit(1)
	}

	//Listen for termination signals from OS
	go gracefulShutdown(&r)

	forever := make(<-chan int)
	<-forever
}
