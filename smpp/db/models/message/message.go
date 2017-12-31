package message

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	goqu "gopkg.in/doug-martin/goqu.v3"
	"strconv"
	"strings"
	"sync"
	"time"
)

type deliverySM map[string]string

// Scan implements scanner interface for deliverySM
func (dsm *deliverySM) Scan(src interface{}) error {
	err := json.Unmarshal(src.([]byte), dsm)
	return err
}

// Value implements the driver.Valuer interface
func (dsm *deliverySM) Value() (driver.Value, error) {
	return json.Marshal(dsm)
}

// Message represents a smpp message inside db
type Message struct {
	ID              int64  `db:"id" goqu:"skipinsert"`
	RespID          string `db:"respid"`
	ConnectionGroup string `db:"connectiongroup"`
	Connection      string `db:"connection"`
	Total           int    `db:"total"`
	Username        string `db:"username"`
	Msg             string `db:"msg"`
	//RealMsg is unmasked version of msg, this shouldn't be exposed to user
	RealMsg     string `json:"-" db:"realmsg"`
	Enc         string `db:"enc"`
	Dst         string `db:"dst"`
	Src         string `db:"src"`
	Priority    int    `db:"priority"`
	QueuedAt    int64  `db:"queuedat"`
	SentAt      int64  `db:"sentat"`
	DeliveredAt int64  `db:"deliveredat"`
	CampaignID  int64  `db:"campaignid"`
	Campaign    string `db:"campaign"`
	Status      Status `db:"status"`
	Error       string `db:"error"`
	SendBefore  string `db:"sendbefore"`
	SendAfter   string `db:"sendafter"`
	ScheduledAt int64  `db:"scheduledat"`
	IsFlash     bool   `db:"isflash"`
}

// Criteria represents filters we can give to List method.
type Criteria struct {
	ID              int64
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
	CampaignID      int64
	Status          Status
	Error           string
	ScheduledAfter  int64
	ScheduledBefore int64
	OrderByKey      string
	OrderByDir      string
	From            string
	PerPage         uint
	DisableOrder    bool
	FetchMsg        bool
}

// Status represents current state of message in
// a lifecycle from submitted to getting delivered
type Status string

// Scan implements scanner interface for Status
func (st *Status) Scan(src interface{}) error {
	*st = Status(fmt.Sprintf("%s", src))
	return nil
}

const (
	//Queued shows that have been put in rabbitmq
	Queued Status = "Queued"
	//Error shows that message was sent to operator but returned error
	Error Status = "Error"
	//Sent shows that message was accepted by operator for delivery
	Sent Status = "Sent"
	//Delivered shows that message was delivered
	Delivered Status = "Delivered"
	//NotDelivered shows message was not delivered by operator
	NotDelivered Status = "Not Delivered"
	//Scheduled shows message is schedueled to be delivered in future
	Scheduled Status = "Scheduled"
	//Stopped shows message was stopped by user intervention
	Stopped Status = "Stopped"
	// QueuedAt field is time at which message was put in rabbitmq queue
	QueuedAt string = "QueuedAt"
)

// Stats records number of messages in different statuses.
type Stats struct {
	Queued       int64
	Sent         int64
	Error        int64
	Delivered    int64
	NotDelivered int64
	Scheduled    int64
	Stopped      int64
	Total        int64
}

var bulkInsertLock sync.Mutex

func (m *Message) Save() (int64, error) {
	con := db.Get()
	result, err := con.From("Message").Insert(m).Exec()
	if err != nil {
		log.WithError(err).Error("Couldn't insert message.")
		return 0, err
	}
	return result.LastInsertId()
}

// SaveBulk saves a list of messages in Message table
func SaveBulk(m []Message) ([]int64, error) {
	bulkInsertLock.Lock()
	defer bulkInsertLock.Unlock()
	con := db.Get()
	var ids []int64
	result, err := con.From("Message").Insert(interface{}(m)).Exec()
	if err != nil {
		log.WithError(err).Error("Couldn't insert message.")
		return ids, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != int64(len(m)) {
		log.WithError(err).WithField("affected", affected).Error("Couldn't get affected rows or unexpected affected rows number")
	}
	err = con.From("Message").Select("id").Order(goqu.I("id").Desc()).Limit(uint(affected)).ScanVals(&ids)
	if err != nil {
		log.WithError(err).WithField("affected", affected).Error("Couldn't load last inserted ids")
		return ids, err
	}
	for k := affected - 1; k >= 0; k-- {
		m[k].ID = ids[k]
	}
	return ids, err
}

// Update updates an existing message in Message table
func (m *Message) Update() error {
	_, err := db.Get().From("Message").Where(goqu.I("id").Eq(m.ID)).Update(m).Exec()
	return err
}

// SaveDelivery updates an existing message in Message table and adds delivery status
func SaveDelivery(respID, status string) error {
	res, err := db.Get().From("Message").Where(goqu.I("RespID").Eq(respID)).Update(goqu.Record{
		"Status":      status,
		"DeliveredAt": time.Now().UTC().Unix(),
	}).Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("Error in updating message.")
		return err
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		log.WithField("RespID", respID).Error("Couldn't update delivery sm. No such response id found.")
		return errors.New("Couldn't update delivery sm. No such response id found.")
	}
	return nil
}

//Get finds a message by primary key
func Get(id int64) (Message, error) {
	var m Message
	found, err := db.Get().From("Message").Where(goqu.I("id").Eq(id)).ScanStruct(&m)
	if err != nil || !found {
		log.WithFields(log.Fields{"error": err, "id": id}).Error("Couldn't get msg.")
		return m, errors.New("Couldn't get message.")
	}
	return m, nil
}

// StopPending marks stopped as true in all messages which are queued or scheduled in a campaign
func StopPending(campID int64) (int64, error) {
	res, err := db.Get().From("Message").Where(goqu.I("CampaignID").Eq(campID),
		goqu.Or(
			goqu.I("Status").Eq(Queued),
			goqu.I("Status").Eq(Scheduled),
		),
	).Update(goqu.Record{"Status": Stopped}).Exec()
	if err != nil {
		log.WithError(err).Error("Couldn't run query")
		return 0, err
	}
	affected, _ := res.RowsAffected()
	return affected, nil
}

// ListWithError returns all messages with status error in a campaign
func ListWithError(campID int64) ([]Message, error) {
	m, err := List(Criteria{
		CampaignID: campID,
		Status:     Error,
		PerPage:    500000,
	})
	if err != nil {
		log.WithError(err).Error("Couldn't load messages")
	}
	return m, err
}

// ListQueued returns all messages with status queued in a campaign
func ListQueued(campID int64) ([]Message, error) {
	m, err := List(Criteria{
		CampaignID: campID,
		Status:     Queued,
		PerPage:    500000,
	})
	if err != nil {
		log.WithError(err).Error("Couldn't load messages")
	}
	return m, err
}

// List filters messages based on criteria
func List(c Criteria) ([]Message, error) {
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
	ds := prepareQuery(c, from)
	if c.PerPage == 0 {
		c.PerPage = 100
	}
	ds.Limit(c.PerPage)
	q, _, _ := ds.ToSql()
	log.WithFields(log.Fields{"query": q, "crtieria": c}).Info("Running query.")
	err = ds.ScanStructs(&m)
	if err != nil {
		log.WithError(err).Error("Couldn't run query.")
	}
	return m, err
}

// GetStats filters messages based on criteria and finds total number of messages in different statuses
func GetStats(c Criteria) (Stats, error) {
	var m Stats
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
	ds := prepareQuery(c, from)
	ds = ds.GroupBy("Status").Select(goqu.L("status, count(*) as total"))
	q, _, _ := ds.ToSql()
	log.WithFields(log.Fields{"query": q, "crtieria": c}).Info("Running query.")
	stats := make(map[string]int64, 8)
	query, args, err := ds.ToSql()
	if err != nil {
		return m, err
	}
	rows, err := db.Get().Db.Query(query, args)
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
	rows.Close()
	for k, v := range stats {
		switch Status(k) {
		case Delivered:
			m.Delivered = v
		case Error:
			m.Error = v
		case Sent:
			m.Sent = v
		case Queued:
			m.Queued = v
		case NotDelivered:
			m.NotDelivered = v
		case Scheduled:
			m.Scheduled = v
		case Stopped:
			m.Stopped = v
		}
	}
	m.Total = m.Delivered + m.Error + m.Sent + m.Queued + m.NotDelivered + m.Scheduled + m.Stopped
	return m, err
}

func prepareQuery(c Criteria, from interface{}) *goqu.Dataset {
	t := db.Get().From("Message")
	if c.OrderByKey == "" {
		c.OrderByKey = QueuedAt
	}
	if c.Username != "" {
		if strings.HasPrefix(c.Username, "(re)") {
			c.Username = strings.Trim(c.Username, "(re)")
			t = t.Where(goqu.L("match(Msg) against('?*' IN BOOLEAN MODE)", c.Username))
		} else {
			t = t.Where(goqu.I("User").Eq(c.Username))
		}
	}
	if c.Msg != "" {
		t = t.Where(goqu.L("match(Msg) against('?')", c.Msg))
	}
	if c.QueuedAfter != 0 {
		t = t.Where(goqu.I("QueuedAt").Gte(c.QueuedAfter))
	}
	if c.QueuedBefore != 0 {
		t = t.Where(goqu.I("QueuedAt").Lte(c.QueuedBefore))
	}
	if c.DeliveredAfter != 0 {
		t = t.Where(goqu.I("DeliveredAt").Gte(c.DeliveredAfter))
	}
	if c.DeliveredBefore != 0 {
		t = t.Where(goqu.I("DeliveredAt").Lte(c.DeliveredBefore))
	}

	if c.SentAfter != 0 {
		t = t.Where(goqu.I("SentAt").Gte(c.SentAfter))
	}
	if c.SentBefore != 0 {
		t = t.Where(goqu.I("SentAt").Lte(c.SentBefore))
	}

	if c.ScheduledAfter != 0 {
		t = t.Where(goqu.I("ScheduledAt").Gte(c.ScheduledAfter))
	}
	if c.ScheduledBefore != 0 {
		t = t.Where(goqu.I("ScheduledAt").Lte(c.ScheduledBefore))
	}
	if c.RespID != "" {
		t = t.Where(goqu.I("RespID").Eq(c.RespID))
	}
	if c.Connection != "" {
		t = t.Where(goqu.I("Connection").Eq(c.Connection))
	}
	if c.ConnectionGroup != "" {
		t = t.Where(goqu.I("ConnectionGroup").Eq(c.ConnectionGroup))
	}
	if c.Src != "" {
		t = t.Where(goqu.I("Src").Eq(c.Src))
	}
	if c.Dst != "" {
		t = t.Where(goqu.I("Dst").Eq(c.Dst))
	}
	if c.Enc != "" {
		t = t.Where(goqu.I("Enc").Eq(c.Enc))
	}
	if c.Status != "" {
		t = t.Where(goqu.I("Status").Eq(c.Status))
	}
	if c.CampaignID != 0 {
		t = t.Where(goqu.I("CampaignID").Eq(c.CampaignID))
	}
	if c.Error != "" {
		t = t.Where(goqu.I("Error").Eq(c.Error))
	}
	if c.Total > 0 {
		t = t.Where(goqu.I("Total").Eq(c.Total))
	}
	if c.Priority > 0 {
		t = t.Where(goqu.I("Priority").Eq(c.Priority))
	}
	if !c.DisableOrder {
		orderDir := "DESC"
		if strings.ToUpper(c.OrderByDir) == "ASC" {
			orderDir = "ASC"
		}
		if from != nil {
			if orderDir == "ASC" {
				t = t.Where(goqu.I(c.OrderByKey).Gt(from))
			} else {
				t = t.Where(goqu.I(c.OrderByKey).Lt(from))
			}
		}
		orderExp := goqu.I(c.OrderByKey).Desc()
		if orderDir == "ASC" {
			orderExp = goqu.I(c.OrderByKey).Asc()
		}
		t = t.Order(orderExp)
	}
	return t
}

// Validate validates a message and returns error messages if any
func (m *Message) Validate() []string {
	var errs []string
	if m.Dst == "" {
		errs = append(errs, "Destination can't be empty.")
	}
	if m.Msg == "" {
		errs = append(errs, "Can't send empty message")
	}
	if m.Src == "" {
		errs = append(errs, "Source address can't be empty.")
	}
	if m.Enc != "ucs" && m.Enc != "latin" {
		errs = append(errs, "Encoding can either be latin or UCS")
	}
	if (m.SendAfter == "" && m.SendBefore != "") || (m.SendBefore == "" && m.SendAfter != "") {
		errs = append(errs, "Send before time and Send after time, both should be provided at a time.")
	}
	parts := strings.Split(m.SendAfter, ":")
	if m.SendAfter != "" {
		if len(parts) != 2 {
			errs = append(errs, "Send after must be of 24 hour format such as \"09:00\".")
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {

				errs = append(errs, "Send after must be of 24 hour format such as \"09:00\".")
			}
		}
	}
	parts = strings.Split(m.SendBefore, ":")
	if m.SendBefore != "" {
		if len(parts) != 2 {

			errs = append(errs, "Send before must be of 24 hour format such as \"09:00\".")
		} else {
			hour, errH := strconv.ParseInt(parts[0], 10, 32)
			minute, errM := strconv.ParseInt(parts[1], 10, 32)
			if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {

				errs = append(errs, "Send before must be of 24 hour format such as \"09:00\".")
			}
		}
	}
	return errs
}
