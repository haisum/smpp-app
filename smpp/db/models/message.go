package models

import (
	"fmt"
	"strconv"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

// Message represents a smpp message
type Message struct {
	Id              string `gorethink:"id,omitempty"`
	RespId          string
	DeliverySM      map[string]string
	ConnectionGroup string
	Connection      string
	Fields          smpp.PduFields
	Total           int
	Username        string
	Msg             string
	Enc             string
	Dst             string
	Src             string
	Priority        int
	QueuedAt        int64
	SentAt          int64
	DeliveredAt     int64
	CampaignId      string
	Status          MessageStatus
	Error           string
}

// MessageCriteria represents filters we can give to GetMessages method.
type MessageCriteria struct {
	Id              string
	RespId          string
	ConnectionGroup string
	Connection      string
	Username        string
	Enc             string
	Dst             string
	Src             string
	QueuedBefore    int64
	QueuedAfter     int64
	SentBefore      int64
	SentAfter       int64
	DeliveredBefore int64
	DeliveredAfter  int64
	CampaignId      string
	Status          MessageStatus
	Error           string
	OrderByKey      string
	OrderByDir      string
	From            string
	PerPage         int
	DisableOrder    bool
}

// MessageStatus represents current state of message in
// a lifecycle from submitted to getting delivered
type MessageStatus string

const (
	MsgQueued       MessageStatus = "Queued"
	MsgError        MessageStatus = "Error"
	MsgSent         MessageStatus = "Sent"
	MsgDelivered    MessageStatus = "Delivered"
	MsgNotDelivered MessageStatus = "Not Delivered"
)

// MessageStats records number of messages in different statuses.
type MessageStats struct {
	Queued       int64
	Sent         int64
	Error        int64
	Delivered    int64
	NotDelivered int64
	Total        int64
}

// Save saves a message struct in Message table
func (m *Message) Save() (string, error) {
	var id string
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return id, err
	}
	resp, err := r.DB(db.DBName).Table("Message").Insert(m).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("Message").Insert(m).String(),
		}).Error("Error in inserting message.")
		return id, err
	}
	id = resp.GeneratedKeys[0]
	return id, nil
}

// Update updates an existing message in Message table
func (m *Message) Update() error {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return err
	}
	err = r.DB(db.DBName).Table("Message").Get(m.Id).Update(m).Exec(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("Message").Get(m.Id).Update(m).String(),
		}).Error("Error in updating message.")
		return err
	}
	return nil
}

//GetMessage finds a message by primary key
func GetMessage(id string) (Message, error) {
	var m Message
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return m, err
	}
	cur, err := r.DB(db.DBName).Table("Message").Get(id).Run(s)
	defer cur.Close()
	if err != nil {
		log.WithError(err).Error("Couldn't get message.")
		return m, err
	}
	cur.One(&m)
	defer cur.Close()
	return m, nil
}

// GetMessages filters messages based on criteria
func GetMessages(c MessageCriteria) ([]Message, error) {
	var m []Message
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return m, err
	}
	var from interface{}
	if c.From != "" && !c.DisableOrder {
		if c.OrderByKey == "QueuedAt" || c.OrderByKey == "DeliveredAt" || c.OrderByKey == "SentAt" {
			from, err = strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return m, fmt.Errorf("Invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
	}
	t := prepareMsgTerm(c, from)
	if c.PerPage == 0 {
		c.PerPage = 100
	}
	t = t.Limit(c.PerPage)
	log.WithFields(log.Fields{"query": t.String(), "crtieria": c}).Info("Running query.")
	cur, err := t.Run(s)
	if err != nil {
		log.WithError(err).Error("Couldn't run query.")
		return m, err
	}
	defer cur.Close()
	err = cur.All(&m)
	if err != nil {
		log.WithError(err).Error("Couldn't load messages.")
	}
	return m, err
}

// GetMessagesStats filters messages based on criteria and finds total number of messages in different statuses
func GetMessageStats(c MessageCriteria) (MessageStats, error) {
	var m MessageStats
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return m, err
	}
	var from interface{}
	if c.From != "" {
		if c.OrderByKey == "QueuedAt" || c.OrderByKey == "DeliveredAt" || c.OrderByKey == "SentAt" {
			from, err = strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return m, fmt.Errorf("Invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
	}
	t := prepareMsgTerm(c, from)
	t = t.Group("Status").Count()

	log.WithFields(log.Fields{"query": t.String(), "crtieria": c}).Info("Running query.")
	cur, err := t.Run(s)
	if err != nil {
		log.WithError(err).Error("Couldn't run query.")
		return m, err
	}
	defer cur.Close()
	stats := make([]map[string]string, 5)
	err = cur.All(&stats)
	if err != nil {
		log.WithError(err).Error("Couldn't load messages.")
	}
	for _, v := range stats {
		switch MessageStatus(v["group"]) {
		case MsgDelivered:
			m.Delivered, _ = strconv.ParseInt(v["reduction"], 10, 64)
		case MsgError:
			m.Error, _ = strconv.ParseInt(v["reduction"], 10, 64)
		case MsgSent:
			m.Sent, _ = strconv.ParseInt(v["reduction"], 10, 64)
		case MsgQueued:
			m.Queued, _ = strconv.ParseInt(v["reduction"], 10, 64)
		case MsgNotDelivered:
			m.NotDelivered, _ = strconv.ParseInt(v["reduction"], 10, 64)
		}
	}
	m.Total = m.Delivered + m.Error + m.Sent + m.Queued + m.NotDelivered
	return m, err
}

func prepareMsgTerm(c MessageCriteria, from interface{}) r.Term {

	t := r.DB(db.DBName).Table("Message")

	// note to self: keep between before Eq filters.
	betweenFields := map[string]map[string]int64{
		"QueuedAt": {
			"after":  c.QueuedAfter,
			"before": c.QueuedBefore,
		},
		"DeliveredAt": {
			"after":  c.DeliveredAfter,
			"before": c.DeliveredBefore,
		},
		"SentAt": {
			"after":  c.SentAfter,
			"before": c.SentBefore,
		},
	}
	t = filterBetweenInt(betweenFields, t)
	strFields := map[string]string{
		"id":              c.Id,
		"RespId":          c.RespId,
		"Connection":      c.Connection,
		"ConnectionGroup": c.ConnectionGroup,
		"Src":             c.Src,
		"Dst":             c.Dst,
		"Enc":             c.Enc,
		"Status":          string(c.Status),
		"CampaignId":      c.CampaignId,
		"Error":           c.Error,
		"Username":        c.Username,
	}
	t = filterEqStr(strFields, t)

	if c.OrderByKey == "" {
		c.OrderByKey = "QueuedAt"
	}
	if !c.DisableOrder {
		t = orderBy(c.OrderByKey, c.OrderByDir, from, t)
	}
	return t
}
