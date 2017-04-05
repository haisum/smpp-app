package settings

import (
	"testing"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gopkg.in/stretchr/testify.v1/assert"
	"regexp"
	"errors"
)

func TestGet(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	assert := assert.New(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `settings` WHERE (`name` = 'foo') LIMIT 1")).WillReturnRows(sqlmock.NewRows([]string{"val"}).AddRow("boo"))
	val, err := Get("foo")
	assert.Nil(err)
	assert.Equal("boo", val)
	//check error when no rows
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `settings` WHERE (`name` = 'foo') LIMIT 1")).WillReturnRows(sqlmock.NewRows([]string{"val"}))
	_, err = Get("foo")
	assert.NotNil(err)
	//check error when error
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `settings` WHERE (`name` = 'foo') LIMIT 1")).WillReturnError(errors.New("error"))
	_, err = Get("foo")
	assert.NotNil(err)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations %s", err)
	}
}
