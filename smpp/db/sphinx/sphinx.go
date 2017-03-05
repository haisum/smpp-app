package sphinx

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/go-sql-driver/mysql"
	"database/sql"
	goqu "gopkg.in/doug-martin/goqu.v3"
)

var db *goqu.Database

func Connect(host, port string) (*goqu.Database, error) {
	config := mysql.Config{
		Addr: fmt.Sprintf("%s:%s", host, port),
		Net:  "tcp",
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
	if err != nil {
		return db, err
	}
	return db, err
}
func Get() *goqu.Database {
	return db
}
