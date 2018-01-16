package token

import (
	"fmt"
	"time"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/user"
	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/doug-martin/goqu.v3"
)

const (
	// defaultTokenValidity is default No. of days token is valid for if unaccessed
	defaultTokenValidity = 30
	// tokenSize is length of token
	tokenSize = 40
)

type tokenStore struct {
	db     *db.DB
	logger logger.Logger
}

// NewStore returns new token store with RDBMS backend
func NewStore(db *db.DB, logger logger.Logger) *tokenStore {
	return &tokenStore{
		db, logger,
	}
}

// Get looks for token in token table and returns it or error if
// it's not found.
func (ts *tokenStore) Get(token string) (user.Token, error) {
	var t user.Token
	found, err := ts.db.From("token").Where(goqu.I("token").Eq(stringutils.ToSHA1(token))).Prepared(true).ScanStruct(&t)
	if err != nil || !found {
		log.WithFields(log.Fields{
			"err":   err,
			"found": found,
		}).Error("Error occured while getting token.")
		return t, err
	}
	now := time.Now()
	if t.Validity == 0 {
		t.Validity = defaultTokenValidity
	}
	// TokenValidity days ago
	then := time.Date(now.Year(), now.Month(), now.Day()-t.Validity, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location()).Unix()
	// if token was accessed an year ago, delete it and return error.
	if t.LastAccessed < then {
		return t, fmt.Errorf("token has expired")
	}
	// renew token last accessed
	t.LastAccessed = now.Unix()
	_, err = ts.db.From("token").Where(goqu.I("ID").Eq(t.ID)).Update(t).Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Error occured while updating last accessed of token.")
	}
	return t, err
}

// Create should be called to create a new token for a user
func (ts *tokenStore) Create(username string, validity int) (string, error) {
	token := stringutils.SecureRandomAlphaString(tokenSize)
	if validity == 0 {
		validity = defaultTokenValidity
	}
	t := user.Token{
		Token:        stringutils.ToSHA1(token),
		LastAccessed: time.Now().UTC().Unix(),
		Username:     username,
		Validity:     validity,
	}
	_, err := ts.db.From("token").Insert(t).Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Error occured while inserting token.")
		return "", err
	}
	return token, nil
}

// Delete deletes a previously created token
// This may be called when user logs out
func (ts *tokenStore) Delete(t *user.Token) error {
	_, err := ts.db.From("token").Where(goqu.I("token").Eq(t.Token)).Delete().Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Error occured while deleting token.")
	}
	return err
}

// DeleteAll deletes all tokens of which have same username as this token
// This may be called when user changes password/get suspended or wants
// to logout from all devices.
func (ts *tokenStore) DeleteAll(t *user.Token) error {
	_, err := ts.db.From("token").Where(goqu.I("username").Eq(t.Username)).Delete().Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Error occured while deleting tokens.")
	}
	return err
}
