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