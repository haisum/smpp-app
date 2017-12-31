package campaign

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/numfile"
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/doug-martin/goqu.v3"
	"strconv"
	"strings"
)

// Campaign represents a message campaign
type Campaign struct {
	ID          int64  `db:"id" goqu:"skipinsert"`
	Description string `db:"description"`
	Src         string `db:"src"`
	Msg         string `db:"msg"`
	Priority    int    `db:"priority"`
	FileID      int64  `db:"numfileid"`
	Username    string `db:"username"`
	SendBefore  string `db:"sendbefore"`
	SendAfter   string `db:"sendafter"`
	ScheduledAt int64  `db:"scheduledat"`
	SubmittedAt int64  `db:"submittedat"`
	Total       int    `db:"total"`
	Errors      stringutils.StringList
}

const (
	// submittedAt is time at which campaign was put in system
	submittedAt string = "submittedAt"
)

// Criteria represents filters we can give to Select method.
type Criteria struct {
	ID              int64
	Username        string
	FileID          int64
	SubmittedAfter  int64
	SubmittedBefore int64
	OrderByKey      string
	OrderByDir      string
	From            string
	PerPage         uint
}

// Report is report of campaign performance
type Report struct {
	ID            int64
	Total         int
	MsgSize       int
	TotalMsgs     int
	FirstQueued   int64
	LastSent      int64
	TotalTime     int
	Throughput    string
	PerConnection string
	Connections   []GroupCount
}

type Progress map[string]int

// Save saves a campaign in db
func (c *Campaign) Save() (int64, error) {
	if c.FileID != 0 {
		f, _ := numfile.List(numfile.Criteria{
			ID: c.FileID,
		})
		if len(f) != 1 {
			return 0, fmt.Errorf("Couldn't find file.")
		}
	}
	resp, err := db.Get().From("Campaign").Insert(c).Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("Error in adding campaign in db.")
		return 0, err
	}
	return resp.LastInsertId()
}

// GetProgress returns count for a campaign in progress
func (c *Campaign) GetProgress() (Progress, error) {
	cp := Progress{
		"Total":        0,
		"Queued":       0,
		"Delivered":    0,
		"NotDelivered": 0,
		"Sent":         0,
		"Error":        0,
		"Scheduled":    0,
		"Stopped":      0,
		"Pending":      0,
	}
	var vals []struct {
		Status string `db:"status"`
		Total  int    `db:"total"`
	}
	err := db.Get().ScanStructs(&vals, "SELECT status, count(*) as total from Message where campaignid = ?  group by status", c.ID)
	if err != nil {
		log.WithError(err).Error("Couldn't get campaign stats")
		return cp, err
	}
	for _, val := range vals {
		cp[val.Status] = val.Total
	}
	camps, err := List(Criteria{ID: c.ID})
	if err != nil || len(camps) != 1 {
		log.Error("Couldn't load campaign")
		return cp, err
	}

	totalInDB := 0
	for _, v := range cp {
		totalInDB = totalInDB + v
	}
	cp["Total"] = camps[0].Total
	cp["Pending"] = camps[0].Total - totalInDB
	return cp, err
}

// GetReport returns Report struct filled with stats from campaign with given id
func (c *Campaign) GetReport() (Report, error) {
	cr := Report{
		ID: c.ID,
	}
	// get total in campaign
	_, err := db.Get().ScanVal(&cr.Total, "SELECT count(*) as Total from Message where campaignID = ?", c.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("Error executing total msgs query")
		return cr, fmt.Errorf("Could't run query.")
	}
	// select message size in campaign
	_, err = db.Get().ScanVal(&cr.MsgSize, "SELECT Total as MsgSize from Message where campaignID = ?", c.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("Error executing MsgSize query")
		return cr, fmt.Errorf("Could't run query.")
	}
	// select min sentat in campaign
	_, err = db.Get().ScanVal(&cr.FirstQueued, "SELECT Min(SentAt) as FirstQueued from Message where campaignID = ? AND SentAt > 0", c.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("Error executing Min(SentAt) query")
	}
	// select max sentat in campaign
	_, err = db.Get().ScanVal(&cr.LastSent, "SELECT Max(SentAt) as LastSent from Message where campaignID=?", c.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("Error executing Max(SentAt) query")
	}
	// Select connection wise
	err = db.Get().ScanStructs(&cr.Connections, "SELECT Connection as Name, count(*) as Count from Message where campaignID= ? group by Connection", c.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
		}).Error("Error executing Connection wise query")
		return cr, fmt.Errorf("Could't run query.")
	}
	cr.TotalMsgs = cr.Total * cr.MsgSize
	if cr.LastSent == 0 {
		cr.TotalTime = 0
		cr.Throughput = "0"
		cr.PerConnection = "0"
		return cr, nil
	}
	cr.TotalTime = int(cr.LastSent - cr.FirstQueued)
	if cr.TotalTime <= 0 {
		cr.Throughput = strconv.FormatInt(int64(cr.TotalMsgs), 10)
	} else {
		cr.Throughput = strconv.FormatFloat(1.0/(float64(cr.TotalTime)/float64(cr.TotalMsgs)), 'f', 2, 64)
	}
	tp, _ := strconv.ParseFloat(cr.Throughput, 64)
	cr.PerConnection = strconv.FormatFloat(tp/float64(len(cr.Connections)), 'f', 2, 64)
	return cr, nil
}

// List fetches list of campaigns based on criteria
func List(c Criteria) ([]Campaign, error) {
	var (
		camps []Campaign
	)
	t := db.Get().From("Campaign")

	var from interface{}
	if c.From != "" {
		if c.OrderByKey == submittedAt || c.OrderByKey == "ScheduledAt" {
			from, err := strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return camps, fmt.Errorf("Invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
	}
	if c.OrderByKey == "" {
		c.OrderByKey = submittedAt
	}
	if c.SubmittedAfter > 0 {
		t = t.Where(goqu.I("submittedat").Gte(c.SubmittedAfter))
	}
	if c.SubmittedBefore > 0 {
		t = t.Where(goqu.I("submittedat").Lte(c.SubmittedBefore))
	}
	if c.ID > 0 {
		t = t.Where(goqu.I("id").Eq(c.ID))
	}
	if c.Username != "" {
		t = t.Where(goqu.I("username").Eq(c.Username))
	}
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
	if c.PerPage == 0 {
		c.PerPage = 100
	}
	t = t.Limit(c.PerPage)
	queryStr, _, _ := t.ToSql()
	log.WithFields(log.Fields{"query": queryStr, "crtieria": c}).Info("Running query.")
	err := t.ScanStructs(&camps)
	if err != nil {
		log.WithError(err).Error("Couldn't run query.")
	}
	return camps, err
}
