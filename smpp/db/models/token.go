package models

import (
	"bitbucket.com/codefreak/hsmpp/smpp/db"
	"fmt"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
)

const (
	TokenValidity int = 60
)

// Token represents a token given produced against valid authentication request
type Token struct {
	Id           string `gorethink:"id,omitempty"`
	LastAccessed int64
	Token        string
	Username     string
}

func (t *Token) Create(s *r.Session, username string) (string, error) {

}

func (t *Token) Delete(s *r.Session, token string) error {

}

func (t *Token) DeleteAll(s *r.Session, username string) error {

}

func (t *Token) Fetch(s *r.Session, token string) (string, error) {

}
