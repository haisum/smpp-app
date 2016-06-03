package models

import (
	"fmt"
	"strconv"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

// Message represents a smpp message
type Message struct {
	ID              string `gorethink:"id,omitempty"`
	RespID          string
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
	CampaignID      string
	Status          MessageStatus
	Error           string
	SendBefore      string
	SendAfter       string
	ScheduledAt     int64
}

// MessageCriteria represents filters we can give to GetMessages method.
type MessageCriteria struct {
	RespID          string
	ConnectionGroup string
	Connection      string
	Username        string
	Enc             string
	Dst             string
	Src             string
	Msg             string
	QueuedBefore    int64
	QueuedAfter     int64
	SentBefore      int64
	SentAfter       int64
	DeliveredBefore int64
	DeliveredAfter  int64
	Total           int
	Priority        int
	CampaignID      string
	Status          MessageStatus
	Error           string
	ScheduledAfer   int64
	ScheduledBefore int64
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
	//MsgQueued shows that have been put in rabbitmq
	MsgQueued MessageStatus = "Queued"
	//MsgError shows that message was sent to operator but returned error
	MsgError MessageStatus = "Error"
	//MsgSent shows that message was accepted by operator for delivery
	MsgSent MessageStatus = "Sent"
	//MsgDelivered shows that message was delivered
	MsgDelivered MessageStatus = "Delivered"
	//MsgNotDelivered shows message was not delivered by operator
	MsgNotDelivered MessageStatus = "Not Delivered"
	//MsgScheduled shows message is schedueled to be delivered in future
	MsgScheduled MessageStatus = "Scheduled"
	//MsgStopped shows message was stopped by user intervention
	MsgStopped MessageStatus = "Stopped"
	// QueuedAt field is time at which message was put in rabbitmq queue
	QueuedAt string = "QueuedAt"
)

// MessageStats records number of messages in different statuses.
type MessageStats struct {
	Queued       int64
	Sent         int64
	Error        int64
	Delivered    int64
	NotDelivered int64
	Scheduled    int64
	Stopped      int64
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
	err = r.DB(db.DBName).Table("Message").Get(m.ID).Update(m).Exec(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("Message").Get(m.ID).Update(m).String(),
		}).Error("Error in updating message.")
		return err
	}
	return nil
}

// SaveDelivery updates an existing message in Message table and adds delivery status
func SaveDelivery(respID, src, status string) error {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return err
	}
	resp, err := r.DB(db.DBName).Table("Message").GetAllByIndex("RespID", respID).Filter(map[string]string{
		"Src": src,
	}).Update(map[string]interface{}{
		"Status":      status,
		"DeliveredAt": time.Now().UTC().Unix(),
	}).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("Message").GetAllByIndex("respID", respID).Filter(map[string]string{
				"Src": src,
			}).Update(map[string]interface{}{
				"Status":      status,
				"DeliveredAt": time.Now().UTC().Unix(),
			}),
		}).Error("Error in updating message.")
		return err
	}
	if resp.Replaced == 0 {
		log.WithField("RespID", respID).Error("Couldn't update delivery sm. No such response id found.")
		return fmt.Errorf("Couldn't update delivery sm. No such response id found.")
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
		if c.OrderByKey == QueuedAt || c.OrderByKey == "DeliveredAt" || c.OrderByKey == "SentAt" || c.OrderByKey == "ScheduledAt" {
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

// GetMessageStats filters messages based on criteria and finds total number of messages in different statuses
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
		case MsgScheduled:
			m.Scheduled, _ = strconv.ParseInt(v["reduction"], 10, 64)
		case MsgStopped:
			m.Stopped, _ = strconv.ParseInt(v["reduction"], 10, 64)
		}
	}
	m.Total = m.Delivered + m.Error + m.Sent + m.Queued + m.NotDelivered + m.Stopped + m.Scheduled
	return m, err
}

func prepareMsgTerm(c MessageCriteria, from interface{}) r.Term {
	t := r.DB(db.DBName).Table("Message")
	indexUsed := false
	if from != nil || c.QueuedAfter+c.QueuedBefore+c.DeliveredAfter+c.DeliveredBefore+c.SentAfter+c.SentBefore+c.ScheduledAfer+c.ScheduledBefore != 0 {
		indexUsed = true
	}
	if !indexUsed {
		if c.CampaignID != "" {
			t = t.GetAllByIndex("CampaignID", c.CampaignID)
			c.CampaignID = ""
		} else if c.Username != "" {
			t = t.GetAllByIndex("Username", c.Username)
			c.Username = ""
		} else if c.RespID != "" {
			t = t.GetAllByIndex("RespID", c.RespID)
			c.RespID = ""
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
		"SentAt": {
			"after":  c.SentAfter,
			"before": c.SentBefore,
		},
		"ScheduledAt": {
			"after":  c.ScheduledAfer,
			"before": c.ScheduledBefore,
		},
	}
	t = filterBetweenInt(betweenFields, t)
	strFields := map[string]string{
		"RespID":          c.RespID,
		"Connection":      c.Connection,
		"ConnectionGroup": c.ConnectionGroup,
		"Src":             c.Src,
		"Dst":             c.Dst,
		"Enc":             c.Enc,
		"Status":          string(c.Status),
		"CampaignID":      c.CampaignID,
		"Error":           c.Error,
		"Username":        c.Username,
	}
	t = filterEqStr(strFields, t)
	if c.Msg != "" {
		t = t.Filter(func(t r.Term) r.Term {
			return t.Field("Msg").Match(c.Msg)
		})
	}
	if c.Total > 0 {
		t = t.Filter(map[string]int{"Total": c.Total})
	}
	if c.Priority > 0 {
		t = t.Filter(map[string]int{"Priority": c.Priority})
	}
	if c.OrderByKey == "" {
		c.OrderByKey = QueuedAt
	}
	if !c.DisableOrder {
		t = orderBy(c.OrderByKey, c.OrderByDir, from, t, true)
	}
	return t
}
