package db

import (
	"database/sql"
	"fmt"
	"testing"

	"context"

	"github.com/haisum/smpp-app/pkg/db/fresh"
	"github.com/haisum/smpp-app/pkg/logger"
	"github.com/go-sql-driver/mysql"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"gopkg.in/doug-martin/goqu.v3"
	_ "gopkg.in/doug-martin/goqu.v3/adapters/mysql"
)

// DB type is main connection type that should be passed to
// wherever database related activity is being performed
type DB struct {
	*goqu.Database
	Logger logger.Logger
	Ctx    context.Context
}

// CheckAndCreateDB Checks if db exists, if not, creates one with basic tables, admin user and indexes
func CheckAndCreateDB(db *DB) (*DB, error) {
	var err error
	if !fresh.Exists(db.Database, db.Logger) {
		err = fresh.Create(db.Database, db.Logger)
		if err != nil {
			db.Logger.Error("error", err, "msg", "couldn't create database")
		}
	}
	return db, err
}

// Connect connects to a database
// context can be supplied to give a connection timeout
func Connect(ctx context.Context, host string, port int, dbName, user, password string) (*DB, error) {
	config := mysql.Config{
		Addr:            fmt.Sprintf("%s:%d", host, port),
		Net:             "tcp",
		User:            user,
		Passwd:          password,
		DBName:          dbName,
		MultiStatements: true,
	}
	ctxLogger := logger.FromContext(ctx)
	ctx = logger.NewContext(ctx, ctxLogger.(logger.WithLogger).With("host", host, "dbName", dbName, "user", user, "port", port))

	db := &DB{
		Ctx:    ctx,
		Logger: logger.FromContext(ctx),
	}
	db.Logger.Info("dsn", config.FormatDSN(), "msg", "Connecting")
	if myLogger, ok := db.Logger.(logger.PrintLogger); ok {
		if myWithLogger, okWith := db.Logger.(logger.WithLogger); okWith {
			myLogger = myWithLogger.With("package", "mysql").(logger.PrintLogger)
		}
		mysql.SetLogger(myLogger)
	}
	con, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return db, err
	}
	err = con.PingContext(ctx)
	if err != nil {
		return db, err
	}
	db.Database = goqu.New("mysql", con)
	return db, nil
}

// ConnectMock makes a mock db connection for testing purposes
// it uses context.Background for context
func ConnectMock(t *testing.T) (*DB, sqlmock.Sqlmock, error) {
	return ConnectMockContext(context.Background(), t)
}

// ConnectMockContext makes a mock db connection for testing purposes
// You may use ctx to supply custom logger
func ConnectMockContext(ctx context.Context, t *testing.T) (*DB, sqlmock.Sqlmock, error) {
	con, mock, err := sqlmock.New()
	if err != nil {
		t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
		t.Fail()
	}
	db := &DB{
		Ctx:      ctx,
		Database: goqu.New("mysql", con),
		Logger:   logger.FromContext(ctx),
	}
	return db, mock, err
}
