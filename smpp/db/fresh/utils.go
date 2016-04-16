package fresh

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
	"golang.org/x/crypto/bcrypt"
)

func hash(pass string) (string, error) {
	password := []byte(pass)
	// Hashing the password with the default cost of 10
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		log.WithFields(log.Fields{
			"err":      err,
			"password": pass,
		}).Error("Couldn't hash password")
		return "", err
	}
	return string(hashedPassword[:]), nil
}

func jsonPrint(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b[:])
}

func createIndexes(s *r.Session, dbname, table string, indexes []string) error {
	for _, index := range indexes {
		err := r.DB(dbname).Table(table).IndexCreate(index).Exec(s)
		if err != nil {
			log.WithFields(log.Fields{
				"Error": err,
				"Index": index,
				"Table": table,
			}).Error("Couldn't create index.")
			return err
		}
	}
	return nil
}
