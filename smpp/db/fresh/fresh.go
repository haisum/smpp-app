package fresh

import (
	"io/ioutil"
	"strings"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
)

const (
	dbValidationQuery = "select MIN(id) from Message"
	SQLFile           = "./sqls/fresh-mysql.sql"
)

// Create creates a fresh database, tables, indexes and populates primary data
func Create(db *db.DB) error {
	b, err := ioutil.ReadFile(SQLFile)
	if err != nil {
		logger.Get().WithError(err).WithField("SQLFile", SQLFile).Error("Couldn't read file")
		return err
	}
	query := stringutils.ByteToString(b)
	replacer := strings.NewReplacer("\n", "", "\r", "")
	_, err = db.Exec(replacer.Replace(query))
	if err != nil {
		logger.Get().WithError(err).WithField("error", err).Error("Couldn't read file")
		return err
	}
	return err
}

// Exists checks if a database exists
func Exists(db *db.DB) bool {
	_, err := db.Exec(dbValidationQuery)
	if err != nil {
		logger.Get().WithError(err).Error("Error in db validation, db probably isn't created.")
		return false
	}
	return true
}
