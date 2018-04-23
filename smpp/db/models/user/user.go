package user

import (
	"fmt"
	"strings"

	"strconv"

	"context"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/logger"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/middleware"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/user"
	"bitbucket.org/codefreak/hsmpp/smpp/routes/user/permission"
	"github.com/pkg/errors"
	"gopkg.in/doug-martin/goqu.v3"
)

const (
	// defaultConnectionGroup is set for each user who doesn't specifically specify a group
	defaultConnectionGroup = "Default"
)

type userStore struct {
	db     *db.DB
	logger logger.Logger
}

type userAuthenticator struct {
	GetUser func(v interface{}) (*user.User, error)
}

func (ua *userAuthenticator) Authenticate(ctx context.Context, HashMatchFunc func(hash, str string) bool, username, password string) (context.Context, middleware.Authorizer, error) {
	u, err := ua.GetUser(username)
	if err != nil {
		return ctx, nil, errors.Wrap(err, "username or password is wrong")
	}
	if ok := HashMatchFunc(u.Password, password); ok {
		return user.NewContext(ctx, u), &userAuthorizer{
			u.Permissions, u.Suspended,
		}, nil
	}
	return ctx, nil, errors.New("username or password is wrong")
}

type userAuthorizer struct {
	Permissions permission.List
	Suspended   bool
}

func (uaz *userAuthorizer) Can(actions ...string) bool {
	if uaz.Suspended {
		return false
	}
	if len(actions) == 1 && actions[0] == "" {
		return true
	}
	for _, action := range actions {
		canDo := false
		for _, permission := range uaz.Permissions {
			if string(permission) == action {
				canDo = true
			}
		}
		if canDo == false {
			return false
		}
	}
	return true
}

func NewAuthenticator(getUser func(v interface{}) (*user.User, error)) middleware.Authenticator {
	return &userAuthenticator{
		GetUser: getUser,
	}
}

// NewStore returns new user store with RDBMS backend
func NewStore(db *db.DB, logger logger.Logger) *userStore {
	return &userStore{
		db, logger,
	}
}

// Add adds a user to database and returns its primary key
func (us *userStore) Add(user *user.User, hash func(string) (string, error)) (int64, error) {
	err := user.Validate()
	if err != nil {
		return 0, err
	}
	if us.Exists(user.Username) {
		return 0, fmt.Errorf("user already exists")
	}
	user.Password, err = hash(user.Password)
	if err != nil {
		us.logger.Error("error", err, "msg", "couldn't hash")
		return 0, fmt.Errorf("couldn't hash password %s", err)
	}
	if user.ConnectionGroup == "" {
		user.ConnectionGroup = defaultConnectionGroup
	}
	w, err := us.db.From("User").Insert(us).Exec()
	if err != nil {
		return 0, err
	}
	user.ID, err = w.LastInsertId()
	return user.ID, err
}

// Update updates an existing user
func (us *userStore) Update(user *user.User, hashFunc func(string) (string, error), passwdChanged bool) error {
	err := user.Validate()
	if err != nil {
		return err
	}
	if passwdChanged {
		user.Password, err = hashFunc(user.Password)
		if err != nil {
			return errors.Wrap(err, "hash error")
		}
	}
	_, err = us.db.From("User").Where(goqu.I("id").Eq(user.ID)).Update(us).Exec()
	if err != nil {
		return errors.Wrap(err, "update error")
	}
	return err
}

// Get gets a single user identified by username (if provided string parameter) or user id (if parameter is int64).
func (us *userStore) Get(v interface{}) (*user.User, error) {
	u := &user.User{}
	q := us.db.From("User")
	switch v.(type) {
	case string:
		q = q.Where(goqu.I("username").Eq(v))
	case int64:
		q = q.Where(goqu.I("id").Eq(v))
	default:
		return u, errors.New("unsupported argument for user.Get. Expected string or int64")
	}
	_, err := q.ScanStruct(u)
	if err != nil {
		return u, errors.Wrap(err, "user select error")
	}
	return u, err
}

// List filters users by a criteria and returns filtered users
func (us *userStore) List(c user.Criteria) ([]user.User, error) {
	var users []user.User
	t := us.db.From("User")
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
	err := t.ScanStructs(&users)
	if err != nil {
		return users, errors.Wrap(err, "user filter error")
	}
	for i := range users {
		users[i].Password = ""
	}
	return users, nil
}

// Exists checks if another user with same username exists
func (us *userStore) Exists(username string) bool {
	count, err := us.db.From("User").Where(goqu.I("username").Eq(username)).Count()
	if err != nil {
		us.logger.Error("error", err, "msg", "error in count query")
		return false
	}
	if count > 0 {
		return true
	}
	return false
}
