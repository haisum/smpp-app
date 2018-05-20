package campaign

import (
	"context"

	"bitbucket.org/codefreak/hsmpp/pkg/stringutils"
)

// Store is interface for campaign store implementations
type Store interface {
	Save(campaign *Campaign) (int64, error)
	List(criteria *Criteria) ([]Campaign, error)
	Progress(ID int64) (Progress, error)
	Report(ID int64) (Report, error)
}

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
	Context     context.Context `db:"-" json:"-"`
}

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

// groupCount is data structure to save results of .group(field).count() queries.
type groupCount struct {
	Name  string `db:"name"`
	Count int64  `db:"count"`
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
