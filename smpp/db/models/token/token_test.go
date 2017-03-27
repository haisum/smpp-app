package models

import (
	"testing"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"regexp"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	"time"
	"errors"
)

func TestGetToken(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	tok1 := "sampletoken1"
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `Token` WHERE (`Token` = ?) LIMIT ?")).WithArgs(stringutils.ToSHA1(tok1), 1).WillReturnRows(sqlmock.NewRows([]string{"id", "token", "username", "lastaccessed"}).AddRow(2,"888f45a334f014f763bc3fb7d0afd24daa6c5e0f", "user1", time.Now().Add(- time.Hour * 24).UTC().Unix()))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `Token` SET `id`=?,`lastaccessed`=?,`token`=?,`username`=?,`validity`=?")).WillReturnResult(sqlmock.NewResult(0, 1))
	gTok, err := GetToken(tok1)
	if err != nil {
		t.Errorf("error: %s", err)
		t.Fail()
	}
	if gTok.Username != "user1" {
		t.Errorf("Expected username user1. Got %s", gTok.Username)
		t.Fail()
	}
	//check validity checks
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `Token` WHERE (`Token` = ?) LIMIT ?")).WithArgs(stringutils.ToSHA1(tok1), 1).WillReturnRows(sqlmock.NewRows([]string{"id", "token", "username", "lastaccessed"}).AddRow(2,"888f45a334f014f763bc3fb7d0afd24daa6c5e0f", "user1", time.Date(now.Year(), now.Month(), now.Day()-DefaultTokenValidity-1, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location()).Unix()))
	gTok, err = GetToken(tok1)
	if err == nil || err.Error() != "Token has expired." {
		t.Errorf("Token has expired. error expected. %s", err)
		t.Fail()
	}
	//check affected
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `Token` WHERE (`Token` = ?) LIMIT ?")).WithArgs(stringutils.ToSHA1(tok1), 1).WillReturnRows(sqlmock.NewRows([]string{"id", "token", "username", "lastaccessed"}).AddRow(2,"888f45a334f014f763bc3fb7d0afd24daa6c5e0f", "user1", time.Now().Add(- time.Hour * 24).UTC().Unix()))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `Token` SET `id`=?,`lastaccessed`=?,`token`=?,`username`=?,`validity`=?")).WillReturnResult(sqlmock.NewResult(0, 0))
	gTok, err = GetToken(tok1)
	if err == nil || err.Error() != "Last affected isn't equal to 1" {
		t.Error("Last affected isn't equal to 1 error expected.")
		t.Fail()
	}
	//check error
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `Token` WHERE (`Token` = ?) LIMIT ?")).WithArgs(stringutils.ToSHA1(tok1), 1).WillReturnRows(sqlmock.NewRows([]string{"id", "token", "username", "lastaccessed"}).AddRow(2,"888f45a334f014f763bc3fb7d0afd24daa6c5e0f", "user1", time.Now().Add(- time.Hour * 24).UTC().Unix()))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE `Token` SET `id`=?,`lastaccessed`=?,`token`=?,`username`=?,`validity`=?")).WillReturnError(errors.New("error"))
	gTok, err = GetToken(tok1)
	if err == nil || err.Error() != "error" {
		t.Error("error expected.")
		t.Fail()
	}
	//check select error
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `Token` WHERE (`Token` = ?) LIMIT ?")).WithArgs(stringutils.ToSHA1(tok1), 1).WillReturnError(errors.New("select error"))
	gTok, err = GetToken(tok1)
	if err == nil || err.Error() != "select error" {
		t.Error("error expected.")
		t.Fail()
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
		t.Fail()
	}

}

func TestCreateToken(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectExec("INSERT INTO `Token` \\(`lastaccessed`, `token`, `username`, `validity`\\) VALUES \\(\\d+, '[a-z0-9]+', 'user1', 100\\)").WillReturnResult(sqlmock.NewResult(2,1))
	_, err := CreateToken("user1", 100)
	if err != nil {
		t.Errorf("error: %s", err)
		t.Fail()
	}
	mock.ExpectExec("INSERT INTO `Token` \\(`lastaccessed`, `token`, `username`, `validity`\\) VALUES \\(\\d+, '[a-z0-9]+', 'user1', 30\\)").WillReturnError(errors.New("error"))
	_, err = CreateToken("user1", 0)
	if err == nil || err.Error() != "error" {
		t.Errorf("error expected %s", err)
		t.Fail()
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
		t.Fail()
	}
}

func TestToken_Delete(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	tok1 := Token{
		ID : 324,
		Token: "sadfsdf",
	}
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `Token` WHERE (`token` = 'sadfsdf')")).WillReturnResult(sqlmock.NewResult(0,1))
	err := tok1.Delete()
	if err != nil {
		t.Errorf("error: %s", err)
		t.Fail()
	}
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `Token` WHERE (`token` = 'sadfsdf')")).WillReturnError(errors.New("error"))
	err = tok1.Delete()
	if err == nil || err.Error() != "error" {
		t.Errorf("error expected %s", err)
		t.Fail()
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
		t.Fail()
	}
}
func TestToken_DeleteAll(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	tok1 := Token{
		ID : 324,
		Token: "sadfsdf",
		Username: "user1",
	}
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `Token` WHERE (`username` = 'user1')")).WillReturnResult(sqlmock.NewResult(0,1))
	err := tok1.DeleteAll()
	if err != nil {
		t.Errorf("error: %s", err)
		t.Fail()
	}
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `Token` WHERE (`username` = 'user1')")).WillReturnError(errors.New("error"))
	err = tok1.DeleteAll()
	if err == nil || err.Error() != "error" {
		t.Errorf("error expected %s", err)
		t.Fail()
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
		t.Fail()
	}
}
