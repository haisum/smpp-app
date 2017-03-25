package models

import (
	"fmt"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/doug-martin/goqu.v3"
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
)

const (
	// DefaultTokenValidity is default No. of days token is valid for if unaccessed
	DefaultTokenValidity int = 30
	// TokenSize is length of token
	TokenSize int = 40
)

// Token represents a token given produced against valid authentication request
type Token struct {
	ID           int64 `db:"id" goqu:"skipinsert"`
	LastAccessed int64 `db: "lastaccessed"`
	Token        string `db: "token"`
	Username     string `db: "username"`
	Validity     int `db: "validity"`
}

// GetToken looks for token in Token table and returns it or error if
// it's not found.
func GetToken(token string) (Token, error) {
	var t Token
	found, err := db.Get().From("Token").Select("Token").Where(goqu.I("Token").Eq(stringutils.ToSHA1(token))).ScanVal(&t)
	if err != nil || !found {
		log.WithFields(log.Fields{
			"err":   err,
			"found" : found,
		}).Error("Error occured while getting token.")
		return t, err
	}
	now := time.Now()
	if t.Validity == 0 {
		t.Validity = DefaultTokenValidity
	}
	// TokenValidity days ago
	then := time.Date(now.Year(), now.Month(), now.Day()-t.Validity, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location()).Unix()
	// if token was accessed an year ago, delete it and return error.
	if t.LastAccessed < then {
		return t, fmt.Errorf("Token has expired.")
	}
	//renew token last accessed
	t.LastAccessed = now.Unix()
	result, err := db.Get().From("Token").Update(t).Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
		}).Error("Error occured while updating last accessed of token.")
		return t, err
	}
	if affected, err := result.RowsAffected(); affected != 1 || err != nil {
		log.WithFields(log.Fields{
			"affected" : affected,
			"err" : err,
		}).Error("Error occured getting last affected")
	}
	return t, err
}

// CreateToken should be called to create a new token for a user
func CreateToken(username string, validity int) (string, error) {
	token := stringutils.SecureRandomAlphaString(TokenSize)
	if validity == 0 {
		validity = DefaultTokenValidity
	}
	t := Token{
		Token:        stringutils.ToSHA1(token),
		LastAccessed: time.Now().UTC().Unix(),
		Username:     username,
		Validity:     validity,
	}
	_, err := db.Get().From("Token").Insert(t).Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
		}).Error("Error occured while inserting token.")
		return "", err
	}
	return token, nil
}

// Delete deletes a previously created token
// This may be called when user logs out
func (t *Token) Delete() error {
	_, err := db.Get().From("Token").Where(goqu.I("token").Eq(t.Token)).Delete().Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
		}).Error("Error occured while deleting token.")
	}
	return err
}

// DeleteAll deletes all tokens of which have same username as this token
// This may be called when user changes password/get suspended or wants
// to logout from all devices.
func (t *Token) DeleteAll() error {
	_, err := db.Get().From("Token").Where(goqu.I("username").Eq(t.Username)).Delete().Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
		}).Error("Error occured while deleting tokens.")
	}
	return err
}
