package user

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"strconv"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/doug-martin/goqu.v3"
)

// User contains data for a single user
type User struct {
	ID              int64           `db:"id" goqu:"skipinsert"`
	Username        string          `db:"username"`
	Password        string          `db:"password"`
	Name            string          `db:"name"`
	Email           string          `db:"email"`
	ConnectionGroup string          `db:"connectiongroup"`
	Permissions     permission.List `db:"permissions"`
	RegisteredAt    int64           `db:"registeredat"`
	Suspended       bool            `db:"suspended"`
}

// Criteria is used to filter users
type Criteria struct {
	Username         string
	Email            string
	Name             string
	Suspended        bool
	RegisteredAfter  int64
	OrderByKey       string
	OrderByDir       string
	RegisteredBefore int64
	ConnectionGroup  string
	From             string
	PerPage          uint
}

// ValidationError is returned when data validation fails for user
type ValidationError struct {
	Errors  map[string]string
	Message string
}

// Error implements Error interface
func (v *ValidationError) Error() string {
	return v.Message
}

const (
	// DefaultConnectionGroup is set for each user who doesn't specifically specify a group
	DefaultConnectionGroup = "Default"
)

// Add adds a user to database and returns its primary key
func (u *User) Add() (int64, error) {
	err := u.Validate()
	if err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"errors": err.(*ValidationError).Errors,
		}).Error("Invalid user data supplied to Add.")
		return 0, err
	}
	if Exists(u.Username) {
		return 0, fmt.Errorf("user already exists")
	}
	u.Password, err = hash(u.Password)
	if err != nil {
		log.WithError(err).Error("Couldn't hash.")
		return 0, fmt.Errorf("couldn't hash password %s", err)
	}
	if u.ConnectionGroup == "" {
		u.ConnectionGroup = DefaultConnectionGroup
	}
	w, err := db.Get().From("User").Insert(u).Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Couldn't insert")
		return 0, err
	}
	u.ID, err = w.LastInsertId()
	return u.ID, err
}

// Update updates an existing user
func (u *User) Update(passwdChanged bool) error {
	err := u.Validate()
	if err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"errors": err.(*ValidationError).Errors,
		}).Error("Invalid user data supplied to Update.")
		return err
	}
	if passwdChanged {
		u.Password, err = hash(u.Password)
		if err != nil {
			log.WithError(err).Error("Couldn't hash.")
			return fmt.Errorf("couldn't hash password: %s", err)
		}
	}
	_, err = db.Get().From("User").Where(goqu.I("id").Eq(u.ID)).Update(u).Exec()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Couldn't update.")
	}
	return err
}

// Get gets a single user identified by username (if provided string parameter) or user id (if parameter is int64).
func Get(v interface{}) (User, error) {
	var u User
	q := db.Get().From("User")
	switch v.(type) {
	case string:
		q = q.Where(goqu.I("username").Eq(v))
	case int64:
		q = q.Where(goqu.I("id").Eq(v))
	default:
		return u, errors.New("unsupported argument for user.Get. Expected string or int64")
	}
	found, err := q.ScanStruct(&u)
	if err != nil || !found {
		log.WithError(err).WithField("found", found).Error("Couldn't get user.")
		if !found {
			err = errors.New("user not found")
		}
	}
	return u, err
}

// List filters users by a criteria and returns filtered users
func List(c Criteria) ([]User, error) {
	var users []User
	log.WithField("Criteria", c).Info("Making query.")
	t := db.Get().From("User")
	if c.OrderByKey == "" {
		c.OrderByKey = "RegisteredAt"
	}
	var from interface{}
	if c.From != "" {
		if c.OrderByKey == "RegisteredAt" {
			var err error
			from, err = strconv.ParseInt(c.From, 10, 64)
			if err != nil {
				return users, fmt.Errorf("invalid value for from: %s", from)
			}
		} else {
			from = c.From
		}
	}
	if c.ConnectionGroup != "" {
		t = t.Where(goqu.I("ConnectionGroup").Eq(c.ConnectionGroup))
	}
	if c.RegisteredAfter > 0 {
		t = t.Where(goqu.I("RegisteredAfter").Gte(c.RegisteredAfter))
	}
	if c.RegisteredBefore > 0 {
		t = t.Where(goqu.I("RegisteredBefore").Lte(c.RegisteredBefore))
	}
	if c.Username != "" {
		t = t.Where(goqu.I("Username").Eq(c.Username))
	}
	if c.Email != "" {
		t = t.Where(goqu.I("Email").Eq(c.Email))
	}
	if c.Name != "" {
		t = t.Where(goqu.I("Name").Eq(c.Name))
	}
	if c.Suspended == true {
		t = t.Where(goqu.I("Suspended").Eq(c.Suspended))
	}
	if c.PerPage == 0 {
		c.PerPage = 100
	}

	orderDir := "DESC"
	if strings.ToUpper(c.OrderByDir) == "ASC" {
		orderDir = "ASC"
	}
	if c.From != "" {
		if orderDir == "ASC" {
			t = t.Where(goqu.I(c.OrderByKey).Gt(from))
		} else {
			t = t.Where(goqu.I(c.OrderByKey).Lt(from))
		}
	}
	orderExp := goqu.I(c.OrderByKey).Desc()
	if orderDir == "ASC" {
		orderExp = goqu.I(c.OrderByKey).Asc()
	}
	t = t.Order(orderExp)
	if c.PerPage == 0 {
		c.PerPage = 100
	}
	t = t.Limit(c.PerPage)
	queryStr, _, _ := t.ToSql()
	log.WithField("Query", queryStr).Info("Fetching users.")
	err := t.ScanStructs(&users)
	if err != nil {
		log.WithError(err).Error("Error in filtering")
		return users, err
	}
	for i := range users {
		users[i].Password = ""
	}
	return users, nil
}

// Exists checks if another user with same username exists
func Exists(username string) bool {
	count, err := db.Get().From("User").Where(goqu.I("username").Eq(username)).Count()
	if err != nil {
		log.WithError(err).Error("Error in count query.")
		return false
	}
	if count > 0 {
		return true
	}
	return false
}

// Validate performs sanity checks on User data
func (u *User) Validate() error {
	errs := make(map[string]string)
	if len(u.Username) < 4 {
		errs["Username"] = "username must be 4 characters or more"
	}
	if len(u.Password) < 6 {
		errs["Password"] = "password must be 6 characters or more"
	}
	_, err := mail.ParseAddress(u.Email)
	if err != nil {
		errs["Email"] = "invalid email address"
	}
	err = u.Permissions.Validate()
	if err != nil {
		errs["Permissions"] = err.Error()
	}
	if len(errs) > 0 {
		return &ValidationError{
			Message: "validation failed",
			Errors:  errs,
		}
	}
	return nil
}

// Auth authenticates given password against user's password hash
func (u *User) Auth(pass string) bool {
	return hashMatch(u.Password, pass)
}

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

func hashMatch(hash, pass string) bool {
	hashedPassword := []byte(hash)
	password := []byte(pass)
	// Comparing the password with the hash
	err := bcrypt.CompareHashAndPassword(hashedPassword, password)
	return err == nil
}
