package message

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/entities/message"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"github.com/pkg/errors"
	"gopkg.in/doug-martin/goqu.v3"
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

const (
	// queuedAt field is time at which message was put in queue
	queuedAt = "queuedAt"
	// userTextSearchLiteral is used to do full text query for user
	userTextSearchLiteral = "match(Username) against('?*' IN BOOLEAN MODE)"
	// msgTextSearchLiteral is used to do full text query for message
	msgTextSearchLiteral = "match(Msg) against('?')"
	// maxPerPageListing is maximum number of records per List query
	maxPerPageListing = 500000
	// defaultPerPageListing is default number of records per List query
	defaultPerPageListing = 100
)

var bulkInsertLock sync.Mutex

type store struct {
	db  *db.DB
	log logger.Logger
}

// NewStore returns a message store
func NewStore(db *db.DB, log logger.Logger) *store {
	return &store{db, log}
}

// Save saves a message in db
func (store *store) Save(m *message.Message) (int64, error) {
	result, err := store.db.From("Message").Insert(m).Exec()
	if err != nil {
		return 0, errors.Wrap(err, "couldn't insert message")
	}
	return result.LastInsertId()
}

// Get finds a message by primary key
func (store *store) Get(id int64) (*message.Message, error) {
	m := &message.Message{}
	found, err := store.db.From("Message").Where(goqu.I("id").Eq(id)).ScanStruct(m)
	if err != nil || !found {
		return m, errors.New("couldn't get message")
	}
	return m, nil
}

// SaveBulk saves a list of messages in Message table
func (store *store) SaveBulk(m []message.Message) ([]int64, error) {
	bulkInsertLock.Lock()
	defer bulkInsertLock.Unlock()
	var ids []int64
	result, err := store.db.From("Message").Insert(interface{}(m)).Exec()
	if err != nil {
		return ids, err
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != int64(len(m)) {
		return ids, errors.Wrap(err, "couldn't get affected rows or unexpected affected rows number")
	}
	err = store.db.From("Message").Select("id").Order(goqu.I("id").Desc()).Limit(uint(affected)).ScanVals(&ids)
	if err != nil {
		return ids, err
	}
	for k := affected - 1; k >= 0; k-- {
		m[k].ID = ids[k]
	}
	return ids, err
}

// Update updates an existing message in Message table
func (store *store) Update(m *message.Message) error {
	_, err := store.db.From("Message").Where(goqu.I("id").Eq(m.ID)).Update(m).Exec()
	return err
}

// List filters messages based on criteria
func (store *store) List(c *message.Criteria) ([]message.Message, error) {
	var m []message.Message
	var (
		from interface{}
		err  error
	)
	if c.OrderByKey == "" {
		c.OrderByKey = queuedAt
	}
	if c.From != "" && !c.DisableOrder {
		if c.OrderByKey == queuedAt || c.OrderByKey == "DeliveredAt" || c.OrderByKey == "SentAt" || c.OrderByKey == "ScheduledAt" {
			from, err = strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return m, fmt.Errorf("invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
	}
	ds := store.prepareQuery(c, from)
	if c.PerPage == 0 {
		c.PerPage = defaultPerPageListing
	}
	ds = ds.Limit(c.PerPage)
	q, _, _ := ds.ToSql()
	err = ds.ScanStructs(&m)
	return m, err
}

// Stats filters messages based on criteria and finds total number of messages in different statuses
func (store *store) Stats(c *message.Criteria) (*message.Stats, error) {
	m := &message.Stats{}
	var from interface{}
	if c.OrderByKey == "" {
		c.OrderByKey = queuedAt
	}
	if c.From != "" {
		if c.OrderByKey == queuedAt || c.OrderByKey == "DeliveredAt" || c.OrderByKey == "SentAt" {
			var err error
			from, err = strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return m, fmt.Errorf("invalid value for from: %d", from)
			}
		} else {
			from = c.From
		}
	}
	ds := store.prepareQuery(c, from)
	ds = ds.GroupBy("Status").Select(goqu.L("status, count(*) as total"))
	q, _, _ := ds.ToSql()
	stats := make(map[string]int64, 8)
	query, args, err := ds.ToSql()
	if err != nil {
		return m, err
	}
	rows, err := store.db.Db.Query(query, args...)
	if err != nil {
		return m, errors.Wrap(err, "couldn't run query: "+query)
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
		switch message.Status(k) {
		case message.Delivered:
			m.Delivered = v
		case message.Error:
			m.Error = v
		case message.Sent:
			m.Sent = v
		case message.Queued:
			m.Queued = v
		case message.NotDelivered:
			m.NotDelivered = v
		case message.Scheduled:
			m.Scheduled = v
		case message.Stopped:
			m.Stopped = v
		}
	}
	m.Total = m.Delivered + m.Error + m.Sent + m.Queued + m.NotDelivered + m.Scheduled + m.Stopped
	return m, err
}

func (store *store) prepareQuery(c *message.Criteria, from interface{}) *goqu.Dataset {
	t := store.db.From("Message")
	if c.OrderByKey == "" {
		c.OrderByKey = queuedAt
	}
	if c.Username != "" {
		if strings.HasPrefix(c.Username, "(re)") {
			c.Username = strings.Trim(c.Username, "(re)")
			t = t.Where(goqu.L(userTextSearchLiteral, c.Username))
		} else {
			t = t.Where(goqu.I("Username").Eq(c.Username))
		}
	}
	if c.Msg != "" {
		t = t.Where(goqu.L(msgTextSearchLiteral, c.Msg))
	}
	if c.QueuedAfter != 0 {
		t = t.Where(goqu.I("queuedAt").Gte(c.QueuedAfter))
	}
	if c.QueuedBefore != 0 {
		t = t.Where(goqu.I("queuedAt").Lte(c.QueuedBefore))
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

/*
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
		log.WithField("RespID", respID).Error("couldn't update delivery sm. No such response id found")
		return errors.New("couldn't update delivery sm. No such response id found")
	}
	return nil
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
		PerPage:    maxPerPageListing,
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
		PerPage:    maxPerPageListing,
	})
	if err != nil {
		log.WithError(err).Error("Couldn't load messages")
	}
	return m, err
}
*/
