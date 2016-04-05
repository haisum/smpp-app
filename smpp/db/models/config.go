package models

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/db"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

// GetConfig loads configuration from database table Config
func GetConfig() (smpp.Config, error) {
	s := db.GetSession()
	cur, err := r.Table("Config").GetAll().Run(s)
	var c smpp.Config
	err = cur.One(&c)
	if err != nil {
		log.WithError(err).Error("Error loading config.")
	}
	return c, err
}

// SetConfig updates configuration and sets it to provided struct
func SetConfig(c smpp.Config) error {
	s := db.GetSession()
	err := r.Table("Config").GetAll().Update(c).Exec(s)
	if err != nil {
		log.WithError(err).Error("Error in updating config.")
	}
	return err
}
