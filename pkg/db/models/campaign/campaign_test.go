package campaign

import (
	"regexp"
	"testing"

	"bitbucket.org/codefreak/hsmpp/pkg/db"
	"bitbucket.org/codefreak/hsmpp/pkg/db/models/campaign/file"
	"github.com/pkg/errors"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gopkg.in/doug-martin/goqu.v3"
	"gopkg.in/stretchr/testify.v1/assert"
)

func TestCampaign_Save(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	c := Campaign{
		Username:    "user1",
		FileID:      2,
		ScheduledAt: 3223,
	}
	numExpected, _, _ := db.Get().From("NumFile").Select(&file.NumFile{}).Where(goqu.I("ID").Eq(2), goqu.I("deleted").Is(false)).Order(goqu.I("SubmittedAt").Desc()).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(numExpected)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	expected, _, _ := db.Get().From("Campaign").ToInsertSql(&c)
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(11, 1))
	num, err := c.Save()
	assert := assert.New(t)
	assert.Nil(err)
	assert.Equal(int64(11), num)
	// error case:
	mock.ExpectQuery(regexp.QuoteMeta(numExpected)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	expected, _, _ = db.Get().From("Campaign").ToInsertSql(&c)
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnError(errors.New("error"))
	num, err = c.Save()
	assert.NotNil(err)
	assert.Equal(int64(0), num)
	assert.Equal(err.Error(), "error")
	assert.Nil(mock.ExpectationsWereMet())
}

func TestCampaign_GetProgress(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	prExp := Progress{
		"Total":        100,
		"Queued":       19,
		"Delivered":    21,
		"NotDelivered": 5,
		"Sent":         5,
		"Error":        10,
		"Scheduled":    20,
		"Stopped":      5,
		"Pending":      15,
	}
	cp := Campaign{ID: 1}
	expected, _, _ := db.Get().From("Message").Select(goqu.L("status, count(*) as total")).Where(goqu.I("campaignid").Eq(1)).GroupBy("status").ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"status", "total"}).AddRow(
		"Queued", 19).AddRow("Delivered", 21).AddRow(
		"NotDelivered", 5).AddRow("Sent", 5).AddRow(
		"Error", 10).AddRow("Scheduled", 20).AddRow(
		"Stopped", 5))
	expected, _, _ = db.Get().From("Campaign").Select(&cp).Where(goqu.I("id").Eq(1)).Order(goqu.I(submittedAt).Desc()).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"id", "total"}).AddRow(1, prExp["Total"]))
	pr, err := cp.GetProgress()
	assert := assert.New(t)
	assert.Nil(err)
	assert.Equal(prExp, pr)
	assert.Nil(mock.ExpectationsWereMet())
}

func TestCampaign_GetReport(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	expReport := Report{
		ID: 1,
		Connections: []groupCount{{
			Name:  "Conn1",
			Count: 1200,
		}, {
			Name:  "Conn2",
			Count: 800,
		}},
		FirstQueued:   2000,
		LastSent:      2100,
		MsgSize:       2,
		PerConnection: "10.00",
		Throughput:    "20.00",
		Total:         1000,
		TotalMsgs:     2000,
		TotalTime:     100,
	}
	ds := db.Get().From("Message").Where(goqu.I("CampaignID").Eq(1))
	expected, _, _ := ds.Select(goqu.L("count(*) as Total")).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"Total"}).AddRow(expReport.Total))
	expected, _, _ = ds.Select(goqu.L("Total as MsgSize")).Limit(1).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"MsgSize"}).AddRow(expReport.MsgSize))
	expected, _, _ = ds.Select(goqu.L("Min(SentAt) as FirstQueued")).Where(goqu.I("sentat").Gt(0)).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"FirstQueued"}).AddRow(expReport.FirstQueued))
	expected, _, _ = ds.Select(goqu.L("Max(SentAt) as LastSent")).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"LastSent"}).AddRow(expReport.LastSent))
	expected, _, _ = ds.Select(goqu.L("Connection as name, count(*) as count")).GroupBy("Connection").ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"name", "count"}).AddRow(expReport.Connections[0].Name, expReport.Connections[0].Count).AddRow(expReport.Connections[1].Name, expReport.Connections[1].Count))
	c := Campaign{ID: 1}
	report, err := c.GetReport()
	assert := assert.New(t)
	assert.Nil(err)
	assert.Equal(expReport, report)

	// errors:
	err = errors.New("error")
	expected, _, _ = ds.Select(goqu.L("count(*) as Total")).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnError(err)
	expected, _, _ = ds.Select(goqu.L("Total as MsgSize")).Limit(1).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnError(err)
	expected, _, _ = ds.Select(goqu.L("Min(SentAt) as FirstQueued")).Where(goqu.I("sentat").Gt(0)).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnError(err)
	expected, _, _ = ds.Select(goqu.L("Max(SentAt) as LastSent")).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnError(err)
	expected, _, _ = ds.Select(goqu.L("Connection as name, count(*) as count")).GroupBy("Connection").ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnError(err)
	report, err = c.GetReport()
	assert.EqualError(err, "total query: error\nmsgSize query: error\nmin(SentAt) query: error\nmax(SentAt) query: error\nconnection query: error")
	assert.NotEqual(expReport, report)

	assert.Nil(mock.ExpectationsWereMet())

}

func TestList(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	cr := Criteria{
		Username:        "user1",
		SubmittedAfter:  10,
		SubmittedBefore: 20,
		OrderByKey:      "scheduledat",
		From:            "10",
		OrderByDir:      "ASC",
	}
	expCamps := []Campaign{
		{
			ID:       1,
			Username: "user1",
		},
		{
			ID:       2,
			Username: "user2",
		},
	}
	expected, _, _ := db.Get().From("Campaign").Select(&expCamps[0]).Where(goqu.I("submittedat").Gte(cr.SubmittedAfter), goqu.I("submittedat").Lte(cr.SubmittedBefore), goqu.I("username").Eq(cr.Username), goqu.I(cr.OrderByKey).Gt(10)).Order(goqu.I(cr.OrderByKey).Asc()).Limit(100).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(1, "user1").AddRow(2, "user2"))
	camps, err := List(cr)
	assert := assert.New(t)
	assert.Nil(err)
	assert.Equal(expCamps, camps)
	assert.Nil(mock.ExpectationsWereMet())
}
