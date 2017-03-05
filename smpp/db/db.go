package db

import (
	"fmt"

	"bitbucket.org/codefreak/hsmpp/smpp/db/fresh"
	log "github.com/Sirupsen/logrus"
	"github.com/go-sql-driver/mysql"
	goqu "gopkg.in/doug-martin/goqu.v3"
	"database/sql"
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

func Connect(host, port, dbName, user, password string) (*goqu.Database, error) {
	config := mysql.Config{
		Addr:   fmt.Sprintf("%s:%s", host, port),
		Net:    "tcp",
		User:   user,
		Passwd: password,
		DBName: dbName,
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
	db = goqu.New("", con)
	db.Logger(log.StandardLogger())
	return db, nil
}

func Get() *goqu.Database {
	return db
}
