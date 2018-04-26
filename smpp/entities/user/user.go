package user

import (
	"context"
	"net/mail"

	"bitbucket.org/codefreak/hsmpp/smpp/entities/user/permission"
	"bitbucket.org/codefreak/hsmpp/smpp/errs"
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

type UserStorer interface {
	Add(user *User) (int64, error)
	Update(user *User, passwdChanged bool) error
	Get(v interface{}) (*User, error)
}

type Authorizer interface {
	Can(actions ...string) bool
}

type Authenticator interface {
	Authenticate(ctx context.Context, username, password string) (context.Context, Authorizer, error)
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