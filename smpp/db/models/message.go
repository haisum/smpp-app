package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/sphinx"
	"bitbucket.org/codefreak/hsmpp/smpp/db/utils"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

// Message represents a smpp message
type Message struct {
	ID              string `gorethink:"id,omitempty" db:"msgid"`
	SphinxID        int    `json:"-" gorethink:"-" db:"id"`
	RespID          string
	DeliverySM      map[string]string `gorethink:"DeliverySM,omitempty"`
	ConnectionGroup string
	Connection      string
	Fields          smpp.PduFields `gorethink:"Fields,omitempty"`
	Total           int
	Username        string `db:"user"`
	Msg             string `gorethink:"Msg,omitempty"`
	//RealMsg is unmasked version of msg, this shouldn't be exposed to user
	RealMsg     string `json:"-" gorethink:"RealMsg,omitempty"`
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
	IsFlash     bool
}

// MessageCriteria represents filters we can give to GetMessages method.
type MessageCriteria struct {
	ID              string
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
	ScheduledAfter  int64
	ScheduledBefore int64
	OrderByKey      string
	OrderByDir      string
	From            string
	PerPage         int
	DisableOrder    bool
	FetchMsg        bool
}

// MessageStatus represents current state of message in
// a lifecycle from submitted to getting delivered
type MessageStatus string

// Scan implements scannner interface for MessageStatus
func (st *MessageStatus) Scan(src interface{}) error {
	*st = MessageStatus(fmt.Sprintf("%s", src))
	return nil
}

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
	m.ID = id
	err = SaveInSphinx([]Message{*m})
	return id, err
}

func SaveInSphinx(m []Message) error {
	sp := sphinx.Get()
	if sp == nil {
		return fmt.Errorf("Sphinx db connection is not initialized yet")
	}
	if len(m) < 1 {
		return fmt.Errorf("No messages provided to save.")
	}
	query := `INSERT INTO Message(id, Msg, Username, ConnectionGroup, Connection, MsgID, RespID, Total, Enc, Dst, 
		Src, Priority, QueuedAt, SentAt, DeliveredAt, CampaignID, Status, Error, User, ScheduledAt, IsFlash) VALUES `
	var valuePart []string
	for _, v := range m {
		isFlash := 0
		if v.IsFlash {
			isFlash = 1
		}
		params := []interface{}{
			sphinx.Nextval("Message"), v.Msg, v.Username, v.ConnectionGroup,
			v.Connection, v.ID, v.RespID, v.Total, v.Enc, v.Dst, v.Src, v.Priority,
			v.QueuedAt, v.SentAt, v.DeliveredAt, v.CampaignID, string(v.Status), v.Error,
			v.Username, v.ScheduledAt, isFlash,
		}
		values := fmt.Sprintf(`(%d, '%s', '%s', '%s', '%s', '%s', '%s', %d, '%s', '%s', '%s',
			%d , %d, %d, %d, '%s', '%s', '%s', '%s', %d, %d)`, params...)
		valuePart = append(valuePart, values)
	}
	query = query + strings.Join(valuePart, ",")
	rs, err := sp.Exec(query)
	if err != nil {
		log.WithFields(log.Fields{"query": query, "error": err}).Error("Couldn't insert in db.")
		return err
	}
	affected, _ := rs.RowsAffected()
	if affected != int64(len(m)) {
		return fmt.Errorf("DB couldn't insert all of rows. Expected: %d, Inserted: %d", len(m), affected)
	}
	return nil
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
	for k, v := range resp.GeneratedKeys {
		m[k].ID = v
	}
	err = SaveInSphinx(m)
	return resp.GeneratedKeys, err
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
	err = UpdateInSphinx(*m)
	return err
}

func (m *Message) GetSphinxID() (int64, error) {
	query := fmt.Sprintf(`SELECT id FROM Message WHERE MsgID = '%s'`, m.ID)
	var id int64
	sp := sphinx.Get()
	err := sp.Get(&id, query)
	return id, err
}

func SaveDeliveryInSphinx(respID string) error {
	query := fmt.Sprintf(`SELECT msgID FROM Message WHERE RespID = '%s'`, respID)
	var id string
	sp := sphinx.Get()
	err := sp.Get(&id, query)
	if err != nil {
		return err
	}
	m, err := GetMessage(id)
	if err != nil {
		return err
	}
	return UpdateInSphinx(m)
}

func StopCampaignInSphinx(campaignID string) error {
	query := fmt.Sprintf(`SELECT msgID FROM Message WHERE campaignID = '%s'`, campaignID)
	var ids []string
	sp := sphinx.Get()
	err := sp.Select(&ids, query)
	if err != nil {
		return err
	}
	ms, err := GetMessages(MessageCriteria{
		CampaignID: campaignID,
		Status:     MsgStopped,
	})
	if err != nil {
		return err
	}
	for _, m := range ms {
		err = UpdateInSphinx(m)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateInSphinx(m Message) error {
	sp := sphinx.Get()
	query := `REPLACE INTO Message(id, Msg, Username, ConnectionGroup, Connection, MsgID, RespID, Total, Enc, Dst, 
		Src, Priority, QueuedAt, SentAt, DeliveredAt, CampaignID, Status, Error, User, ScheduledAt) VALUES `
	var valuePart []string
	spID, err := m.GetSphinxID()
	if err != nil {
		return err
	}
	params := []interface{}{
		spID, m.Msg, m.Username, m.ConnectionGroup,
		m.Connection, m.ID, m.RespID, m.Total, m.Enc, m.Dst, m.Src, m.Priority,
		m.QueuedAt, m.SentAt, m.DeliveredAt, m.CampaignID, string(m.Status), m.Error,
		m.Username, m.ScheduledAt,
	}
	values := fmt.Sprintf(`(%d, '%s', '%s', '%s', '%s', '%s', '%s', %d, '%s', '%s', '%s',
			%d , %d, %d, %d, '%s', '%s', '%s', '%s', %d)`, params...)
	valuePart = append(valuePart, values)
	query = query + strings.Join(valuePart, ",")
	_, err = sp.Exec(query)
	if err != nil {
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
	err = SaveDeliveryInSphinx(respID)
	if err != nil {
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
	err = StopCampaignInSphinx(campID)
	if err != nil {
		log.WithError(err).Error("Couldn't update records in sphinx")
		return 0, err
	}
	return resp.Replaced, nil
}

// GetErrorMessages returns all messages with status error in a campaign
func GetErrorMessages(campID string) ([]Message, error) {
	m, err := GetMessages(MessageCriteria{
		CampaignID: campID,
		Status:     MsgError,
		PerPage:    500000,
	})
	if err != nil {
		log.WithError(err).Error("Couldn't load messages")
	}
	return m, err
}

// GetQueuedMessages returns all messages with status queued in a campaign
func GetQueuedMessages(campID string) ([]Message, error) {
	m, err := GetMessages(MessageCriteria{
		CampaignID: campID,
		Status:     MsgQueued,
		PerPage:    500000,
	})
	if err != nil {
		log.WithError(err).Error("Couldn't load messages")
	}
	return m, err
}

// GetMessages filters messages based on criteria
func GetMessages(c MessageCriteria) ([]Message, error) {
	var m []Message
	var (
		from interface{}
		err  error
	)
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
	qb := prepareMsgTerm(c, from)
	if c.PerPage == 0 {
		c.PerPage = 100
	}
	qb.Limit(strconv.Itoa(c.PerPage))
	log.WithFields(log.Fields{"query": qb.GetQuery() + "  option max_matches=500000", "crtieria": c}).Info("Running query.")
	err = sphinx.Get().Select(&m, qb.GetQuery()+"  option max_matches=500000")
	if err != nil {
		log.WithError(err).Error("Couldn't run query.")
	}
	if c.FetchMsg && len(m) > 0 {
		msg, err := GetMessage(m[0].ID)
		if err != nil {
			log.WithError(err).Error("Something ain't right. We couldn't get sphinx msg from rethinkdb")
			return m, err
		}
		if msg.RealMsg == msg.Msg && c.CampaignID == msg.CampaignID {
			for k, _ := range m {
				m[k].Msg = msg.Msg
			}
		} else {
			for k, _ := range m {
				msg, err = GetMessage(m[k].ID)
				if err != nil {
					log.WithError(err).WithField("msg", m[k]).Error("Something ain't right. We couldn't get sphinx msg from rethinkdb")
					return m, err
				}
				m[k].Msg = msg.Msg
			}
		}
	}
	return m, err
}

// GetMessageStats filters messages based on criteria and finds total number of messages in different statuses
func GetMessageStats(c MessageCriteria) (MessageStats, error) {
	var m MessageStats
	var from interface{}
	if c.From != "" {
		if c.OrderByKey == "QueuedAt" || c.OrderByKey == "DeliveredAt" || c.OrderByKey == "SentAt" {
			from, err := strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return m, fmt.Errorf("Invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
	}
	qb := prepareMsgTerm(c, from)
	qb.Select("status, count(*) as total").From("Message")
	qb.GroupBy("Status")

	log.WithFields(log.Fields{"query": qb.GetQuery(), "crtieria": c}).Info("Running query.")
	stats := make(map[string]int64, 8)
	rows, err := sphinx.Get().Queryx(qb.GetQuery())
	if err != nil {
		log.WithError(err).Error("Couldn't run query.")
		return m, err
	}
	for rows.Next() {
		var (
			status string
			total  int64
		)
		rows.Scan(&status, &total)
		stats[status] = total
	}
	for k, v := range stats {
		switch MessageStatus(k) {
		case MsgDelivered:
			m.Delivered = v
		case MsgError:
			m.Error = v
		case MsgSent:
			m.Sent = v
		case MsgQueued:
			m.Queued = v
		case MsgNotDelivered:
			m.NotDelivered = v
		case MsgScheduled:
			m.Scheduled = v
		case MsgStopped:
			m.Stopped = v
		}
	}
	m.Total = m.Delivered + m.Error + m.Sent + m.Queued + m.NotDelivered + m.Scheduled + m.Stopped
	return m, err
}

func prepareMsgTerm(c MessageCriteria, from interface{}) utils.QueryBuilder {
	qb := utils.QueryBuilder{}
	qb.Select("*").From("Message")

	if c.OrderByKey == "" {
		c.OrderByKey = QueuedAt
	}
	if c.Username != "" {
		if strings.HasPrefix(c.Username, "(re)") {
			qb.WhereAnd("match('@Username " + c.Username + "')")
		} else {
			qb.WhereAnd("User = '" + c.Username + "'")
		}
	}
	if c.Msg != "" {
		qb.WhereAnd("match('@Msg " + c.Msg + "')")
	}
	if c.QueuedAfter != 0 {
		qb.WhereAnd("QueuedAt > " + strconv.FormatInt(c.QueuedAfter, 10))
	}
	if c.QueuedBefore != 0 {
		qb.WhereAnd("QueuedAt < " + strconv.FormatInt(c.QueuedBefore, 10))
	}

	if c.DeliveredAfter != 0 {
		qb.WhereAnd("DeliveredAt > " + strconv.FormatInt(c.DeliveredAfter, 10))
	}
	if c.DeliveredBefore != 0 {
		qb.WhereAnd("DeliveredAt < " + strconv.FormatInt(c.DeliveredBefore, 10))
	}

	if c.SentAfter != 0 {
		qb.WhereAnd("SentAt > " + strconv.FormatInt(c.SentAfter, 10))
	}
	if c.SentBefore != 0 {
		qb.WhereAnd("SentAt < " + strconv.FormatInt(c.SentBefore, 10))
	}

	if c.ScheduledAfter != 0 {
		qb.WhereAnd("ScheduledAt > " + strconv.FormatInt(c.ScheduledAfter, 10))
	}
	if c.ScheduledBefore != 0 {
		qb.WhereAnd("ScheduledAt < " + strconv.FormatInt(c.ScheduledBefore, 10))
	}
	if c.ID != "" {
		qb.WhereAnd("MsgID = '" + c.ID + "'")
	}
	if c.RespID != "" {
		qb.WhereAnd("RespID = '" + c.RespID + "'")
	}
	if c.Connection != "" {
		qb.WhereAnd("Connection = '" + c.Connection + "'")
	}
	if c.ConnectionGroup != "" {
		qb.WhereAnd("ConnectionGroup = '" + c.ConnectionGroup + "'")
	}
	if c.Src != "" {
		qb.WhereAnd("Src = '" + c.Src + "'")
	}
	if c.Dst != "" {
		qb.WhereAnd("Dst = '" + c.Dst + "'")
	}
	if c.Enc != "" {
		qb.WhereAnd("Enc = '" + c.Enc + "'")
	}
	if c.Status != "" {
		qb.WhereAnd("Status = '" + string(c.Status) + "'")
	}
	if c.CampaignID != "" {
		qb.WhereAnd("CampaignID = '" + c.CampaignID + "'")
	}
	if c.Error != "" {
		qb.WhereAnd("Error = '" + string(c.Error) + "'")
	}
	if c.Total > 0 {
		qb.WhereAnd("Total = " + strconv.Itoa(c.Total))
	}
	if c.Priority > 0 {
		qb.WhereAnd("Priority = " + strconv.Itoa(c.Priority))
	}
	if !c.DisableOrder {
		orderDir := "DESC"
		if strings.ToUpper(c.OrderByDir) == "ASC" {
			orderDir = "ASC"
		}
		if from != nil {
			if orderDir == "ASC" {
				qb.WhereAnd(c.OrderByKey + " > '" + fmt.Sprintf("%s", from) + "'")
			} else {
				qb.WhereAnd(c.OrderByKey + " < '" + fmt.Sprintf("%s", from) + "'")
			}
		}
		qb.OrderBy(c.OrderByKey + " " + orderDir)
	}
	return qb
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
