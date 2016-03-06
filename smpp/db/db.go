package db

import (
	"bitbucket.com/codefreak/hsmpp/smpp/db/fresh"
	"fmt"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

var (
	s          *r.Session
	DBName     string = "hsmppdb"
	DBTestName string = "hsmpptestdb"
	DBHost     string = "localhost"
	DBPort     int    = 28015
)

// Connect makes a new session to rethinkdb
func Connect() (*r.Session, error) {
	r.SetVerbose(true)
	opt := getOpts()
	var err error
	s, err = r.Connect(opt)
	if err != nil {
		log.WithFields(log.Fields{
			"err":         err,
			"ConnectOpts": opt,
		}).Error("Couldn't connect to rethinkdb.")
	}
	if !fresh.Exists(s, DBName) {
		err = fresh.Create(s, DBName)
		if err != nil {
			log.WithError(err).Fatal("Couldn't create database.")
			return s, err
		}
	}
	return s, err
}

// GetSession returns rethinkdb session created earlier. If there isn't
// already a session, it creates it.
func GetSession() (*r.Session, error) {
	var err error
	if s == nil {
		_, err = Connect()
	}
	return s, err
}

func getOpts() r.ConnectOpts {
	return r.ConnectOpts{
		Address: fmt.Sprintf("%s:%d", DBHost, DBPort),
	}
}
