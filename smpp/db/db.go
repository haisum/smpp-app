package db

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db/fresh"
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

//CheckAndCreateDB Checks if db exists, if not, creates one with basic tables, admin user and indexes
func CheckAndCreateDB() (*goqu.Database, error) {
	var err error
	if !fresh.Exists(db) {
		err = fresh.Create(db)
		if err != nil {
			log.WithError(err).Fatal("Couldn't create database.")
		}
	}
	return db, err
}

func Connect(host string, port int, dbName, user, password string) (*goqu.Database, error) {
	config := mysql.Config{
		Addr:            fmt.Sprintf("%s:%d", host, port),
		Net:             "tcp",
		User:            user,
		Passwd:          password,
		DBName:          dbName,
		MultiStatements: true,
	}
	log.WithField("dsn", config.FormatDSN()).Info("Connecting")
	con, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return db, err
	}
	err = con.Ping()
	if err != nil {
		return db, err
	}
	db = goqu.New("mysql", con)
	db.Logger(log.StandardLogger())
	return db, nil
}

func Get() *goqu.Database {
	return db
}

// ConnectMock makes a mock db connection for testing purposes
func ConnectMock(t *testing.T) (*goqu.Database, sqlmock.Sqlmock, error) {
	con, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
		t.Fail()
	}
	db = goqu.New("mysql", con)
	return db, mock, err
}
