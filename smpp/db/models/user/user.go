package user

import (
	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/doug-martin/goqu.v3"
	"net/mail"
	"strings"
)

type permissions []permission.Permission

// User contains data for a single user
type User struct {
	ID              int64       `db:"id,skipinsert"`
	Username        string      `db:"username"`
	Password        string      `db:"password"`
	Name            string      `db:"name"`
	Email           string      `db:"email"`
	ConnectionGroup string      `db:"connectiongroup"`
	Permissions     permissions `db:"permissions"`
	RegisteredAt    int64       `db:"registeredat"`
	Suspended       bool        `db:"suspended"`
}

func (p *permissions) Scan(perms interface{}) error {
	ps := strings.Split(fmt.Sprintf("%s", perms), ",")
	for _, v := range ps {
		*p = append(*p, permission.Permission(v))
	}
	return nil
}

func (p *permissions) String() string {
	var perms []string
	for _, v := range *p {
		perms = append(perms, string(v))
	}
	return strings.Join(perms, ",")
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
	PerPage          uint
	Permissions      permissions
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
func (u *User) Add() (int64, error) {
	verrors, err := u.Validate()
	if err != nil {
		log.WithFields(log.Fields{
			"err":    err,
			"errors": verrors,
		}).Error("Invalid user data supplied to Add.")
		return 0, err
	}
	if Exists(u.Username) {
		return 0, fmt.Errorf("User already exists.")
	}
	u.Password, err = hash(u.Password)
	if err != nil {
		log.WithError(err).Error("Couldn't hash.")
		return 0, fmt.Errorf("Couldn't hash password. %s", err)
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
	return w.LastInsertId()
}

// Update updates an existing user
func (u *User) Update(passwdChanged bool) error {
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
	q := goqu.From("User")
	switch v.(type) {
	case string:
		q = q.Where(goqu.I("username").Eq(v))
	case int64:
		q = q.Where(goqu.I("userid").Eq(v))
	default:
		return u, errors.New("Unsupported argument for user.Get. Expected string or int64")
	}
	found, err := q.ScanStruct(&u)
	if err != nil || !found {
		log.WithError(err).WithField("found", found).Error("Couldn't get user.")
		if !found {
			err = errors.New("User not found")
		}
	}
	return u, err
}

// GetUsers filters users by a criteria and returns filtered users
func GetUsers(c UserCriteria) ([]User, error) {
	var users []User
	log.WithField("Criteria", c).Info("Making query.")
	t := db.Get().From("User")
	if c.ConnectionGroup != "" {
		t = t.Where(goqu.I("ConnectionGroup").Eq(c.ConnectionGroup))
	}
	if len(c.Permissions) > 0 {
		t = t.Where(goqu.I("Permissions").Eq(c.Permissions.String()))
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
		t = t.Where(goqu.I("Name ").Eq(c.Name))
	}
	if c.ConnectionGroup != "" {
		t = t.Where(goqu.I("ConnectionGroup").Eq(c.ConnectionGroup))
	}
	if c.Suspended == true {
		t = t.Where(goqu.I("Suspended").Eq(c.Suspended))
	}
	key := c.OrderByKey
	if key == "" {
		key = "RegisteredAt"
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
			t = t.Where(goqu.I(c.OrderByKey).Gt(c.From))
		} else {
			t = t.Where(goqu.I(c.OrderByKey).Lt(c.From))
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
		for _, y := range permission.GetList() {
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
