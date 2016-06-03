package models

import (
	"crypto/sha1"
	"fmt"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

const (
	// TokenValidity is No. of days token is valid for if unaccessed
	TokenValidity int = 30
	// TokenSize is length of token
	TokenSize int = 40
)

// Token represents a token given produced against valid authentication request
type Token struct {
	ID           string `gorethink:"id,omitempty"`
	LastAccessed int64
	Token        []byte
	Username     string
}

// GetToken looks for token in Token table and returns it or error if
// it's not found.
func GetToken(s *r.Session, token string) (Token, error) {
	var t Token
	cur, err := r.DB(db.DBName).Table("Token").Filter(r.Row.Field("Token").Eq(toSHA1(token))).Run(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"query": r.DB(db.DBName).Table("Token").Filter(r.Row.Field("Token").Eq(toSHA1(token))).String(),
		}).Error("Error occured while getting token.")
		return t, err
	}
	err = cur.One(&t)
	if err != nil {
		log.WithError(err).Error("Couldn't read data from cursor to struct.")
		return t, err
	}
	defer cur.Close()
	now := time.Now()
	// TokenValidity days ago
	then := time.Date(now.Year(), now.Month(), now.Day()-TokenValidity, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location()).Unix()
	// if token was accessed an year ago, delete it and return error.
	if t.LastAccessed < then {
		return t, fmt.Errorf("Token has expired.")
	}
	//renew token last accessed
	t.LastAccessed = now.Unix()
	err = r.DB(db.DBName).Table("Token").Get(t.ID).Update(t).Exec(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"query": r.DB(db.DBName).Table("Token").Get(t.ID).Update(t).String(),
		}).Error("Error occured while updating last accessed of token.")
		return t, err
	}
	return t, err
}

// CreateToken should be called to create a new token for a user
func CreateToken(s *r.Session, username string) (string, error) {
	token := secureRandomAlphaString(TokenSize)
	t := Token{
		Token:        toSHA1(token),
		LastAccessed: time.Now().UTC().Unix(),
		Username:     username,
	}
	err := r.DB(db.DBName).Table("Token").Insert(t).Exec(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"query": r.DB(db.DBName).Table("Token").Insert(t).String(),
		}).Error("Error occured while inserting token.")
		return "", err
	}
	return token, nil
}

// Delete deletes a previously created token
// This may be called when user logs out
func (t *Token) Delete(s *r.Session) error {
	err := r.DB(db.DBName).Table("Token").Get(t.ID).Delete().Exec(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"query": r.DB(db.DBName).Table("Token").Insert(t).String(),
		}).Error("Error occured while deleting token.")
	}
	return err
}

// DeleteAll deletes all tokens of which have same username as this token
// This may be called when user changes password/get suspended or wants
// to logout from all devices.
func (t *Token) DeleteAll(s *r.Session) error {
	err := r.DB(db.DBName).Table("Token").Filter(r.Row.Field("Username").Eq(t.Username)).Delete().Exec(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"query": r.DB(db.DBName).Table("Token").Filter(r.Row.Field("Username").Eq(t.Username)).Delete().Exec(s),
		}).Error("Error occured while deleting tokens.")
	}
	return err
}

func toSHA1(s string) []byte {
	sh := sha1.New()
	sh.Write([]byte(s))
	return sh.Sum(nil)
}
