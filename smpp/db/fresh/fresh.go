package fresh

import (
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	log "github.com/Sirupsen/logrus"
	goqu "gopkg.in/doug-martin/goqu.v3"
	"io/ioutil"
	"strings"
)

const (
	dbValidationQuery = "select MIN(id) from Message"
	SQLFile           = "./sqls/fresh-mysql.sql"
)

// Create creates a fresh database, tables, indexes and populates primary data
func Create(db *goqu.Database) error {
	b, err := ioutil.ReadFile(SQLFile)
	if err != nil {
		log.WithError(err).WithField("SQLFile", SQLFile).Error("Couldn't read file")
		return err
	}
	query := stringutils.ByteToString(b)
	replacer := strings.NewReplacer("\n", "", "\r", "")
	_, err = db.Exec(replacer.Replace(query))
	if err != nil {
		log.WithError(err).WithField("error", err).Error("Couldn't read file")
		return err
	}
	return err
}

// Exists checks if a database exists
func Exists(db *goqu.Database) bool {
	_, err := db.Exec(dbValidationQuery)
	if err != nil {
		log.WithError(err).Error("Error in db validation, db probably isn't created.")
		return false
	}
	return true
}
