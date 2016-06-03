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

// GetCampaigns fetches list of campaigns based on criteria
func GetCampaigns(c CampaignCriteria) ([]Campaign, error) {
	var camps []Campaign
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
	t = filterBetweenInt(betweenFields, t)
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
	t = filterEqStr(strFields, t)

	if c.OrderByKey == "" {
		c.OrderByKey = SubmittedAt
	}
	t = orderBy(c.OrderByKey, c.OrderByDir, from, t, true)
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
