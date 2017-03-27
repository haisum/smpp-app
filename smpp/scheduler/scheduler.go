package scheduler

import (
	"time"
	log "github.com/Sirupsen/logrus"
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
	"fmt"
	"strings"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/messages"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/config"
)

// Given a list of strings and a string,
// this function returns a list item if large string starts with list item.
// string in parameter noKey is returned if no matches could be found
func MatchKey(keys []string, str string, noKey string) string {
	for _, key := range keys {
		if strings.HasPrefix(str, key) {
			return key
		}
	}
	return noKey
}


func GetMessagesBetween(after, before time.Time) ([]messages.Message, error) {
	//fetch 10k at  a time
	ms, err := messages.Filter(messages.Criteria{
		ScheduledAfter:  after.Unix(),
		ScheduledBefore: before.Unix(),
		Status:          "Scheduled",
		PerPage:         500000,
	})
	return ms, err
}

func GetKey(m messages.Message) (string, error) {
	config, err := config.Get()
	if err != nil {
		log.Error("Couldn't get config")
	}
	keys := config.GetKeys(m.ConnectionGroup)
	var noKey string
	group, err := config.GetGroup(m.ConnectionGroup)
	if err != nil {
		log.Error("Couldn't find group for message.")
		return "", err
	}
	noKey = group.DefaultPfx
	key := MatchKey(keys, m.Dst, noKey)
	return key, nil
}

func ProcessMessages(q queue.MQ) error {
	now := time.Now().UTC()
	after := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, now.Location())
	before := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 59, 999999999, now.Location())
	log.WithFields(log.Fields{
		"ScheduledAfer":   after.String(),
		"ScheduledBefore": before.String(),
	}).Info("Looking for messages")
	ms := []messages.Message{messages.Message{}}
	var err error
	for len(ms) != 0 {
		ms, err = GetMessagesBetween(after, before)
		if err != nil {
			log.WithError(err).Error("Couldn't get messages.")
			return err
		}
		log.WithFields(log.Fields{
			"Total": len(ms),
		}).Info("Messages found.")
		for _, m := range ms {
			qItem := queue.Item{
				MsgID: m.ID,
				Total: m.Total,
			}
			respJSON, _ := qItem.ToJSON()
			key, _ := GetKey(m)
			err = q.Publish(fmt.Sprintf("%s-%s", m.ConnectionGroup, key), respJSON, queue.Priority(m.Priority))
			if err != nil {
				log.WithError(err).Error("Couldn't publish message.")
				return err
			}
			m.Status = messages.Queued
			m.QueuedAt = time.Now().UTC().Unix()
			m.Update()
		}
	}
	return nil
}
