package user

import (
	"context"
	"net/mail"

	"bitbucket.org/codefreak/hsmpp/smpp/routes/user/permission"
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

type userStorer interface {
	Add(user *User, hash func(string) (string, error)) (int64, error)
	Update(user *User, hash func(string) (string, error), passwdChanged bool) error
	Get(v interface{}) (*User, error)
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

// validationError is returned when data validation fails for user
type validationError struct {
	Errors  map[string]string
	Message string
}

// Error implements Error interface
func (v *validationError) Error() string {
	return v.Message
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
		return &validationError{
			Message: "validation failed",
			Errors:  errs,
		}
	}
	return nil
}

// context
var contextKey = "user"

// FromContext returns a defaultLogger with context
// @todo write tests
func fromContext(ctx context.Context) (*User, error) {
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
