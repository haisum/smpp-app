package main

import (
	"time"

	"bitbucket.org/codefreak/hsmpp/pkg/influx"
	log "github.com/Sirupsen/logrus"
)

func writeInfluxBatch() {
	cl, err := influx.GetClient()
	if err != nil {
		log.WithError(err).Error("Couldn't get influxdb client")
	}
	for {
		<-time.After(time.Second * 5)
		if cl.TotalPoints() > 0 {
			log.WithField("count", cl.TotalPoints()).Info("Writing batch to influx")
			err = cl.Write()
			if err != nil {
				log.WithError(err).Error("Error in writing batch to influxdb")
			}
		}
	}
}
