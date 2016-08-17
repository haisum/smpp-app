package models

import (
	"fmt"
	"strconv"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

// Campaign represents a message campaign
type Campaign struct {
	ID            string `gorethink:"id,omitempty"`
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
	FileName        string
	Src             string
	Msg             string
	Enc             string
	UserID          string
	SubmittedAfter  int64
	SubmittedBefore int64
	ScheduledAfter  int64
	ScheduledBefore int64
	SendBefore      string
	SendAfter       string
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
	f, _ := GetNumFiles(NumFileCriteria{
		ID: c.FileID,
	})
	if len(f) != 1 {
		return id, fmt.Errorf("Couldn't find file.")
	}
	c.FileLocalName = f[0].LocalName
	c.FileName = f[0].Name
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
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return cr, err
	}

	cur, err := r.DB("hsmppdb").Table("Message").GetAllByIndex("CampaignID", id).Count().Run(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB("hsmppdb").Table("Message").GetAllByIndex("CampaignID", id).Count().String(),
		}).Error("Error executing message count query")
		return cr, fmt.Errorf("Could't run query.")
	}
	err = cur.One(&cr.Total)
	if err != nil {
		log.WithError(err).Error("Couldn't load in cr.Total")
		return cr, fmt.Errorf("Couldn't load in cr.Total")
	}
	cur.Close()
	cur, err = r.DB("hsmppdb").Table("Message").GetAllByIndex("CampaignID", id).Limit(1).Field("Total").Run(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB("hsmppdb").Table("Message").GetAllByIndex("CampaignID", id).Limit(1).Field("Total").String(),
		}).Error("Error executing message size query")
		return cr, fmt.Errorf("Could't run query.")
	}
	err = cur.One(&cr.MsgSize)
	if err != nil {
		log.WithError(err).Error("Couldn't load in cr.MsgSize")
		return cr, fmt.Errorf("Couldn't load in cr.MsgSize")
	}
	cur.Close()
	cur, err = r.DB("hsmppdb").Table("Message").GetAllByIndex("CampaignID", id).Min("SentAt").Field("SentAt").Run(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB("hsmppdb").Table("Message").GetAllByIndex("CampaignID", id).Min("SentAt").Field("SentAt").String(),
		}).Error("Error executing min queued at query")
		return cr, fmt.Errorf("Could't run query.")
	}
	err = cur.One(&cr.FirstQueued)
	if err != nil {
		log.WithError(err).Error("Couldn't load in cr.FirstQueued")
		return cr, fmt.Errorf("Couldn't load in cr.FirstQueued")
	}
	cur.Close()
	cur, err = r.DB("hsmppdb").Table("Message").GetAllByIndex("CampaignID", id).Max("SentAt").Field("SentAt").Run(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB("hsmppdb").Table("Message").GetAllByIndex("CampaignID", id).Max("SentAt").Field("SentAt").String(),
		}).Error("Error executing sent at query")
		return cr, fmt.Errorf("Could't run query.")
	}
	err = cur.One(&cr.LastSent)
	if err != nil {
		log.WithError(err).Error("Couldn't load in cr.LastSent")
		return cr, fmt.Errorf("Couldn't load in cr.LastSent")
	}
	cur.Close()
	cur, err = r.DB("hsmppdb").Table("Message").GetAllByIndex("CampaignID", id).Group("Connection").Count().Run(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB("hsmppdb").Table("Message").GetAllByIndex("CampaignID", id).Group("Connection").Count().String(),
		}).Error("Error executing connection count query")
		return cr, fmt.Errorf("Could't run query.")
	}
	err = cur.All(&cr.Connections)
	if err != nil {
		log.WithError(err).Error("Couldn't load in cr.Connections")
		return cr, fmt.Errorf("Couldn't load in cr.Connections")
	}
	cur.Close()
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
	if from != nil || c.ScheduledAfter+c.ScheduledBefore+c.SubmittedAfter+c.SubmittedBefore != 0 {
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
		"ScheduledAt": {
			"after":  c.ScheduledAfter,
			"before": c.ScheduledBefore,
		},
	}
	t, filterUsed = filterBetweenInt(betweenFields, t)
	strFields := map[string]string{
		"id":         c.ID,
		"Username":   c.Username,
		"UserID":     c.UserID,
		"FileName":   c.FileName,
		"Src":        c.Src,
		"Msg":        c.Msg,
		"Enc":        c.Enc,
		"SendBefore": c.SendBefore,
		"SendAfter":  c.SendAfter,
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
