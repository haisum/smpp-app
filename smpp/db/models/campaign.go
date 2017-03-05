package models

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/sphinx"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
	"strconv"
)

// Campaign represents a message campaign
type Campaign struct {
	ID            string `gorethink:"id,omitempty" db:"campaignid"`
	Description   string
	Src           string
	Msg           string
	Enc           string
	FileName      string
	Priority      int
	FileLocalName string
	FileID        string
	UserID        string
	Username      string
	SendBefore    string
	SendAfter     string
	ScheduledAt   int64
	SubmittedAt   int64
}

const (
	//SubmittedAt is time at which campaign was put in system
	SubmittedAt string = "SubmittedAt"
)

// CampaignCriteria represents filters we can give to GetCampaigns method.
type CampaignCriteria struct {
	ID              string
	Username        string
	FileID          string
	SubmittedAfter  int64
	SubmittedBefore int64
	OrderByKey      string
	OrderByDir      string
	From            string
	PerPage         int
}

// CampaignReport is report of campaign performance
type CampaignReport struct {
	ID            string
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

// Save saves a campaign in db
func (c *Campaign) Save() (string, error) {
	var id string
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return id, err
	}
	if c.FileID != "" {
		f, _ := GetNumFiles(NumFileCriteria{
			ID: c.FileID,
		})
		if len(f) != 1 {
			return id, fmt.Errorf("Couldn't find file.")
		}
		c.FileLocalName = f[0].LocalName
		c.FileName = f[0].Name
	}
	resp, err := r.DB(db.DBName).Table("Campaign").Insert(c).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("Campaign").Insert(c).String(),
		}).Error("Error in adding campaign in db.")
		return id, err
	}
	id = resp.GeneratedKeys[0]
	return id, nil
}

// GetReport returns CampaignReport struct filled with stats from campaign with given id
func GetReport(id string) (CampaignReport, error) {
	cr := CampaignReport{
		ID: id,
	}
	c, err := GetCampaigns(CampaignCriteria{
		ID: id,
	})
	if err != nil {
		return cr, fmt.Errorf("Couldn't fetch report from db. %s.", err)
	}
	if len(c) == 0 {
		return cr, fmt.Errorf("No campaign with id %s could be found.", id)
	}
	// get total in campaign
	err = sphinx.Get().Get(&cr, "SELECT count(*) as Total from Message where campaignID='"+id+"'")
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": "SELECT count(*) as Total from Message where campaignID='" + id + "'",
		}).Error("Error executing total msgs query")
		return cr, fmt.Errorf("Could't run query.")
	}
	//select message size in campaign
	err = sphinx.Get().Get(&cr, "SELECT Total as MsgSize from Message where campaignID='"+id+"'")
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": "SELECT Total as MsgSize from Message where campaignID='" + id + "'",
		}).Error("Error executing MsgSize query")
		return cr, fmt.Errorf("Could't run query.")
	}
	//select min sentat in campaign
	err = sphinx.Get().Get(&cr, "SELECT Min(SentAt) as FirstQueued from Message where campaignID='"+id+"' AND SentAt > 0")
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": "SELECT Min(SentAt) as FirstQueued from Message where campaignID='" + id + "'",
		}).Error("Error executing Min(SentAt) query")
	}
	//select max sentat in campaign
	err = sphinx.Get().Get(&cr, "SELECT Max(SentAt) as LastSent from Message where campaignID='"+id+"'")
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": "SELECT Max(SentAt) as LastSent from Message where campaignID='" + id + "'",
		}).Error("Error executing Max(SentAt) query")
	}
	//Select connection wise
	err = sphinx.Get().Select(&cr.Connections, "SELECT Connection as Name, count(*) as Count from Message where campaignID='"+id+"' group by Connection")
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": "SELECT Connection as Name, count(*) as Count from Message where campaignID='" + id + "' group by Connection",
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

// GetCampaigns fetches list of campaigns based on criteria
func GetCampaigns(c CampaignCriteria) ([]Campaign, error) {
	var (
		camps      []Campaign
		indexUsed  bool
		filterUsed bool
	)
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return camps, err
	}
	t := r.DB(db.DBName).Table("Campaign")

	var from interface{}
	if c.From != "" {
		if c.OrderByKey == SubmittedAt || c.OrderByKey == "ScheduledAt" {
			from, err = strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return camps, fmt.Errorf("Invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
	}
	if from != nil || c.SubmittedAfter+c.SubmittedBefore != 0 {
		indexUsed = true
	}
	if c.OrderByKey == "" {
		c.OrderByKey = SubmittedAt
	}
	if !indexUsed {
		if c.Username != "" {
			if c.OrderByKey == SubmittedAt && !indexUsed {
				t = t.Between([]interface{}{c.Username, r.MinVal}, []interface{}{c.Username, r.MaxVal}, r.BetweenOpts{
					Index: "Username_SubmittedAt",
				})
				c.OrderByKey = "Username_SubmittedAt"
			} else {
				t = t.GetAllByIndex("Username", c.Username)
				indexUsed = true
			}
			c.Username = ""
		}
	}
	// keep between before Eq
	betweenFields := map[string]map[string]int64{
		"SubmittedAt": {
			"after":  c.SubmittedAfter,
			"before": c.SubmittedBefore,
		},
	}
	t, filterUsed = filterBetweenInt(betweenFields, t)
	strFields := map[string]string{
		"id":         c.ID,
		"Username":   c.Username,
	}
	var filtered bool
	t, filtered = filterEqStr(strFields, t)
	filterUsed = filtered || filterUsed
	t = orderBy(c.OrderByKey, c.OrderByDir, from, t, indexUsed, filterUsed)
	if c.PerPage == 0 {
		c.PerPage = 100
	}
	t = t.Limit(c.PerPage)
	log.WithFields(log.Fields{"query": t.String(), "crtieria": c}).Info("Running query.")
	cur, err := t.Run(s)
	if err != nil {
		log.WithError(err).Error("Couldn't run query.")
		return camps, err
	}
	defer cur.Close()
	err = cur.All(&camps)
	if err != nil {
		log.WithError(err).Error("Couldn't load files.")
	}
	return camps, err
}
