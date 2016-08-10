package db

import (
	"fmt"

	"bitbucket.org/codefreak/hsmpp/smpp/db/fresh"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

var (
	s *r.Session
	//DBName is rethinkdb name
	DBName = "hsmppdb"
	//DBTestName is db name used for tests
	DBTestName = "hsmpptestdb"
	//DBHost is host address of rethink db
	DBHost = "localhost"
	//DBPort is port to bind connection to
	DBPort = 28015
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
		return s, err
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
