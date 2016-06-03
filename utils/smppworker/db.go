package main

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models"
	log "github.com/Sirupsen/logrus"
)

func updateMessage(m models.Message, respID, con, errMsg string, fields smpp.PduFields, sent int64) {
	m.RespID = respID
	m.Connection = con
	m.Error = errMsg
	m.Fields = fields
	m.SentAt = sent
	m.Status = models.MsgSent
	if errMsg != "" {
		m.Status = models.MsgError
	}
	err := m.Update()
	if err != nil {
		log.WithError(err).Error("Couldn't update message.")
	}
}
