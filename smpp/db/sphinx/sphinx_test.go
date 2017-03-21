package sphinx

import (
	"testing"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestSetNamesUtf8(t *testing.T) {
	db, mock, _ := ConnectMock(t)
	defer db.Db.Close()
	mock.ExpectExec("SET NAMES utf8").WillReturnResult(sqlmock.NewResult(0, 0))
	// now we execute our method
	if err := setNamesUtf8(); err != nil {
		t.Errorf("error was not expected while executing set names: %s", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil{
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}
