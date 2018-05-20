package main

import (
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/message"
	log "github.com/Sirupsen/logrus"
)

func updateMessage(m message.Message, respID, con, errMsg string, sent int64) {
	m.RespID = respID
	m.Connection = con
	m.Error = errMsg
	m.SentAt = sent
	m.Status = message.Sent
	if errMsg != "" {
		m.Status = message.Error
	}
	err := m.Update()
	if err != nil {
		log.WithError(err).Error("Couldn't update message.")
	}
}
