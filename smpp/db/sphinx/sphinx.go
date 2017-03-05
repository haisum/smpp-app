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

var incs  = make(map[string]*autoIncrement, 5)

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
	return db, err
}

func setMaxID(tbl string) error {
	err := db.Get(&incs[tbl].ID, "SELECT MAX(id) FROM " + tbl)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			incs[tbl].ID = 0
			return nil
		}
		return err
	}
	return nil
}

func Get() *sqlx.DB {
	return db
}

func makeInc(tbl string){
	if _, ok := incs[tbl]; !ok {
		incs[tbl] = &autoIncrement{}
		setMaxID(tbl)
	}
}

func Currval(tbl string) int64 {
	makeInc(tbl)
	incs[tbl].Lock()
	defer incs[tbl].Unlock()
	return incs[tbl].ID
}

func Nextval(tbl string) int64 {
	makeInc(tbl)
	incs[tbl].Lock()
	defer incs[tbl].Unlock()
	incs[tbl].ID = incs[tbl].ID + 1
	return incs[tbl].ID
}
