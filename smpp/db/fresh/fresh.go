package fresh

import (
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"bytes"
	goqu "gopkg.in/doug-martin/goqu.v3"
)

const (
	dbValidationQuery  = "select MIN(id) from Message"
	SQLFile = "./sqls/fresh-mysql.sql"
)
// Create creates a fresh database, tables, indexes and populates primary data
func Create(db *goqu.Database) error {
	b, err := ioutil.ReadFile(SQLFile)
	if err != nil {
		log.WithError(err).WithField("SQLFile", SQLFile).Error("Couldn't read file")
		return err
	}
	n := bytes.Index(b, []byte{0})
	_, err = db.Exec(string(b[:n]))
	if err != nil {
		log.WithError(err).WithField("error", err).Error("Couldn't read file")
		return err
	}
	return err
}

// Exists checks if a database exists
func Exists(db *goqu.Database) bool {
	_, err := db.Exec(dbValidationQuery);
	if err != nil {
		log.WithError(err).Error("Error in db validation, db probably isn't created.")
		return false
	}
	return true
}
