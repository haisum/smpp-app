package fresh

import (
	"io/ioutil"
	"strings"

	"bitbucket.org/codefreak/hsmpp/pkg/logger"
	"bitbucket.org/codefreak/hsmpp/pkg/stringutils"
	"gopkg.in/doug-martin/goqu.v3"
)

const (
	dbValidationQuery = "select MIN(id) from Message"
	SQLFile           = "./sqls/fresh-mysql.sql"
)

// Create creates a fresh database, tables, indexes and populates primary data
func Create(db *goqu.Database, logger logger.Logger) error {
	b, err := ioutil.ReadFile(SQLFile)
	if err != nil {
		logger.Error("error", err, "SQLFile", SQLFile, "msg", "couldn't read file")
		return err
	}
	query := stringutils.ByteToString(b)
	replacer := strings.NewReplacer("\n", "", "\r", "")
	_, err = db.Exec(replacer.Replace(query))
	if err != nil {
		logger.Error("error", err, "msg", "couldn't read file")
		return err
	}
	return err
}

// Exists checks if a database exists
func Exists(db *goqu.Database, logger logger.Logger) bool {
	_, err := db.Exec(dbValidationQuery)
	if err != nil {
		logger.Error("error", err, "msg", "error in db validation, db probably isn't created")
		return false
	}
	return true
}
