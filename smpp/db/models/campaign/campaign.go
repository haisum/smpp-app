package campaign

import (
	"fmt"
	"strconv"
	"strings"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/numfile"
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"gopkg.in/doug-martin/goqu.v3"
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
	submittedAt string = "submittedat"
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
	Connections   []groupCount
}

// Progress shows status of messages in a campaign
// Current map of progress is like this:
// "Total":        int,
// "Queued":       int,
// "Delivered":    int,
// "NotDelivered": int,
// "Sent":         int,
// "Error":        int,
// "Scheduled":    int,
// "Stopped":      int,
// "Pending":      int,
type Progress map[string]int

// Save saves a campaign in db
func (c *Campaign) Save() (int64, error) {
	if c.FileID != 0 {
		f, err := numfile.List(numfile.Criteria{
			ID: c.FileID,
		})
		if len(f) != 1 || err != nil {
			return 0, fmt.Errorf("couldn't find file")
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
	err := db.Get().From("Message").Select(goqu.L("status, count(*) as total")).Where(goqu.I("campaignid").Eq(c.ID)).GroupBy("status").ScanStructs(&vals)
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
	ds := db.Get().From("Message").Where(goqu.I("CampaignID").Eq(c.ID))
	var errs []string
	// get total in campaign
	_, err := ds.Select(goqu.L("count(*) as Total")).ScanVal(&cr.Total)
	errs = appendNotNil(errs, errors.WithMessage(err, "total query"))
	// select message size in campaign
	_, err = ds.Select(goqu.L("Total as MsgSize")).Limit(1).ScanVal(&cr.MsgSize)
	errs = appendNotNil(errs, errors.WithMessage(err, "msgSize query"))
	// select min sentat in campaign
	_, err = ds.Select(goqu.L("Min(SentAt) as FirstQueued")).Where(goqu.I("sentat").Gt(0)).ScanVal(&cr.FirstQueued)
	errs = appendNotNil(errs, errors.WithMessage(err, "min(SentAt) query"))
	// select max sentat in campaign
	_, err = ds.Select(goqu.L("Max(SentAt) as LastSent")).ScanVal(&cr.LastSent)
	errs = appendNotNil(errs, errors.WithMessage(err, "max(SentAt) query"))
	// Select connection wise
	err = ds.Select(goqu.L("Connection as name, count(*) as count")).GroupBy("Connection").ScanStructs(&cr.Connections)
	errs = appendNotNil(errs, errors.WithMessage(err, "connection query"))
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "\n"))
		return cr, err
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

	if c.OrderByKey == "" {
		c.OrderByKey = submittedAt
	}
	var from interface{}
	if c.From != "" {
		if c.OrderByKey == submittedAt || c.OrderByKey == "scheduledat" {
			var err error
			from, err = strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return camps, fmt.Errorf("invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
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

func appendNotNil(errs []string, err error) []string {
	if err != nil {
		errs = append(errs, err.Error())
	}
	return errs
}
