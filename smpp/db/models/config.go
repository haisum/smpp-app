package models

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

// GetConfig loads configuration from database table Config
func GetConfig() (smpp.Config, error) {
	var c smpp.Config
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return c, err
	}
	cur, err := r.DB(db.DBName).Table("Config").Run(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("Config").String(),
		}).Error("Couldn't run query.")
		return c, err
	}
	err = cur.One(&c)
	if err != nil {
		log.WithError(err).Error("Error loading config.")
	}
	defer cur.Close()
	return c, err
}

// SetConfig updates configuration and sets it to provided struct
func SetConfig(c smpp.Config) error {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Error("Couldn't get session.")
		return err
	}
	err = r.DB(db.DBName).Table("Config").Update(c).Exec(s)
	if err != nil {
		log.WithFields(log.Fields{
			"Error": err,
			"Query": r.DB(db.DBName).Table("Config").Update(c).String(),
		}).Error("Error in updating config.")
	}
	return err
}
