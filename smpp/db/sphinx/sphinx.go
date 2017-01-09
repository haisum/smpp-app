package sphinx

import (
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

type autoIncrement struct {
	sync.Mutex
	ID int64
}

var autoIncr *autoIncrement

func Connect(host, port string) (*sqlx.DB, error) {
	config := mysql.Config{
		Addr: fmt.Sprintf("%s:%s", host, port),
		Net:  "tcp",
	}
	log.WithField("dsn", config.FormatDSN()).Info("Connecting")
	var err error
	db, err = sqlx.Connect("mysql", config.FormatDSN())
	if err != nil {
		return db, err
	}
	autoIncr = &autoIncrement{}
	err = setMaxID()
	return db, err
}

func setMaxID() error {
	err := db.Get(&autoIncr.ID, "SELECT MAX(id) FROM Message")
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			autoIncr.ID = 0
			return nil
		}
		return err
	}
	return nil
}

func Get() *sqlx.DB {
	return db
}

func Currval() int64 {
	autoIncr.Lock()
	defer autoIncr.Unlock()
	return autoIncr.ID
}

func Nextval() int64 {
	autoIncr.Lock()
	defer autoIncr.Unlock()
	autoIncr.ID = autoIncr.ID + 1
	return autoIncr.ID
}
