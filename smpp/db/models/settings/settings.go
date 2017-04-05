package settings

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/doug-martin/goqu.v3"
	"fmt"
)

// Get gets value against a name from settings table
func Get(name string) (string, error) {
	var val string
	found, err := db.Get().From("settings").Where(goqu.I("name").Eq(name)).ScanVal(&val)
	if !found && err == nil {
		err = fmt.Errorf("Setting %s not found in db.", name)
	}
	if err != nil {
		log.WithError(err).Error("Couldn't get setting.")
	}
	return val, err
}