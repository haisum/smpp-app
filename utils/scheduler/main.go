package main

import (
	"os"
	"time"

	_ "bitbucket.org/codefreak/hsmpp/pkg"
	"bitbucket.org/codefreak/hsmpp/pkg/db"
	"bitbucket.org/codefreak/hsmpp/pkg/license"
	"bitbucket.org/codefreak/hsmpp/pkg/queue"
	"bitbucket.org/codefreak/hsmpp/pkg/scheduler"
	log "github.com/Sirupsen/logrus"
	"github.com/spf13/viper"
)

func main() {
	go license.CheckExpiry()
	tick := time.NewTicker(time.Minute / 2)
	defer tick.Stop()
	log.Info("Connecting database.")
	conn, err := db.Connect(viper.GetString("MYSQL_HOST"), viper.GetInt("MYSQL_PORT"), viper.GetString("MYSQL_DBNAME"), viper.GetString("MYSQL_USER"), viper.GetString("MYSQL_PASSWORD"))
	if err != nil {
		log.WithError(err).Fatal("Couldn't setup database connection.")
	}
	defer conn.Db.Close()
	q, err := queue.ConnectRabbitMQ(viper.GetString("RABBITMQ_URL"), viper.GetString("RABBITMQ_EXCHANGE"), 1)
	if err != nil {
		log.Error("Couldn't connect to rabbitmq")
		os.Exit(2)
	}
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
