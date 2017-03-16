package sphinx

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

func Connect(host, port string) (*sqlx.DB, error) {
	config := mysql.Config{
		Addr: fmt.Sprintf("%s:%s", host, port),
		Net:  "tcp",
		InterpolateParams: true,
	}
	log.WithField("dsn", config.FormatDSN()).Info("Connecting")
	var err error
	db, err = sqlx.Connect("mysql", config.FormatDSN())
	if err != nil {
		return db, err
	}
	_, err = db.Exec("SET NAMES utf8")
	if err != nil {
		log.WithError(err).Error("Couldn't run SET NAMES utf8")
	}
	return db, err
}

func Get() *sqlx.DB {
	return db
}