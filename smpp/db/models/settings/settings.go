package settings

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/doug-martin/goqu.v3"
)

// Get gets value against a name from settings table
func Get(name string) (string, error) {
	var val string
	found, err := db.Get().From("settings").Select("value").Where(goqu.I("name").Eq(name)).ScanVal(&val)
	if !found && err == nil {
		err = fmt.Errorf("Setting %s not found in db.", name)
	}
	if err != nil {
		log.WithError(err).Error("Couldn't get setting.")
	}
	return val, err
}

// Set sets value against a name from settings table
func Set(name, value string) error {
	_, err := db.Get().From("settings").Where(goqu.I("name").Eq(name)).Delete().Exec()
	if err != nil {
		err = fmt.Errorf("Couldn't delete from db. %s", err)
		return err
	}
	_, err = db.Get().From("settings").Insert(goqu.Record{"name": name, "value": value}).Exec()
	return err
}
