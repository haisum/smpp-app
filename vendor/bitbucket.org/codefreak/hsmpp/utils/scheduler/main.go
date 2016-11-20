package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	"bitbucket.org/codefreak/hsmpp/smpp/license"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
)

func main() {
	go license.CheckExpiry()
	tick := time.NewTicker(time.Minute / 2)
	defer tick.Stop()
	q, err := queue.GetQueue("amqp://guest:guest@localhost:5672/", "smppworker-exchange", 1)
	if err != nil {
		log.Error("Couldn't connect to rabbitmq")
		os.Exit(2)
	}
	log.Info("Waiting for scheduled messages.")
	for {
		now := time.Now().UTC()
		after := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
		before := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 59, 999999999, now.Location())
		log.WithFields(log.Fields{
			"ScheduledAfer":   after.String(),
			"ScheduledBefore": before.String(),
		}).Info("Looking for messages")
		ms := []models.Message{models.Message{}}
		var err error
		for len(ms) != 0 {
			//fetch 10k at  a time
			ms, err = models.GetMessages(models.MessageCriteria{
				ScheduledAfer:   after.Unix(),
				ScheduledBefore: before.Unix(),
				Status:          "Scheduled",
				PerPage:         10000,
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
