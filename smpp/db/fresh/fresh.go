package fresh

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

func Create(s *r.Session, dbname string) error {
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
	err = tuser(s, dbname)
	if err != nil {
		return err
	}
	return nil
}

func tuser(s *r.Session, dbname string) error {
	_, err := r.DB(dbname).TableCreate("User").RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"name":  dbname,
			"table": "User",
		}).Error("Error occured in creating table.")
		return err
	}
	err = r.DB(dbname).Table("User").IndexCreate("Username").Exec(s)
	if err != nil {
		log.WithError(err).Error("Couldn't create Username index.")
		return err
	}
	err = r.DB(dbname).Table("User").IndexCreate("RegisteredAt").Exec(s)
	if err != nil {
		log.WithError(err).Error("Couldn't create RegisteredAt index.")
		return err
	}
	err = r.DB(dbname).Table("User").IndexCreate("ConnectionGroup").Exec(s)
	if err != nil {
		log.WithError(err).Error("Couldn't create ConnectionGroup index.")
		return err
	}
	err = r.DB(dbname).Table("User").IndexCreate("Permissions").Exec(s)
	if err != nil {
		log.WithError(err).Error("Couldn't create Permissions index.")
	}
	return err
}

func Drop(s *r.Session, name string) error {
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

func Exists(s *r.Session, name string) bool {
	cur, err := r.DBList().Run(s)
	if err != nil {
		log.WithError(err).Fatal("Couldn't get database list.")
		return false
	}
	var dbs []string
	cur.All(&dbs)
	for _, db := range dbs {
		if db == name {
			return true
		}
	}
	return false
}

func jsonPrint(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b[:])
}
