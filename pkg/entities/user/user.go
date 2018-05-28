package user

import (
	"context"
	"net/mail"

	"github.com/haisum/smpp-app/pkg/entities/user/permission"
	"github.com/haisum/smpp-app/pkg/errs"
	"github.com/pkg/errors"
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

// Store is interface for user store
type Store interface {
	Add(user *User) (int64, error)
	Update(user *User, passwdChanged bool) error
	Get(v interface{}) (*User, error)
	List(c Criteria) ([]User, error)
}

// Authenticator validates username and password of a user and returns user if found and error otherwise
type Authenticator interface {
	Authenticate(username, password string) (*User, error)
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

// Can checks if user has permission to perform given actions
func (u *User) Can(actions ...string) bool {
	if u.Suspended {
		return false
	}
	if len(actions) == 1 && actions[0] == "" {
		return true
	}
	for _, action := range actions {
		canDo := false
		for _, permission := range u.Permissions {
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

// Validate performs sanity checks on User data
func (u *User) Validate() error {
	errMap := make(map[string]string)
	if len(u.Username) < 4 {
		errMap["Username"] = "username must be 4 characters or more"
	}
	if len(u.Password) < 6 {
		errMap["Password"] = "password must be 6 characters or more"
	}
	_, err := mail.ParseAddress(u.Email)
	if err != nil {
		errMap["Email"] = "invalid email address"
	}
	err = u.Permissions.Validate()
	if err != nil {
		errMap["Permissions"] = err.Error()
	}
	if len(errMap) > 0 {
		return &errs.ValidationError{
			Message: "validation failed",
			Errors:  errMap,
		}
	}
	return nil
}

// context
var contextKey = "user"

// FromContext returns a defaultLogger with context
// @todo write tests
func FromContext(ctx context.Context) (*User, error) {
	user, ok := ctx.Value(contextKey).(*User)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	return user, nil
}

// NewContext creates a new context containing defaultLogger
// @todo write tests
func NewContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, contextKey, user)
}
