package models

import (
	"testing"
	"reflect"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"bitbucket.org/codefreak/hsmpp/smpp/db/sphinx"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
)

func TestMessage_Validate(t *testing.T) {
	msg := Message{}
	errors := msg.Validate()
	if len(errors) == 0{
		t.Error("No errors returned when errors were expected.")
		t.Fail()
	}
	expectedErrors := []string{
		"Destination can't be empty.",
		"Can't send empty message",
		"Source address can't be empty.",
		"Encoding can either be latin or UCS",
	}
	if !reflect.DeepEqual(expectedErrors, errors){
		t.Errorf("Couldn't find expected error msgs. Expected: %+v, Got: %+v", expectedErrors, errors)
		t.Fail()
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
		t.Fail()
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
		t.Fail()
	}
	msg.SendAfter = "09:00"
	msg.SendBefore = "07:00"
	if len(msg.Validate()) > 0{
		t.Errorf("No errors were expected")
		t.Fail()
	}
	msg.SendBefore = "23:782"
	msg.SendAfter = "241:09"
	expectedErrors = []string{
		"Send after must be of 24 hour format such as \"09:00\".",
		"Send before must be of 24 hour format such as \"09:00\".",
	}
	if !reflect.DeepEqual(expectedErrors, msg.Validate()){
		t.Errorf("Couldn't find expected error msgs. Expected: %+v, Got: %+v", expectedErrors, errors)
		t.Fail()
	}
}

func TestPrepareMsgTerm(t *testing.T){
	qb := prepareMsgTerm(Criteria{}, nil)
	expected := "SELECT * FROM Message ORDER BY QueuedAt DESC"
	assertEqual(t, expected, qb.GetQuery())
	qb = prepareMsgTerm(Criteria{OrderByDir: "ASC", OrderByKey:"Username"}, nil)
	expected = "SELECT * FROM Message ORDER BY Username ASC"
	assertEqual(t, expected, qb.GetQuery())
	qb = prepareMsgTerm(Criteria{Username: "(re)hello world"}, nil)
	expected = "SELECT * FROM Message WHERE match('@Username hello world') ORDER BY QueuedAt DESC"
	assertEqual(t, qb.GetQuery(), expected)
	expected = "SELECT * FROM Message WHERE User = 'haisum' AND match('@Msg hello world') AND QueuedAt > 23 AND QueuedAt < 435 AND DeliveredAt > 234 AND DeliveredAt < 3232 AND SentAt > 23 AND SentAt < 34 AND ScheduledAt > 32 AND ScheduledAt < 3245 AND RespID = 'ASdfsdf' AND Connection = 'asdf' AND ConnectionGroup = 'Asdfsdf' AND Src = 'sadfsdf' AND Dst = 'asdf' AND Enc = 'latin' AND Status = 'ASdfsfd' AND CampaignID = 12 AND Error = 'error' AND Total = 32 AND Priority = 3 AND SentAt < '234' ORDER BY SentAt DESC"
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
	qb = prepareMsgTerm(cr, 234)
	assertEqual(t, expected, qb.GetQuery())
	expected = "SELECT * FROM Message WHERE User = 'haisum' AND match('@Msg hello world') AND QueuedAt > 23 AND QueuedAt < 435 AND DeliveredAt > 234 AND DeliveredAt < 3232 AND SentAt > 23 AND SentAt < 34 AND ScheduledAt > 32 AND ScheduledAt < 3245 AND RespID = 'ASdfsdf' AND Connection = 'asdf' AND ConnectionGroup = 'Asdfsdf' AND Src = 'sadfsdf' AND Dst = 'asdf' AND Enc = 'latin' AND Status = 'ASdfsfd' AND CampaignID = 12 AND Error = 'error' AND Total = 32 AND Priority = 3 AND SentAt > '234' ORDER BY SentAt ASC"
	cr.OrderByDir = "ASC"
	qb = prepareMsgTerm(cr, int64(234))
	assertEqual(t, expected, qb.GetQuery())
}

func TestDeliverySM_Scan(t *testing.T) {
	dm := deliverySM{}
	dm.Scan([]byte(`{"hello" : "world"}`))
	if v, ok := dm["hello"]; !ok || v != "world"{
		t.Error("Couldn't scan deliverySM")
		t.Fail()
	}
}

func TestDeliverySM_Value(t *testing.T) {
	dm := deliverySM{"hello" : "world"}
	v, err := dm.Value()
	if err != nil {
		t.Errorf("error occurred: %s", err)
		t.Fail()
	}
	if string(v.([]byte)[:]) != `{"hello":"world"}` {
		t.Errorf("Unexpected value %s", string(v.([]byte)[:]) )
		t.Fail()
	}
}

func TestStatus_Scan(t *testing.T) {
	var st Status
	st.Scan([]uint8(Queued))
	if st != Queued{
		t.Error("Couldn't scan status")
		t.Fail()
	}
}

func TestGetErrorMessages(t *testing.T) {
	spdb, mock, _ := sphinx.ConnectMock(t)
	defer spdb.Db.Close()
	mock.ExpectQuery("Status = 'Error' AND CampaignID = 1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "status", "campaignid"}).AddRow(
			1,string(Error),1).AddRow(
			2,string(Error),1))
	msgs, err := GetErrorMessages(1)
	if err != nil {
		t.Errorf("Error in getting msgs %s", err)
		t.Fail()
	}
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(msgs))
		t.Fail()
	}
	if msgs[0].ID != 1 || msgs[0].Status != Error || msgs[1].CampaignID != 1{
		t.Errorf("Unexpected msg objects. %+v", msgs)
		t.Fail()
	}
	if err := mock.ExpectationsWereMet(); err != nil {

	}
}


func TestGetMessage(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectQuery(`"id" = 21`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(21))
	msg, err := GetMessage(21)
	if err != nil {
		t.Errorf("error in getting msg. %s", err)
		t.FailNow()
	}
	if msg.ID != 21 {
		t.Errorf("Unexpected msg %+v", msg)
		t.Fail()
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
		t.Fail()
	}
	con, mock, _ = db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectQuery(`"id" = 21`).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
	msg, err = GetMessage(21)
	if err == nil {
		t.Error("error expected.")
		t.FailNow()
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
		t.Fail()
	}

}

func TestGetMessageStats(t *testing.T) {

}

func TestGetQueuedMessages(t *testing.T) {

}

func TestGetMessages(t *testing.T) {

}

func TestMessage_Save(t *testing.T) {

}

func TestMessage_Update(t *testing.T) {

}

func TestSaveBulk(t *testing.T) {

}

func TestSaveDelivery(t *testing.T) {
	sp, mock, _ := sphinx.ConnectMock(t)
	defer sp.Db.Close()
	db, dbmock, _ := db.ConnectMock(t)
	defer db.Db.Close()
	query  := `"RespID" = '1234abcd'`
	mock.ExpectQuery(query).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	dbmock.ExpectQuery(`SELECT "campaign"`).WillReturnRows(sqlmock.NewRows([]string{"id", "respid", "isflash"}).AddRow(1, "1234abcd", 1))
	mock.ExpectExec("REPLACE INTO Message.*1234abcd").WillReturnResult(sqlmock.NewResult(0, 1))
	// now we execute our method
	if err := SaveDelivery("1234abcd", string(Delivered)); err != nil {
		t.Errorf("error was not expected while saving delivery: %s", err)
		t.Fail()
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
		t.Fail()
	}
	if err := dbmock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
		t.Fail()
	}
}

func assertEqual(t *testing.T, expected, got string) {
	if got !=  expected{
		t.Errorf("Expected: %s\n Got: %s", expected, got)
		t.Fail()
	}
}