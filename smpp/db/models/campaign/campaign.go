package campaigns

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/sphinx"
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
	Total int
	Errors []string
}

const (
	//SubmittedAt is time at which campaign was put in system
	SubmittedAt string = "SubmittedAt"
)

// Criteria represents filters we can give to Select method.
type Criteria struct {
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

// Report is report of campaign performance
type Report struct {
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

type Progress map[string]int

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

//GetProgress returns count for a campaign in progress
func GetProgress(id string) (Progress, error) {
	cp := Progress{
		"Total" : 0,
		"Queued"  : 0,
		"Delivered" : 0,
		"NotDelivered" : 0,
		"Sent" : 0,
		"Error" : 0,
		"Scheduled" : 0,
		"Stopped" : 0,
		"Pending" : 0,
	}
	rows,err := sphinx.Get().Queryx("SELECT status, count(*) as total from Message where campaignid = ?  group by status", id)
	if err != nil {
		log.WithError(err).Error("Couldn't get campaign stats")
		return cp, err
	}
	defer rows.Close()
	for rows.Next(){
		var vals struct {
			Status string
			Total int
		}
		err = rows.StructScan(&vals)
		if err != nil {
			return cp, err
		}
		cp[vals.Status] = vals.Total
	}
	camps, err := Select(Criteria{ID : id})
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

// GetReport returns CampaignReport struct filled with stats from campaign with given id
func GetReport(id string) (Report, error) {
	cr := Report{
		ID: id,
	}
	c, err := GetCampaigns(Criteria{
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

// Filter fetches list of campaigns based on criteria
func Filter(camps *[]Campaign, c Criteria) (error) {
	var (
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
