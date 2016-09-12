package fresh

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

// Create creates a fresh database, tables, indexes and populates primary data
func Create(s r.QueryExecutor, dbname string) error {
	w, err := r.DBCreate(dbname).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"name": dbname,
		}).Error("Error occured in creating database.")
		return err
	}
	if w.DBsCreated != 1 {
		log.WithFields(log.Fields{
			"DBsCreated":    w.DBsCreated,
			"name":          dbname,
			"WriteResponse": jsonPrint(w),
		}).Error("Error occured in creating database.")
		return fmt.Errorf("Error occured in creating database.")
	}

	if err = tuser(s, dbname); err != nil {
		return err
	}
	if err = ttoken(s, dbname); err != nil {
		return err
	}
	if err = tconfig(s, dbname); err != nil {
		return err
	}
	if err = tmessage(s, dbname); err != nil {
		return err
	}
	if err = tnumfile(s, dbname); err != nil {
		return err
	}
	if err = tcampaign(s, dbname); err != nil {
		return err
	}
	return nil
}

// Drop drops existing database
func Drop(s r.QueryExecutor, name string) error {
	w, err := r.DBDrop(name).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"name": name,
		}).Error("Error occured in droping database.")
		return err
	}
	if w.DBsDropped != 1 {
		log.WithFields(log.Fields{
			"DBsDropped":    w.DBsDropped,
			"name":          name,
			"WriteResponse": jsonPrint(w),
		}).Error("Error occured in dropping database.")
		return fmt.Errorf("Error occured in dropping database.")
	}
	return nil
}

// Exists checks if a database exists
func Exists(s r.QueryExecutor, name string) bool {
	cur, err := r.DBList().Run(s)
	if err != nil {
		log.WithError(err).Fatal("Couldn't get database list.")
		return false
	}
	var dbs []string
	cur.All(&dbs)
	defer cur.Close()
	for _, db := range dbs {
		if db == name {
			return true
		}
	}
	return false
}
