// Package logger centralizes all logger code for this project
// Logging should be done by methods provided in this package only
// Any external library code for logging should go here so that we have a single place
// to manage logs and related code.
//
// Currently we use github.com/sirupsen/logrus for logging.
package logger

import (
	"context"

	"os"

	"io"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

var loggerKey = "defaultLogger"

// Logger is wrapper on logrus.FieldLogger interface
type Logger interface {
	Info(keyvals ...interface{}) error
	Warn(keyvals ...interface{}) error
	Error(keyvals ...interface{}) error
}

// WithLogger has all methods of Logger and an additional With method
type WithLogger interface {
	Logger
	With(keyvals ...interface{}) WithLogger
}

type defaultLogger struct {
	logger log.Logger
}

var dl Logger

func (l *defaultLogger) Info(keyvals ...interface{}) error {
	return level.Info(l.logger).Log(keyvals...)
}

func (l *defaultLogger) Warn(keyvals ...interface{}) error {
	return level.Warn(l.logger).Log(keyvals...)
}
func (l *defaultLogger) Error(keyvals ...interface{}) error {
	return level.Error(l.logger).Log(keyvals...)
}

func (l *defaultLogger) With(keyvals ...interface{}) WithLogger {
	l.logger = log.With(l.logger, keyvals...)
	return l
}

// Get returns standard defaultLogger for this application
func Get() Logger {
	if dl == nil {
		dl = newLogger(os.Stderr)
	}
	return dl
}

func newLogger(w io.Writer) WithLogger {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(w))
	logger = level.NewFilter(logger, level.AllowAll())
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	return &defaultLogger{logger}
}

// FromContext returns a defaultLogger with context
func FromContext(ctx context.Context) Logger {
	logger, ok := ctx.Value(loggerKey).(Logger)
	if !ok {
		logger = Get()
	}
	return logger
}

// NewContext creates a new context containing defaultLogger
func NewContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}
