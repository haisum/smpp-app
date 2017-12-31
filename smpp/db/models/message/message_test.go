package message

import (
	"testing"
	"reflect"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"regexp"
	"gopkg.in/stretchr/testify.v1/assert"
	"fmt"
)

func TestMessage_Validate(t *testing.T) {
	msg := Message{}
	errors := msg.Validate()
	if len(errors) == 0{
		t.Error("No errors returned when errors were expected.")
	}
	expectedErrors := []string{
		"Destination can't be empty.",
		"Can't send empty message",
		"Source address can't be empty.",
		"Encoding can either be latin or UCS",
	}
	if !reflect.DeepEqual(expectedErrors, errors){
		t.Errorf("Couldn't find expected error msgs. Expected: %+v, Got: %+v", expectedErrors, errors)
	}
	msg = Message{
		SendAfter: "jsadf",
		Dst : "asdf",
		Src : "Asdf",
		Msg : "asdffds",
		Enc :"invalid enc",
	}
	expectedErrors = []string{
		"Encoding can either be latin or UCS",
		"Send before time and Send after time, both should be provided at a time.",
		"Send after must be of 24 hour format such as \"09:00\".",
	}
	if !reflect.DeepEqual(expectedErrors, msg.Validate()){
		t.Errorf("Couldn't find expected error msgs. Expected: %+v, Got: %+v", expectedErrors, errors)
	}
	msg = Message{
		SendBefore: "jsadf",
		Dst : "asdf",
		Src : "Asdf",
		Msg : "asdffds",
		Enc :"ucs",
	}
	expectedErrors = []string{
		"Send before time and Send after time, both should be provided at a time.",
		"Send before must be of 24 hour format such as \"09:00\".",
	}
	if !reflect.DeepEqual(expectedErrors, msg.Validate()){
		t.Errorf("Couldn't find expected error msgs. Expected: %+v, Got: %+v", expectedErrors, errors)
	}
	msg.SendAfter = "09:00"
	msg.SendBefore = "07:00"
	if len(msg.Validate()) > 0{
		t.Errorf("No errors were expected")
	}
	msg.SendBefore = "23:782"
	msg.SendAfter = "241:09"
	expectedErrors = []string{
		"Send after must be of 24 hour format such as \"09:00\".",
		"Send before must be of 24 hour format such as \"09:00\".",
	}
	if !reflect.DeepEqual(expectedErrors, msg.Validate()){
		t.Errorf("Couldn't find expected error msgs. Expected: %+v, Got: %+v", expectedErrors, errors)
	}
}

func TestPrepareQuery(t *testing.T){
	con, _, _ := db.ConnectMock(t)
	defer con.Db.Close()
	qb := prepareQuery(Criteria{}, nil)
	assert := assert.New(t)
	expected := "SELECT * FROM Message ORDER BY QueuedAt DESC"
	actual,_,err := qb.ToSql()
	assert.Nil(err)
	assert.Equal(expected, actual)
	qb = prepareQuery(Criteria{OrderByDir: "ASC", OrderByKey: "Username"}, nil)
	expected = "SELECT * FROM Message ORDER BY Username ASC"
	actual,_,err = qb.ToSql()
	assert.Nil(err)
	assert.Equal(expected, actual)
	qb = prepareQuery(Criteria{Username: "(re)hello world"}, nil)
	expected = "SELECT * FROM Message WHERE match('@Username hello world') ORDER BY QueuedAt DESC"
	actual,_,err = qb.ToSql()
	assert.Nil(err)
	assert.Equal(expected, actual)
	expected = "SELECT * FROM Message WHERE User = 'haisum' AND match('@Msg hello world') AND QueuedAt >= 23 AND QueuedAt <= 435 AND DeliveredAt >= 234 AND DeliveredAt <= 3232 AND SentAt >= 23 AND SentAt <= 34 AND ScheduledAt >= 32 AND ScheduledAt <= 3245 AND RespID = 'ASdfsdf' AND Connection = 'asdf' AND ConnectionGroup = 'Asdfsdf' AND Src = 'sadfsdf' AND Dst = 'asdf' AND Enc = 'latin' AND Status = 'ASdfsfd' AND CampaignID = 12 AND Error = 'error' AND Total = 32 AND Priority = 3 AND SentAt < '234' ORDER BY SentAt DESC"
	cr := Criteria{
		ID : 23,
		Username: "haisum",
		Error: "error",
		Connection: "asdf",
		ConnectionGroup: "Asdfsdf",
		CampaignID: 12,
		Total : 32,
		Enc: "latin",
		Dst :"asdf",
		Src :"sadfsdf",
		Status: "ASdfsfd",
		RespID: "ASdfsdf",
		Msg :"hello world",
		Priority: 3,
		From : "234",
		DeliveredBefore:3232,
		DeliveredAfter:234,
		SentAfter: 23,
		SentBefore: 34,
		QueuedAfter:23,
		QueuedBefore:435,
		ScheduledAfter:32,
		ScheduledBefore:3245,
		OrderByKey: "SentAt",

	}
	qb = prepareQuery(cr, 234)
	actual,_,err = qb.ToSql()
	assert.Nil(err)
	assert.Equal(expected, actual)
	expected = "SELECT * FROM Message WHERE User = 'haisum' AND match('@Msg hello world') AND QueuedAt >= 23 AND QueuedAt <= 435 AND DeliveredAt >= 234 AND DeliveredAt <= 3232 AND SentAt >= 23 AND SentAt <= 34 AND ScheduledAt >= 32 AND ScheduledAt <= 3245 AND RespID = 'ASdfsdf' AND Connection = 'asdf' AND ConnectionGroup = 'Asdfsdf' AND Src = 'sadfsdf' AND Dst = 'asdf' AND Enc = 'latin' AND Status = 'ASdfsfd' AND CampaignID = 12 AND Error = 'error' AND Total = 32 AND Priority = 3 AND SentAt > '234' ORDER BY SentAt ASC"
	cr.OrderByDir = "ASC"
	qb = prepareQuery(cr, int64(234))
	actual,_,err = qb.ToSql()
	assert.Nil(err)
	assert.Equal(expected, actual)
}

func TestDeliverySM_Scan(t *testing.T) {
	dm := deliverySM{}
	dm.Scan([]byte(`{"hello" : "world"}`))
	if v, ok := dm["hello"]; !ok || v != "world"{
		t.Error("Couldn't scan deliverySM")
	}
}

func TestDeliverySM_Value(t *testing.T) {
	dm := deliverySM{"hello" : "world"}
	v, err := dm.Value()
	if err != nil {
		t.Errorf("error occurred: %s", err)
	}
	if string(v.([]byte)[:]) != `{"hello":"world"}` {
		t.Errorf("Unexpected value %s", string(v.([]byte)[:]) )
	}
}

func TestStatus_Scan(t *testing.T) {
	var st Status
	st.Scan([]uint8(Queued))
	if st != Queued{
		t.Error("Couldn't scan status")
	}
}

func TestListWithError(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectQuery("Status = 'Error' AND CampaignID = 1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "status", "campaignid"}).AddRow(
			1,string(Error),1).AddRow(
			2,string(Error),1))
	msgs, err := ListWithError(1)
	if err != nil {
		t.Errorf("Error in getting msgs %s", err)
	}
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(msgs))
	} else if msgs[0].ID != 1 || msgs[0].Status != Error || msgs[1].CampaignID != 1{
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
	if err := mock.ExpectationsWereMet(); err != nil{
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
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}

}

func TestGetStats(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, count(*) as total FROM Message WHERE QueuedAt < '3245' GROUP BY Status ORDER BY QueuedAt DESC")).WillReturnRows(
		sqlmock.NewRows([]string{"status", "total"}).AddRow(
			string(Queued), 1).AddRow(
			string(Error) , 4).AddRow(
			string(Sent) , 2).AddRow(
			string(Delivered) , 65).AddRow(
			string(NotDelivered) , 12).AddRow(
			string(Scheduled) , 45).AddRow(
			string(Stopped) , 23))
	stats, err := GetStats(Criteria{
		From : "3245",
	})
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	if stats.Stopped != 23 || stats.Scheduled != 45 || stats.Sent != 2 || stats.Delivered != 65  || stats.NotDelivered != 12 || stats.Error != 4 || stats.Queued != 1{
		t.Errorf("Error unexpected stats: %+v", stats)
	}
	if stats.Total != 152 {
		t.Errorf("Error unexpected stats: %+v", stats)
	}

	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestListQueued(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM Message WHERE Status = 'Queued' AND CampaignID = 33 ORDER BY QueuedAt DESC LIMIT 500000 option max_matches=500000")).WillReturnRows(
		sqlmock.NewRows([]string{"id"}).AddRow(
			1).AddRow(
			4).AddRow(
			2).AddRow(
			65).AddRow(
			12).AddRow(
			45).AddRow(
			23))
	ms , err := ListQueued(33)
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
	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestList(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM Message WHERE QueuedAt < '344' ORDER BY QueuedAt DESC`)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT `campaign`, `campaignid`, `connection`, `connectiongroup`, `deliveredat`, `dst`, `enc`, `error`, `id`, `isflash`, `msg`, `priority`, `queuedat`, `realmsg`, `respid`, `scheduledat`, `sendafter`, `sendbefore`, `sentat`, `src`, `status`, `total`, `username` FROM `Message` WHERE (`id` = 1) LIMIT 1")).WillReturnRows(sqlmock.NewRows([]string{"id", "msg"}).AddRow(1, "hello"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT `campaign`, `campaignid`, `connection`, `connectiongroup`, `deliveredat`, `dst`, `enc`, `error`, `id`, `isflash`, `msg`, `priority`, `queuedat`, `realmsg`, `respid`, `scheduledat`, `sendafter`, `sendbefore`, `sentat`, `src`, `status`, `total`, `username` FROM `Message` WHERE (`id` = 1) LIMIT 1")).WillReturnRows(sqlmock.NewRows([]string{"id", "msg"}).AddRow(1, "hello"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT `campaign`, `campaignid`, `connection`, `connectiongroup`, `deliveredat`, `dst`, `enc`, `error`, `id`, `isflash`, `msg`, `priority`, `queuedat`, `realmsg`, `respid`, `scheduledat`, `sendafter`, `sendbefore`, `sentat`, `src`, `status`, `total`, `username` FROM `Message` WHERE (`id` = 2) LIMIT 1")).WillReturnRows(sqlmock.NewRows([]string{"id", "msg"}).AddRow(2, "world"))
	ms, err := List(Criteria{
		From : "344",
		FetchMsg: true,
	})
	if err != nil {
		t.Errorf("Error %s", err)
	}
	if len (ms) != 2{
		t.Errorf("Unexpected msg count %d", len(ms))
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestMessage_Save(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `Message` (`respid`, `connectiongroup`, `connection`, `total`, `username`, `msg`, `realmsg`, `enc`, `dst`, `src`, `priority`, `queuedat`, `sentat`, `deliveredat`, `campaignid`, `campaign`, `status`, `error`, `sendbefore`, `sendafter`, `scheduledat`, `isflash`) VALUES ('23434asf', '', 'Default', 0, '', '', '', '', '', '', 0, 0, 0, 0, 0, '', 'Not Delivered', 'NotDelivered', '', '', 0, 0)")).WillReturnResult(sqlmock.NewResult(21, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO Message(id, Msg, Username, ConnectionGroup, Connection, RespID, Total, Enc, Dst, Src, Priority, QueuedAt, SentAt, DeliveredAt, CampaignID, Campaign, Status, Error, User, ScheduledAt, IsFlash, SendAfter, SendBefore) VALUES (21, '', '', '', 'Default', '23434asf', 0, '', '', '', 0, 0 , 0, 0, 0, '', 'Not Delivered', 'NotDelivered', '', 0, 0, '', '')`)).WillReturnResult(sqlmock.NewResult(0,1))
	m := Message{
		RespID: "23434asf",
		Connection: "Default",
		Error: "NotDelivered",
		Status: NotDelivered,
	}
	id, err := m.Save()
	if err != nil {
		t.Errorf("Error: %s", err)
	}
	if id != 21 {
		t.Errorf("id was unexpected: %d", id)
	}
	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestMessage_Update(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `Message` SET `id`=34,`respid`='',`connectiongroup`='',`connection`='',`total`=0,`username`='',`msg`='asdf',`realmsg`='',`enc`='',`dst`='',`src`='',`priority`=0,`queuedat`=0,`sentat`=0,`deliveredat`=0,`campaignid`=0,`campaign`='',`status`='',`error`='',`sendbefore`='',`sendafter`='',`scheduledat`=0,`isflash`=0 WHERE (`id` = 34)")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`REPLACE INTO Message(id, Msg, Username, ConnectionGroup, Connection, RespID, Total, Enc, Dst, Src, Priority, QueuedAt, SentAt, DeliveredAt, CampaignID, Campaign, Status, Error, User, ScheduledAt, IsFlash, SendAfter, SendBefore) VALUES (34, 'asdf', '', '', '', '', 0, '', '', '', 0, 0 , 0, 0, 0, '', '', '', '', 0, 0, '', '')`)).WillReturnResult(sqlmock.NewResult(0, 1))
	m := Message{
		ID: 34,
		Msg : "asdf",
	}
	err := m.Update()
	if err != nil {
		t.Errorf("%s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestSaveDelivery(t *testing.T) {
	db, mock, _ := db.ConnectMock(t)
	defer db.Db.Close()
	mock.ExpectQuery(`SELECT \* FROM Message WHERE RespID = '1234abcd'`).WillReturnRows(sqlmock.NewRows([]string{"id", "respid"}).AddRow(1, "1234abcd"))
	mock.ExpectExec("UPDATE `Message` SET `DeliveredAt`=\\d+,`Status`='Delivered' WHERE \\(`RespID` = '1234abcd'\\)").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("REPLACE INTO Message.*1234abcd").WillReturnResult(sqlmock.NewResult(0, 1))
	// now we execute our method
	if err := SaveDelivery("1234abcd", string(Delivered)); err != nil {
		t.Errorf("error was not expected while saving delivery: %s", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestSaveBulk(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	messages := []Message{
		{
			ID : 34,
			Msg : "hello world",
			Enc : "latin",
		},
		{
			ID : 35,
			Msg : "pa pa",
			Enc : "ucs",
		},
	}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `Message` (`respid`, `connectiongroup`, `connection`, `total`, `username`, `msg`, `realmsg`, `enc`, `dst`, `src`, `priority`, `queuedat`, `sentat`, `deliveredat`, `campaignid`, `campaign`, `status`, `error`, `sendbefore`, `sendafter`, `scheduledat`, `isflash`) VALUES ('', '', '', 0, '', 'hello world', '', 'latin', '', '', 0, 0, 0, 0, 0, '', '', '', '', '', 0, 0), ('', '', '', 0, '', 'pa pa', '', 'ucs', '', '', 0, 0, 0, 0, 0, '', '', '', '', '', 0, 0)")).WillReturnResult(sqlmock.NewResult(35, 2))
	mock.ExpectQuery("SELECT `id` FROM `Message` ORDER BY `id` DESC LIMIT " + fmt.Sprintf("%d", len(messages))).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(34).AddRow(35))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO Message(id, Msg, Username, ConnectionGroup, Connection, RespID, Total, Enc, Dst, Src, Priority, QueuedAt, SentAt, DeliveredAt, CampaignID, Campaign, Status, Error, User, ScheduledAt, IsFlash, SendAfter, SendBefore) VALUES (34, 'hello world', '', '', '', '', 0, 'latin', '', '', 0, 0 , 0, 0, 0, '', '', '', '', 0, 0, '', ''),(35, 'pa pa', '', '', '', '', 0, 'ucs', '', '', 0, 0 , 0, 0, 0, '', '', '', '', 0, 0, '', '')")).WillReturnResult(sqlmock.NewResult(35, 2))
	ids, err := SaveBulk(messages)
	if err != nil {
		t.Errorf("error :%s", err)
		t.FailNow()
	}
	if ids[0] != 34 || ids[1] != 35 {
		t.Errorf("unexpected ids:%v", ids)
	}
	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestStopPending(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `Message` SET `Status`='Stopped' WHERE ((`CampaignID` = 1) AND ((`Status` = 'Queued') OR (`Status` = 'Scheduled')))")).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(`SELECT \* FROM Message WHERE Status = 'Stopped' AND CampaignID = 1`).WillReturnRows(sqlmock.NewRows([]string{"id", "status", "campaignid"}).AddRow(1, "Stopped", 1).AddRow(2, "Stopped", 1))
	mock.ExpectExec(regexp.QuoteMeta(`REPLACE INTO Message(id, Msg, Username, ConnectionGroup, Connection, RespID, Total, Enc, Dst, Src, Priority, QueuedAt, SentAt, DeliveredAt, CampaignID, Campaign, Status, Error, User, ScheduledAt, IsFlash, SendAfter, SendBefore) VALUES (1, '', '', '', '', '', 0, '', '', '', 0, 0 , 0, 0, 1, '', 'Stopped', '', '', 0, 0, '', ''),(2, '', '', '', '', '', 0, '', '', '', 0, 0 , 0, 0, 1, '', 'Stopped', '', '', 0, 0, '', '')`)).WillReturnResult(sqlmock.NewResult(0, 2))
	n, err := StopPending(1)
	if err != nil {
		t.Errorf("Error %s was not expected", err)
	}
	if n != 2 {
		t.Errorf("2 expected, got %d", n)
	}
	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
	if err = mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}