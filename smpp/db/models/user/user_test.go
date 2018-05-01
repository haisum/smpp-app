package user

import (
	"testing"

	"regexp"

	"bitbucket.org/codefreak/hsmpp/smpp/db"
	"bitbucket.org/codefreak/hsmpp/smpp/db/models/user/permission"
	"github.com/pkg/errors"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gopkg.in/doug-martin/goqu.v3"
	"gopkg.in/stretchr/testify.v1/assert"
)

func TestExists(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	expected, _, _ := con.From("User").Where(goqu.I("username").Eq("user1")).Select(goqu.L("COUNT(*)").As("count")).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(10))
	assert := assert.New(t)
	exists := Exists(con, "user1")
	assert.True(exists)

	expected, _, _ = con.From("User").Where(goqu.I("username").Eq("user2")).Select(goqu.L("COUNT(*)").As("count")).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
	exists = Exists(con, "user2")
	assert.False(exists)

	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnError(errors.New("error"))
	exists = Exists(con, "user2")
	assert.False(exists)

	assert.Nil(mock.ExpectationsWereMet())
}

func TestGet(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	assert := assert.New(t)
	expUser := User{
		ID: 2,
	}
	expected, _, _ := con.From("User").Select(&expUser).Where(goqu.I("username").Eq("user1")).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
	user, err := Get(con, "user1")
	assert.Nil(err)
	assert.Equal(expUser, user)

	expected, _, _ = con.From("User").Select(&expUser).Where(goqu.I("id").Eq(2)).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	_, err = Get(con, expUser.ID)
	assert.EqualError(err, "user not found")

	_, err = Get(con, 2.00)
	assert.EqualError(err, "unsupported argument for user.Get. Expected string or int64")

	assert.Nil(mock.ExpectationsWereMet())
}

func TestList(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	assert := assert.New(t)
	cr := Criteria{
		Username:         "user1",
		Email:            "email@email",
		From:             "10",
		Name:             "myName",
		RegisteredAfter:  1000,
		RegisteredBefore: 2000,
		ConnectionGroup:  "default",
		Suspended:        true,
		OrderByDir:       "ASC",
	}
	expUsers := []User{
		{
			ID: 1,
		},
		{
			ID: 2,
		},
	}
	expected, _, _ := con.From("User").Select(&expUsers[0]).Where(goqu.I("ConnectionGroup").Eq(cr.ConnectionGroup), goqu.I("RegisteredAfter").Gte(cr.RegisteredAfter), goqu.I("RegisteredBefore").Lte(cr.RegisteredBefore), goqu.I("Username").Eq(cr.Username), goqu.I("Email").Eq(cr.Email), goqu.I("Name").Eq(cr.Name), goqu.I("suspended").Is(true), goqu.I("RegisteredAt").Gt(10)).Order(goqu.I("RegisteredAt").Asc()).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1).AddRow(2))
	users, err := List(con, cr)
	assert.Nil(err)
	assert.Equal(expUsers, users)
	assert.Nil(mock.ExpectationsWereMet())
}

func TestUser_Add(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	assert := assert.New(t)
	user1 := User{
		ID: 2,
	}
	_, err := user1.Add(con)
	assert.EqualError(err, "validation failed")

	user1 = User{
		Username:    "hello",
		Password:    "password",
		Email:       "email@email",
		Permissions: permission.List{permission.Mask},
	}
	expected, _, _ := con.From("User").Where(goqu.I("username").Eq(user1.Username)).Select(goqu.L("COUNT(*)").As("count")).ToSql()
	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))
	_, err = user1.Add(con)
	assert.EqualError(err, "user already exists")

	mock.ExpectQuery(regexp.QuoteMeta(expected)).WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
	mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	id, err := user1.Add(con)
	assert.Nil(err)
	assert.Equal(int64(1), id)
	assert.True(hashMatch(user1.Password, "password"))

	assert.Nil(mock.ExpectationsWereMet())
}

func TestUser_Auth(t *testing.T) {
	pass := "samplepasws15"
	hashPass, _ := hash(pass)
	user1 := User{
		Password: hashPass,
	}
	assert.True(t, user1.Auth(pass))
}

func TestUser_Update(t *testing.T) {
	con, mock, _ := db.ConnectMock(t)
	defer con.Db.Close()
	assert := assert.New(t)
	user1 := User{
		Username:    "hello",
		Password:    "password",
		Email:       "email@email",
		Permissions: permission.List{permission.Mask},
	}
	expected, _, _ := con.From("User").Where(goqu.I("id").Eq(user1.ID)).ToUpdateSql(&user1)
	mock.ExpectExec(regexp.QuoteMeta(expected)).WillReturnResult(sqlmock.NewResult(0, 1))
	err := user1.Update(con, false)
	assert.Nil(err)
	assert.Equal(user1.Password, "password")

	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
	err = user1.Update(con, true)
	assert.Nil(err)
	assert.True(hashMatch(user1.Password, "password"))
	assert.Nil(mock.ExpectationsWereMet())
}

func TestUser_Validate(t *testing.T) {
	user := User{
		Username: "asd",
		Password: "pass1",
		Email:    "notvalidemail",
		Permissions: permission.List{
			permission.Mask,
			permission.GetStatus,
			permission.Permission("Perm1"),
		},
	}
	err := user.Validate()
	assert.Equal(t, err.(*ValidationError), &ValidationError{
		Message: "validation failed",
		Errors: map[string]string{
			"Username":    "username must be 4 characters or more",
			"Password":    "password must be 6 characters or more",
			"permissions": "one or more permissions are invalid:Perm1",
			"Email":       "invalid email address",
		},
	})
}
