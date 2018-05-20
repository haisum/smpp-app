package message

import (
	"reflect"
	"regexp"
	"testing"

	"time"

	"bitbucket.org/codefreak/hsmpp/pkg/db"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gopkg.in/doug-martin/goqu.v3"
	"gopkg.in/stretchr/testify.v1/assert"
)

func TestMessage_Validate(t *testing.T) {
	msg := Message{}
	errors := msg.Validate()
	if len(errors) == 0 {
		t.Error("No errors returned when errors were expected.")
	}
	expectedErrors := []string{
		"Destination can't be empty.",
		"Can't send empty message",
		"Source address can't be empty.",
		"Encoding can either be latin or UCS",
	}
	if !reflect.DeepEqual(expectedErrors, errors) {
		t.Errorf("Couldn't find expected error msgs. Expected: %+v, Got: %+v", expectedErrors, errors)
	}
	msg = Message{
		SendAfter: "jsadf",
		Dst:       "asdf",
		Src:       "Asdf",
		Msg:       "asdffds",
		Enc:       "invalid enc",
	}
	expectedErrors = []string{
		"Encoding can either be latin or UCS",
		"Send before time and Send after time, both should be provided at a time.",
		"Send after must be of 24 hour format such as \"09:00\".",
	}
	if !reflect.DeepEqual(expectedErrors, msg.Validate()) {
		t.Errorf("Couldn't find expected error msgs. Expected: %+v, Got: %+v", expectedErrors, errors)
	}
	msg = Message{
		SendBefore: "jsadf",
		Dst:        "asdf",
		Src:        "Asdf",
		Msg:        "asdffds",
		Enc:        "ucs",
	}
	expectedErrors = []string{
		"Send before time and Send after time, both should be provided at a time.",
		"Send before must be of 24 hour format such as \"09:00\".",
	}
	if !reflect.DeepEqual(expectedErrors, msg.Validate()) {
		t.Errorf("Couldn't find expected error msgs. Expected: %+v, Got: %+v", expectedErrors, errors)
	}
	msg.SendAfter = "09:00"
	msg.SendBefore = "07:00"
	if len(msg.Validate()) > 0 {
		t.Errorf("No errors were expected")
	}
	msg.SendBefore = "23:782"
	msg.SendAfter = "241:09"
	expectedErrors = []string{
		"Send after must be of 24 hour format such as \"09:00\".",
		"Send before must be of 24 hour format such as \"09:00\".",
	}
	if !reflect.DeepEqual(expectedErrors, msg.Validate()) {
		t.Errorf("Couldn't find expected error msgs. Expected: %+v, Got: %+v", expectedErrors, errors)
	}
}

func TestPrepareQuery(t *testing.T) {
	con, _, _ := db.ConnectMock(t)
	defer con.Db.Close()
	qb := prepareQuery(Criteria{}, nil)
	assert := assert.New(t)
	expected, _, err := db.Get().From("Message").Order(goqu.I("queuedAt").Desc()).ToSql()
	assert.Nil(err)
	actual, _, err := qb.ToSql()
	assert.Nil(err)
	assert.Equal(expected, actual)
	qb = prepareQuery(Criteria{OrderByDir: "ASC", OrderByKey: "Username"}, nil)
	expected, _, err = db.Get().From("Message").Order(goqu.I("Username").Asc()).ToSql()
	assert.Nil(err)
	actual, _, err = qb.ToSql()
	assert.Nil(err)
	assert.Equal(expected, actual)
	qb = prepareQuery(Criteria{Username: "(re)hello world"}, nil)
	expected, _, err = db.Get().From("Message").Where(goqu.L(userTextSearchLiteral, "hello world")).Order(goqu.I("queuedAt").Desc()).ToSql()
	assert.Nil(err)
	actual, _, err = qb.ToSql()
	assert.Nil(err)
	assert.Equal(expected, actual)
	cr := Criteria{
		ID:              23,
		Username:        "haisum",
		Error:           "error",
		Connection:      "asdf",
		ConnectionGroup: "Asdfsdf",
		CampaignID:      12,
		Total:           32,
		Enc:             "latin",
		Dst:             "asdf",
		Src:             "sadfsdf",
		Status:          "ASdfsfd",
		RespID:          "ASdfsdf",
		Msg:             "hello world",
		Priority:        3,
		From:            "234",
		DeliveredBefore: 3232,
		DeliveredAfter:  234,
		SentAfter:       23,
		SentBefore:      34,
		QueuedAfter:     23,
		QueuedBefore:    435,
		ScheduledAfter:  32,
		ScheduledBefore: 3245,
		OrderByKey:      "SentAt",
	}
	expected, _, err = db.Get().From("Message").Where(goqu.I("Username").Eq("haisum"), goqu.L(msgTextSearchLiteral, "hello world"),
		goqu.I("queuedAt").Gte(23), goqu.I("queuedAt").Lte(435),
		goqu.I("DeliveredAt").Gte(234), goqu.I("DeliveredAt").Lte(3232), goqu.I("SentAt").Gte(23),
		goqu.I("SentAt").Lte(34), goqu.I("ScheduledAt").Gte(32),
		goqu.I("ScheduledAt").Lte(3245), goqu.I("RespID").Eq("ASdfsdf"), goqu.I("Connection").Eq("asdf"), goqu.I("ConnectionGroup").Eq("Asdfsdf"), goqu.I("Src").Eq("sadfsdf"), goqu.I("Dst").Eq("asdf"), goqu.I("Enc").Eq("latin"),
		goqu.I("Status").Eq("ASdfsfd"), goqu.I("CampaignID").Eq(12), goqu.I("Error").Eq("error"), goqu.I("Total").Eq(32), goqu.I("Priority").Eq(3), goqu.I("SentAt").Lt(234)).Order(goqu.I("SentAt").Desc()).ToSql()
	assert.Nil(err)
	qb = prepareQuery(cr, 234)
	actual, _, err = qb.ToSql()
	assert.Nil(err)
	assert.Equal(expected, actual)
	expected = "SELECT * FROM Message WHERE User = 'haisum' AND match('@Msg hello world') AND queuedAt >= 23 AND queuedAt <= 435 AND DeliveredAt >= 234 AND DeliveredAt <= 3232 AND SentAt >= 23 AND SentAt <= 34 AND ScheduledAt >= 32 AND ScheduledAt <= 3245 AND RespID = 'ASdfsdf' AND Connection = 'asdf' AND ConnectionGroup = 'Asdfsdf' AND Src = 'sadfsdf' AND Dst = 'asdf' AND Enc = 'latin' AND Status = 'ASdfsfd' AND CampaignID = 12 AND Error = 'error' AND Total = 32 AND Priority = 3 AND SentAt > '234' ORDER BY SentAt ASC"
	cr.OrderByDir = "ASC"
	expected, _, err = db.Get().From("Message").Where(goqu.I("Username").Eq("haisum"), goqu.L(msgTextSearchLiteral, "hello world"),
		goqu.I("queuedAt").Gte(23), goqu.I("queuedAt").Lte(435),
		goqu.I("DeliveredAt").Gte(234), goqu.I("DeliveredAt").Lte(3232), goqu.I("SentAt").Gte(23),
		goqu.I("SentAt").Lte(34), goqu.I("ScheduledAt").Gte(32),
		goqu.I("ScheduledAt").Lte(3245), goqu.I("RespID").Eq("ASdfsdf"), goqu.I("Connection").Eq("asdf"), goqu.I("ConnectionGroup").Eq("Asdfsdf"), goqu.I("Src").Eq("sadfsdf"), goqu.I("Dst").Eq("asdf"), goqu.I("Enc").Eq("latin"),
		goqu.I("Status").Eq("ASdfsfd"), goqu.I("CampaignID").Eq(12), goqu.I("Error").Eq("error"), goqu.I("Total").Eq(32), goqu.I("Priority").Eq(3), goqu.I("SentAt").Gt(234)).Order(goqu.I("SentAt").Asc()).ToSql()
	qb = prepareQuery(cr, int64(234))
	actual, _, err = qb.ToSql()
	assert.Nil(err)
	assert.Equal(expected, actual)
}

func TestDeliverySM_Scan(t *testing.T) {
	dm := deliverySM{}
	dm.Scan([]byte(`{"hello" : "world"}`))
	if v, ok := dm["hello"]; !ok || v != "world" {
		t.Error("Couldn't scan deliverySM")
	}
}

func TestDeliverySM_Value(t *testing.T) {
	dm := deliverySM{"hello": "world"}
	v, err := dm.Value()
	if err != nil {
		t.Errorf("error occurred: %s", err)
	}
	if string(v.([]byte)[:]) != `{"hello":"world"}` {
		t.Errorf("Unexpected value %s", string(v.([]byte)[:]))
	}
}

func TestStatus_Scan(t *testing.T) {
	var st Status
	st.Scan([]uint8(Queued))
	if st != Queued {
		t.Error("Couldn't scan status")
	}
}

func TestListWithError(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	expected, _, _ := db.Get().From("Message").Select(&Message{}).Where(goqu.I("Status").Eq(Error), goqu.I("CampaignID").Eq(1)).Order(goqu.I("queuedAt").Desc()).Limit(maxPerPageListing).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(
		sqlmock.NewRows([]string{"id", "status", "campaignid"}).AddRow(
			1, string(Error), 1).AddRow(
			2, string(Error), 1))
	msgs, err := ListWithError(1)
	if err != nil {
		t.Errorf("Error in getting msgs %s", err)
	}
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(msgs))
	} else if msgs[0].ID != 1 || msgs[0].Status != Error || msgs[1].CampaignID != 1 {
		t.Errorf("Unexpected msg objects. %+v", msgs)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Expectations not met. %s", err)
	}
}

func TestGet(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectQuery("`id` = 21").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(21))
	msg, err := Get(21)
	if err != nil {
		t.Errorf("error in getting msg. %s", err)
		t.FailNow()
	}
	if msg.ID != 21 {
		t.Errorf("Unexpected msg %+v", msg)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	con, mock, _ = db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectQuery("`id` = 21").WillReturnRows(sqlmock.NewRows([]string{"ID"}))
	msg, err = Get(21)
	if err == nil {
		t.Error("error expected.")
		t.FailNow()
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}

}

func TestGetStats(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	expected, _, _ := db.Get().From("Message").Where(goqu.I("queuedAt").Lt(3245)).GroupBy("Status").Select(goqu.L("status, count(*) as total")).Order(goqu.I("queuedAt").Desc()).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(
		sqlmock.NewRows([]string{"status", "total"}).AddRow(
			string(Queued), 1).AddRow(
			string(Error), 4).AddRow(
			string(Sent), 2).AddRow(
			string(Delivered), 65).AddRow(
			string(NotDelivered), 12).AddRow(
			string(Scheduled), 45).AddRow(
			string(Stopped), 23))
	stats, err := GetStats(Criteria{
		From: "3245",
	})
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	if stats.Stopped != 23 || stats.Scheduled != 45 || stats.Sent != 2 || stats.Delivered != 65 || stats.NotDelivered != 12 || stats.Error != 4 || stats.Queued != 1 {
		t.Errorf("Error unexpected stats: %+v", stats)
	}
	if stats.Total != 152 {
		t.Errorf("Error unexpected stats: %+v", stats)
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestListQueued(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	expected, _, err := db.Get().From("Message").Select(&Message{}).Where(goqu.I("Status").Eq("Queued"), goqu.I("CampaignID").Eq(33)).Order(goqu.I("queuedAt").Desc()).Limit(maxPerPageListing).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(
			1).AddRow(
			4).AddRow(
			2).AddRow(
			65).AddRow(
			12).AddRow(
			45).AddRow(
			23))
	ms, err := ListQueued(33)
	if err != nil {
		t.Errorf("Error %s", err)
	}
	if len(ms) != 7 {
		t.Error("Unpexcted msg count")
		t.FailNow()
	}
	if ms[5].ID != 45 {
		t.Errorf("Unexpected msg value")
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestList(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	expected, _, err := db.Get().From("Message").Select(&Message{}).Where(goqu.I("queuedAt").Lt(344)).Order(goqu.I("queuedAt").Desc()).Limit(defaultPerPageListing).ToSql()
	assert.Nil(t, err)
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))
	ms, err := List(Criteria{
		From: "344",
	})
	if err != nil {
		t.Errorf("Error %s", err)
	}
	if len(ms) != 2 {
		t.Errorf("Unexpected msg count %d", len(ms))
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestMessage_Save(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	m := Message{
		RespID:     "23434asf",
		Connection: "Default",
		Error:      "NotDelivered",
		Status:     NotDelivered,
	}
	expected, _, _ := db.Get().From("Message").ToInsertSql(&m)
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(0, 1))
	_, err := m.Save()
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestMessage_Update(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	m := Message{
		ID:  34,
		Msg: "asdf",
	}
	expected, _, _ := db.Get().From("Message").Where(goqu.I("id").Eq(m.ID)).ToUpdateSql(&m)
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(0, 1))
	err := m.Update()
	if err != nil {
		t.Errorf("%s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestSaveDelivery(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	expected, _, _ := db.Get().From("Message").Where(goqu.I("RespID").Eq("1234abcd")).ToUpdateSql(goqu.Record{
		"Status":      string(Delivered),
		"DeliveredAt": time.Now().UTC().Unix(),
	})
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := SaveDelivery("1234abcd", string(Delivered)); err != nil {
		t.Errorf("error was not expected while saving delivery: %s", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestSaveBulk(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	messages := []Message{
		{
			ID:  34,
			Msg: "hello world",
			Enc: "latin",
		},
		{
			ID:  35,
			Msg: "pa pa",
			Enc: "ucs",
		},
	}
	expected, _, _ := db.Get().From("Message").ToInsertSql(interface{}(messages))
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(35, 2))
	expected, _, _ = db.Get().From("Message").Select("id").Order(goqu.I("id").Desc()).Limit(uint(2)).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(34).AddRow(35))
	ids, err := SaveBulk(messages)
	if err != nil {
		t.Errorf("error :%s", err)
		t.FailNow()
	}
	if ids[0] != 34 || ids[1] != 35 {
		t.Errorf("unexpected ids:%v", ids)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestStopPending(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	expected, _, _ := db.Get().From("Message").Where(goqu.I("CampaignID").Eq(1),
		goqu.Or(
			goqu.I("Status").Eq(Queued),
			goqu.I("Status").Eq(Scheduled),
		),
	).ToUpdateSql(goqu.Record{"Status": Stopped})
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(0, 2))
	n, err := StopPending(1)
	if err != nil {
		t.Errorf("Error %s was not expected", err)
	}
	if n != 2 {
		t.Errorf("2 expected, got %d", n)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}
