package models

import (
	"fmt"
	"strconv"
	"strings"
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
	//RealMsg is unmasked version of msg, this shouldn't be exposed to user
	RealMsg     string `json:"-"`
	Enc         string
	Dst         string
	Src         string
	Priority    int
	QueuedAt    int64
	SentAt      int64
	DeliveredAt int64
	CampaignID  string
	Campaign    string
	Status      MessageStatus
	Error       string
	SendBefore  string
	SendAfter   string
	ScheduledAt int64
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

// SaveBulk saves a list of message structs in Message table
func SaveBulk(m []Message) ([]string, error) {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return nil, err
	}
	resp, err := r.DB(db.DBName).Table("Message").Insert(m).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("Message").Insert(m).String(),
		}).Error("Error in inserting message.")
		return nil, err
	}
	return resp.GeneratedKeys, nil
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
func SaveDelivery(respID, status string) error {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return err
	}
	resp, err := r.DB(db.DBName).Table("Message").GetAllByIndex("RespID", respID).Update(map[string]interface{}{
		"Status":      status,
		"DeliveredAt": time.Now().UTC().Unix(),
	}).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("Message").GetAllByIndex("respID", respID).Update(map[string]interface{}{
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

// StopPendingMessages marks stopped as true in all messages which are queued or scheduled in a campaign
func StopPendingMessages(campID string) (int, error) {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return 0, err
	}
	resp, err := r.DB(db.DBName).Table("Message").GetAllByIndex("CampaignID", campID).Filter(r.Row.Field("Status").Eq(MsgQueued).Or(r.Row.Field("Status").Eq(MsgScheduled))).Update(map[string]MessageStatus{"Status": MsgStopped}).RunWrite(s)
	if err != nil {
		log.WithError(err).Error("Couldn't run query")
		return 0, err
	}
	return resp.Replaced, nil
}

// GetErrorMessages returns all messages with status error in a campaign
func GetErrorMessages(campID string) ([]Message, error) {
	s, err := db.GetSession()
	var m []Message
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return m, err
	}
	cur, err := r.DB(db.DBName).Table("Message").GetAllByIndex("CampaignID", campID).Filter(r.Row.Field("Status").Eq(MsgError)).Run(s)
	if err != nil {
		log.WithError(err).Error("Couldn't run query")
		return m, err
	}
	defer cur.Close()
	err = cur.All(&m)
	if err != nil {
		log.WithError(err).Error("Couldn't load messages")
	}
	return m, err
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
	filterUsed := false
	if from != nil || c.QueuedAfter+c.QueuedBefore+c.DeliveredAfter+c.DeliveredBefore+c.SentAfter+c.SentBefore+c.ScheduledAfer+c.ScheduledBefore != 0 {
		indexUsed = true
	}
	if c.OrderByKey == "" {
		c.OrderByKey = QueuedAt
	}
	if !indexUsed {
		if c.CampaignID != "" {
			t = t.GetAllByIndex("CampaignID", c.CampaignID)
			c.CampaignID = ""
			indexUsed = true
		} else if c.Username != "" && !strings.HasPrefix(c.Username, "(re)") {
			if c.OrderByKey == QueuedAt && !indexUsed {
				t = t.Between([]interface{}{c.Username, r.MinVal}, []interface{}{c.Username, r.MaxVal}, r.BetweenOpts{
					Index: "Username_QueuedAt",
				})
				c.OrderByKey = "Username_QueuedAt"
			} else {
				t = t.GetAllByIndex("Username", c.Username)
				indexUsed = true
			}
			c.Username = ""
		} else if c.RespID != "" {
			t = t.GetAllByIndex("RespID", c.RespID)
			c.RespID = ""
			indexUsed = true
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
	var filtered bool
	t, filtered = filterBetweenInt(betweenFields, t)
	filterUsed = filterUsed || filtered
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
	}
	if !strings.HasPrefix(c.Username, "(re)") {
		strFields["Username"] = c.Username
	} else {
		t = t.Filter(func(t r.Term) r.Term {
			return t.Field("Username").Match(strings.TrimPrefix(c.Username, "(re)"))
		})
		filterUsed = true
	}
	t, filtered = filterEqStr(strFields, t)
	filterUsed = filtered || filterUsed
	if c.Msg != "" {
		t = t.Filter(func(t r.Term) r.Term {
			return t.Field("Msg").Match(c.Msg)
		})
		filterUsed = true
	}
	if c.Total > 0 {
		t = t.Filter(map[string]int{"Total": c.Total})
		filterUsed = true
	}
	if c.Priority > 0 {
		t = t.Filter(map[string]int{"Priority": c.Priority})
		filterUsed = true
	}
	if !c.DisableOrder {
		t = orderBy(c.OrderByKey, c.OrderByDir, from, t, indexUsed, filterUsed)
	}
	return t
}

// Validate validates a message and returns error messages if any
func (m *Message) Validate() []string {
	var errors []string
	if m.Dst == "" {
		errors = append(errors, "Destination can't be empty.")
	}
	if m.Msg == "" {
		errors = append(errors, "Can't send empty message")
	}
	if m.Src == "" {
		errors = append(errors, "Source address can't be empty.")
	}
	if m.Enc != "ucs" && m.Enc != "latin" {
		errors = append(errors, "Encoding can either be latin or UCS")
	}
	if (m.SendAfter == "" && m.SendBefore != "") || (m.SendBefore == "" && m.SendAfter != "") {
		errors = append(errors, "Send before time and Send after time, both should be provided at a time.")
	}
	parts := strings.Split(m.SendAfter, ":")
	if m.SendAfter != "" {
		if len(parts) != 2 {
			errors = append(errors, "Send after must be of 24 hour format such as \"09:00\".")
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {

				errors = append(errors, "Send after must be of 24 hour format such as \"09:00\".")
			}
		}
	}
	parts = strings.Split(m.SendBefore, ":")
	if m.SendBefore != "" {
		if len(parts) != 2 {

			errors = append(errors, "Send before must be of 24 hour format such as \"09:00\".")
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {

				errors = append(errors, "Send before must be of 24 hour format such as \"09:00\".")
			}
		}
	}
	return errors
}
