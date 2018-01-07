package token

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gopkg.in/doug-martin/goqu.v3"
)

func TestGet(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	tok1 := "sampletoken1"
	selectExpected, _, _ := db.Get().From("Token").Where(goqu.I("Token").Eq(stringutils.ToSHA1(tok1))).Prepared(true).Select(&Token{}).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(selectExpected)).WithArgs(stringutils.ToSHA1(tok1), 1).WillReturnRows(sqlmock.NewRows([]string{"id", "token", "username", "lastaccessed"}).AddRow(2, "888f45a334f014f763bc3fb7d0afd24daa6c5e0f", "user1", time.Now().Add(-time.Hour*24).UTC().Unix()))
	token := Token{
		ID:           2,
		Username:     "user1",
		Token:        "888f45a334f014f763bc3fb7d0afd24daa6c5e0f",
		LastAccessed: time.Now().Unix(),
		Validity:     defaultTokenValidity,
	}
	updateExpected, _, _ := db.Get().From("Token").Where(goqu.I("ID").Eq(2)).ToUpdateSql(&token)
	mock.ExpectExec(regexp.QuoteMeta(updateExpected)).WillReturnResult(sqlmock.NewResult(0, 1))
	gTok, err := Get(tok1)
	if err != nil {
		t.Errorf("error: %s", err)
	}
	if gTok.Username != "user1" {
		t.Errorf("Expected username user1. Got %s", gTok.Username)
	}
	// check validity checks
	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(selectExpected)).WithArgs(stringutils.ToSHA1(tok1), 1).WillReturnRows(sqlmock.NewRows([]string{"id", "token", "username", "lastaccessed"}).AddRow(2, "888f45a334f014f763bc3fb7d0afd24daa6c5e0f", "user1", time.Date(now.Year(), now.Month(), now.Day()-defaultTokenValidity-1, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location()).Unix()))
	gTok, err = Get(tok1)
	if err == nil || err.Error() != "token has expired" {
		t.Errorf("'token has expired' error expected. %s", err)
	}
	// check error
	mock.ExpectQuery(regexp.QuoteMeta(selectExpected)).WithArgs(stringutils.ToSHA1(tok1), 1).WillReturnRows(sqlmock.NewRows([]string{"id", "token", "username", "lastaccessed"}).AddRow(2, "888f45a334f014f763bc3fb7d0afd24daa6c5e0f", "user1", time.Now().Add(-time.Hour*24).UTC().Unix()))
	token.LastAccessed = time.Now().Unix()
	updateExpected, _, _ = db.Get().From("Token").Where(goqu.I("ID").Eq(2)).ToUpdateSql(&token)
	mock.ExpectExec(regexp.QuoteMeta(updateExpected)).WillReturnError(errors.New("error"))
	gTok, err = Get(tok1)
	if err == nil || err.Error() != "error" {
		t.Error("error expected.")
	}
	// check select error
	mock.ExpectQuery(regexp.QuoteMeta(selectExpected)).WithArgs(stringutils.ToSHA1(tok1), 1).WillReturnError(errors.New("select error"))
	gTok, err = Get(tok1)
	if err == nil || err.Error() != "select error" {
		t.Error("error selectExpected.")
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}

}

func TestCreate(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	mock.ExpectExec("INSERT INTO `Token` \\(`lastaccessed`, `token`, `username`, `validity`\\) VALUES \\(\\d+, '[a-z0-9]+', 'user1', 100\\)").WillReturnResult(sqlmock.NewResult(2, 1))
	_, err := Create("user1", 100)
	if err != nil {
		t.Errorf("error: %s", err)
	}
	mock.ExpectExec("INSERT INTO `Token` \\(`lastaccessed`, `token`, `username`, `validity`\\) VALUES \\(\\d+, '[a-z0-9]+', 'user1', 30\\)").WillReturnError(errors.New("error"))
	_, err = Create("user1", 0)
	if err == nil || err.Error() != "error" {
		t.Errorf("error expected %s", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestToken_Delete(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	tok1 := Token{
		ID:    324,
		Token: "sadfsdf",
	}
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `Token` WHERE (`token` = 'sadfsdf')")).WillReturnResult(sqlmock.NewResult(0, 1))
	err := tok1.Delete()
	if err != nil {
		t.Errorf("error: %s", err)
	}
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `Token` WHERE (`token` = 'sadfsdf')")).WillReturnError(errors.New("error"))
	err = tok1.Delete()
	if err == nil || err.Error() != "error" {
		t.Errorf("error expected %s", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}
func TestToken_DeleteAll(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	tok1 := Token{
		ID:       324,
		Token:    "sadfsdf",
		Username: "user1",
	}
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `Token` WHERE (`username` = 'user1')")).WillReturnResult(sqlmock.NewResult(0, 1))
	err := tok1.DeleteAll()
	if err != nil {
		t.Errorf("error: %s", err)
	}
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `Token` WHERE (`username` = 'user1')")).WillReturnError(errors.New("error"))
	err = tok1.DeleteAll()
	if err == nil || err.Error() != "error" {
		t.Errorf("error expected %s", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}
