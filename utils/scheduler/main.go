package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
)

func main() {
	tick := time.NewTicker(time.Minute * 2)
	defer tick.Stop()
	q, err := queue.GetQueue("amqp://guest:guest@localhost:5672/", "smppworker-exchange", 1)
	if err != nil {
		log.Error("Couldn't connect to rabbitmq")
		os.Exit(2)
	}
	log.Info("Waiting for scheduled messages.")
	for {
		after := time.Now().UTC()
		before := time.Now().Add(time.Minute * 2).UTC()
		log.WithFields(log.Fields{
			"ScheduledAfer":   after.String(),
			"ScheduledBefore": before.String(),
		}).Info("Looking for messages")
		ms, err := models.GetMessages(models.MessageCriteria{
			ScheduledAfer:   after.Unix(),
			ScheduledBefore: before.Unix(),
		})
		if err != nil {
			log.Error("Couldn't get messages.")
			// code 2 makes supervisor stop trying to reload this process
			os.Exit(2)
		}
		log.WithFields(log.Fields{
			"Total": len(ms),
		}).Info("Messages found.")
		for _, m := range ms {
			config, err := models.GetConfig()
			if err != nil {
				log.Error("Couldn't get config")
			}
			keys := config.GetKeys(m.ConnectionGroup)
			var noKey string
			group, err := config.GetGroup(m.ConnectionGroup)
			if err != nil {
				log.Error("Couldn't find group for message.")
			}
			noKey = group.DefaultPfx
			key := matchKey(keys, m.Dst, noKey)
			qItem := queue.Item{
				MsgID: m.ID,
				Total: m.Total,
			}
			respJSON, _ := qItem.ToJSON()
			err = q.Publish(fmt.Sprintf("%s-%s", m.ConnectionGroup, key), respJSON, queue.Priority(m.Priority))
			if err != nil {
				log.Error("Couldn't publish message.")
			}
			m.Status = models.MsgQueued
			m.QueuedAt = time.Now().UTC().Unix()
			m.Update()
		}
		<-tick.C
	}
}

// Given a list of strings and a string,
// this function returns a list item if large string starts with list item.
// string in parameter noKey is returned if no matches could be found
func matchKey(keys []string, str string, noKey string) string {
	for _, key := range keys {
		if strings.HasPrefix(str, key) {
			return key
		}
	}
	return noKey
}
