package message

import (
	"fmt"
	"strconv"
	"strings"
)

type MessageStorer interface{
	Save(m *Message) (int64, error)
	SaveBulk(m []Message) ([]int64, error)
	Update(m *Message) error
	Get(id int64) (Message, error)
	List(c Criteria) ([]Message, error)
	GetStats(c Criteria) (Stats, error)
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
	// RealMsg is unmasked version of msg, this shouldn't be exposed to user
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
}

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


// Status represents current state of message in
// a lifecycle from submitted to getting delivered
type Status string

// Scan implements scanner interface for Status
func (st *Status) Scan(src interface{}) error {
	*st = Status(fmt.Sprintf("%s", src))
	return nil
}

const (
	// Queued shows that have been put in rabbitmq
	Queued Status = "Queued"
	// Error shows that message was sent to operator but returned error
	Error Status = "Error"
	// Sent shows that message was accepted by operator for delivery
	Sent Status = "Sent"
	// Delivered shows that message was delivered
	Delivered Status = "Delivered"
	// NotDelivered shows message was not delivered by operator
	NotDelivered Status = "Not Delivered"
	// Scheduled shows message is schedueled to be delivered in future
	Scheduled Status = "Scheduled"
	// Stopped shows message was stopped by user intervention
	Stopped Status = "Stopped"
)