package main

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/sphinx"
	"bitbucket.org/codefreak/hsmpp/smpp/license"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"bitbucket.org/codefreak/hsmpp/smpp/scheduler"
	log "github.com/Sirupsen/logrus"
	"os"
	"time"
)

func main() {
	go license.CheckExpiry()
	tick := time.NewTicker(time.Minute / 2)
	defer tick.Stop()
	log.Info("Connecting database.")
	conn, err := db.Connect("127.0.0.1", "3306", "hsmppdb", "root", "")
	if err != nil {
		log.WithError(err).Fatal("Couldn't setup database connection.")
	}
	defer conn.Db.Close()
	q, err := queue.ConnectRabbitMQ("amqp://guest:guest@localhost:5672/", "smppworker-exchange", 1)
	if err != nil {
		log.Error("Couldn't connect to rabbitmq")
		os.Exit(2)
	}
	spconn, err := sphinx.Connect("127.0.0.1", "9306")
	if err != nil {
		log.WithError(err).Fatalf("Error in connecting to sphinx.")
	}
	defer spconn.Db.Close()
	log.Info("Waiting for scheduled messages.")
	for {
		err = scheduler.ProcessMessages(q)
		if err != nil {
			// code 2 makes supervisor stop trying to reload this process
			os.Exit(2)
		}
		<-tick.C
	}
}
