package settings

import (
	"errors"
	"regexp"
	"testing"

	"bitbucket.org/codefreak/hsmpp/pkg/db"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gopkg.in/doug-martin/goqu.v3"
	"gopkg.in/stretchr/testify.v1/assert"
)

func TestGet(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	assert := assert.New(t)
	expected, _, _ := db.Get().From("settings").Select("value").Where(goqu.I("name").Eq("foo")).Limit(1).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"val"}).AddRow("boo"))
	val, err := Get("foo")
	assert.Nil(err)
	assert.Equal("boo", val)
	// check error when no rows
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"val"}))
	_, err = Get("foo")
	assert.NotNil(err)
	// check error when error
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnError(errors.New("error"))
	_, err = Get("foo")
	assert.NotNil(err)
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations %s", err)
	}
}

func TestSet(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	assert := assert.New(t)
	expected, _, _ := db.Get().From("settings").Where(goqu.I("name").Eq("foo")).ToDeleteSql()
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(0, 1))
	expected, _, _ = db.Get().From("settings").ToInsertSql(goqu.Record{"name": "foo", "value": "boo"})
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(0, 1))
	err := Set("foo", "boo")
	assert.Nil(err)
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations %s", err)
	}
}
