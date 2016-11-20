package models

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/user"
	log "github.com/Sirupsen/logrus"
	r "github.com/dancannon/gorethink"
	"golang.org/x/crypto/bcrypt"
)

// User contains data for a single user
type User struct {
	ID              string `gorethink:"id,omitempty"`
	Username        string
	Password        string
	Name            string
	Email           string
	ConnectionGroup string
	Permissions     []user.Permission
	RegisteredAt    int64
	Suspended       bool
}

// UserCriteria is used to filter users
type UserCriteria struct {
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
	PerPage          int
	Permissions      []user.Permission
}

const (
	//DefaultConnectionGroup is set for each user who doesn't specifically specify a group
	DefaultConnectionGroup string = "Default"
	//RegisteredAt is time at which user got inserted into system
	RegisteredAt = "RegisteredAt"
	//ASC is used in criteria to sort in ascending order, anything else will be assumed to be descending
	ASC = "ASC"
)

// Add adds a user to database and returns its primary key
func (u *User) Add() (string, error) {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Fatal("Couldn't get session")
	}
	verrors, err := u.Validate()
	if err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"errors": verrors,
		}).Error("Invalid user data supplied to Add.")
		return "", err
	}
	if UserExists(u.Username) {
		return "", fmt.Errorf("User already exists.")
	}
	u.Password, err = hash(u.Password)
	if err != nil {
		log.WithError(err).Error("Couldn't hash.")
		return "", fmt.Errorf("Couldn't hash password. %s", err)
	}
	if u.ConnectionGroup == "" {
		u.ConnectionGroup = DefaultConnectionGroup
	}
	w, err := r.DB(db.DBName).Table("User").Insert(u).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
			"w":   jsonPrint(w),
		}).Error("Couldn't insert")
		return "", err
	}
	if w.Inserted != 1 {
		log.Error("Insert count not as expected.")
		return "", fmt.Errorf("Query was succesful but unexpected number of users inserted.")
	}
	return w.GeneratedKeys[0], nil
}

// Update updates an existing user
func (u *User) Update(passwdChanged bool) error {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Fatal("Couldn't get session")
	}
	verrors, err := u.Validate()
	if err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"errors": verrors,
		}).Error("Invalid user data supplied to Update.")
		return err
	}
	if passwdChanged {
		u.Password, err = hash(u.Password)
		if err != nil {
			log.WithError(err).Error("Couldn't hash.")
			return fmt.Errorf("Couldn't hash password. %s", err)
		}
	}
	w, err := r.DB(db.DBName).Table("User").Get(u.ID).Update(u).RunWrite(s)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
			"w":   jsonPrint(w),
		}).Error("Couldn't update.")
		return err
	}
	return nil
}

// GetUser gets a single user identified by username
func GetUser(username string) (User, error) {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Fatal("Couldn't get session")
	}
	var u User
	cur, err := r.DB(db.DBName).Table("User").Filter(r.Row.Field("Username").Eq(username)).Run(s)
	if err != nil {
		log.WithError(err).Error("Couldn't get user.")
		return u, err
	}
	defer cur.Close()
	cur.One(&u)
	defer cur.Close()
	return u, nil
}

// GetIDUser gets a single user identified by an id
func GetIDUser(id string) (User, error) {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Fatal("Couldn't get session")
	}
	var u User
	cur, err := r.DB(db.DBName).Table("User").Get(id).Run(s)
	defer cur.Close()
	if err != nil {
		log.WithError(err).Error("Couldn't get user.")
		return u, err
	}
	cur.One(&u)
	defer cur.Close()
	return u, nil
}

// GetUsers filters users by a criteria and returns filtered users
func GetUsers(c UserCriteria) ([]User, error) {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Fatal("Couldn't get session")
	}
	var users []User
	log.WithField("Criteria", c).Info("Making query.")
	t := r.DB(db.DBName).Table("User")
	if c.ConnectionGroup != "" {
		t.Filter(r.Row.Field("ConnectionGroup").Eq(c.ConnectionGroup))
	}
	if len(c.Permissions) > 0 {
		for _, perm := range c.Permissions {
			t = t.Filter(r.Row.Field("Permissions").Contains(perm))
		}
	}
	if c.RegisteredAfter > 0 && c.RegisteredBefore > 0 {
		t = t.Between(c.RegisteredAfter, c.RegisteredBefore, r.BetweenOpts{
			Index: "RegisteredAt",
		})
	}
	if c.RegisteredAfter > 0 {
		t = t.Filter(r.Row.Field("RegisteredAt").Gt(c.RegisteredAfter))
	}
	if c.RegisteredBefore > 0 {
		t = t.Filter(r.Row.Field("RegisteredAt").Lt(c.RegisteredBefore))
	}
	if c.Username != "" {
		t = t.Filter(r.Row.Field("Username").Eq(c.Username))
	}
	if c.Email != "" {
		t = t.Filter(r.Row.Field("Email").Eq(c.Email))
	}
	if c.Name != "" {
		t = t.Filter(r.Row.Field("Name").Match(c.Name))
	}
	if c.Suspended == true {
		t = t.Filter(r.Row.Field("Suspended").Eq(c.Suspended))
	}

	// See https://rethinkdb.com/blog/beerthink/
	var order func(args ...interface{}) r.Term
	if strings.ToUpper(c.OrderByDir) == ASC {
		order = r.Asc
	} else {
		order = r.Desc
	}
	key := c.OrderByKey
	if key == "" {
		key = "RegisteredAt"
	}
	if c.PerPage == 0 {
		c.PerPage = 100
	}
	if c.From != "" {
		if c.OrderByDir == ASC {
			if c.OrderByKey == RegisteredAt {
				var from int64
				from, err = strconv.ParseInt(c.From, 10, 64)
				if err != nil {
					return users, fmt.Errorf("Invalid value for RegisteredAt")
				}
				t = t.Between(from, r.MaxVal, r.BetweenOpts{
					Index:     key,
					LeftBound: "open",
				})
			} else {
				t = t.Between(c.From, r.MaxVal, r.BetweenOpts{
					Index:     key,
					LeftBound: "open",
				})
			}
		} else {
			if c.OrderByKey == RegisteredAt {
				var upto int64
				upto, err = strconv.ParseInt(c.From, 10, 64)
				if err != nil {
					return users, fmt.Errorf("Invalid value for RegisteredAt")
				}
				t = t.Between(r.MinVal, upto, r.BetweenOpts{
					Index:     key,
					LeftBound: "open",
				})
			} else {
				t = t.Between(r.MinVal, c.From, r.BetweenOpts{
					Index:     key,
					LeftBound: "open",
				})
			}
		}
	}
	t = t.OrderBy(order(key)).Limit(c.PerPage)
	log.WithField("Query", t.String()).Info("Fetching users.")
	cur, err := t.Run(s)
	if err != nil {
		log.WithError(err).Error("Error in filtering")
		return users, err
	}
	err = cur.All(&users)
	if err != nil {
		log.WithError(err).Error("Error in loading users")
		return users, err
	}
	defer cur.Close()
	for i := range users {
		users[i].Password = ""
	}
	return users, nil
}

// UserExists checks if another user with same username exists
func UserExists(username string) bool {
	s, err := db.GetSession()
	if err != nil {
		log.WithError(err).Fatal("Couldn't get session")
	}
	f := map[string]string{"Username": username}
	cur, err := r.DB(db.DBName).Table("User").Filter(f).Count().Run(s)
	if err != nil {
		log.WithError(err).Error("Error from rethink")
		return false
	}
	var count int
	cur.One(&count)
	defer cur.Close()
	if count > 0 {
		return true
	}
	return false
}

// Validate performs sanity checks on User data
func (u *User) Validate() (map[string]string, error) {
	errors := make(map[string]string)
	if len(u.Username) < 4 {
		errors["Username"] = "Username must be 4 characters or more."
	}
	if len(u.Password) < 6 {
		errors["Password"] = "Password must be 6 characters or more."
	}
	_, err := mail.ParseAddress(u.Email)
	if err != nil {
		errors["Email"] = "Invalid email address"
	}
	for _, x := range u.Permissions {
		var isValidPerm bool
		for _, y := range user.GetPermissions() {
			if x == y {
				isValidPerm = true
				break
			}
		}
		if !isValidPerm {
			errors["Permissions"] = "Invalid permissions."
			break
		}
	}
	if len(errors) > 0 {
		return errors, fmt.Errorf("Validation failed")
	}
	return errors, nil
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

func jsonPrint(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b[:])
}
