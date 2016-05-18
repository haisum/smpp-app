package models

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"fmt"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
	"strconv"
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
	SubmittedAt     int64
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
	SubmittedBefore int64
	SubmittedAfter  int64
	DeliveredBefore int64
	DeliveredAfter  int64
	CampaignId      string
	Status          MessageStatus
	Error           string
	OrderByKey      string
	OrderByDir      string
	From            string
	PerPage         int
}

// MessageStatus represents current state of message in
// a lifecycle from submitted to getting delivered
type MessageStatus string

const (
	MsgSubmitted    MessageStatus = "Submitted"
	MsgError        MessageStatus = "Error"
	MsgSent         MessageStatus = "Sent"
	MsgDelivered    MessageStatus = "Delivered"
	MsgNotDelivered MessageStatus = "Not Delivered"
)

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
	t := r.DB(db.DBName).Table("Message")

	var from interface{}
	if c.From != "" {
		if c.OrderByKey == "QueuedAt" || c.OrderByKey == "DeliveredAt" || c.OrderByKey == "SubmittedAt" {
			from, err = strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return m, fmt.Errorf("Invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
	}
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
		"SubmittedAt": {
			"after":  c.SubmittedAfter,
			"before": c.SubmittedBefore,
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
	if c.PerPage == 0 {
		c.PerPage = 100
	}

	if c.OrderByKey == "" {
		c.OrderByKey = "QueuedAt"
	}
	t = orderBy(c.OrderByKey, c.OrderByDir, from, t)
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
