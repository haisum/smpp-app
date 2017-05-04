package sphinx

import (
	"database/sql"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/go-sql-driver/mysql"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	goqu "gopkg.in/doug-martin/goqu.v3"
	_ "gopkg.in/doug-martin/goqu.v3/adapters/mysql"
	"testing"
)

var db *goqu.Database

func Connect(host string, port int) (*goqu.Database, error) {
	config := mysql.Config{
		Addr:              fmt.Sprintf("%s:%d", host, port),
		Net:               "tcp",
		InterpolateParams: true,
	}
	log.WithField("dsn", config.FormatDSN()).Info("Connecting")
	con, err := sql.Open("mysql", config.FormatDSN())
	err = con.Ping()
	if err != nil {
		return db, err
	}
	db = goqu.New("mysql", con)
	if err != nil {
		return db, err
	}
	setNamesUtf8()
	return db, err
}

func setNamesUtf8() error {
	_, err := db.Exec("SET NAMES utf8")
	if err != nil {
		log.WithError(err).Error("Couldn't run SET NAMES utf8")
	}
	return err
}

// ConnectMock makes a mock db connection for testing purposes
func ConnectMock(t *testing.T) (*goqu.Database, sqlmock.Sqlmock, error) {
	con, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
		t.Fail()
	}
	db = goqu.New("msyql", con)
	return db, mock, err
}

func Get() *goqu.Database {
	return db
}
