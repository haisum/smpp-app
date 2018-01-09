package logger

import (
	"testing"

	"context"

	"bytes"

	"bitbucket.org/codefreak/hsmpp/smpp/stringutils"
	"github.com/pkg/errors"
	"gopkg.in/stretchr/testify.v1/assert"
)

func TestGet(t *testing.T) {
	dl = nil
	l := Get()
	_, ok := l.(*defaultLogger)
	assert.True(t, ok)
}

func TestNewContext(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	l := newLogger(buf).(*defaultLogger)
	ctx := NewContext(context.Background(), l.With("mykey", "myvalue"))
	ctxLogger, ok := ctx.Value(loggerKey).(*defaultLogger)
	assert.True(t, ok)
	ctxLogger.Error("error", "hello world")
	logOutput := stringutils.ByteToString(buf.Bytes())
	assert.Contains(t, logOutput, "mykey=myvalue")
	assert.Contains(t, logOutput, "hello world")
}

func TestFromContext(t *testing.T) {
	l := Get()
	ctx := NewContext(context.Background(), l.(WithLogger).With("mykey", "myvalue"))
	ctxLogger, ok := ctx.Value(loggerKey).(*defaultLogger)
	assert.True(t, ok)
	frmLogger, ok := FromContext(ctx).(*defaultLogger)
	assert.True(t, ok)
	assert.Equal(t, ctxLogger, frmLogger)
}

func TestDefaultLogger_WithError(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	l := newLogger(buf).(*defaultLogger)
	l.With("field1", 1).With("field2", 2).Error("msg", "error")
	l.With("error", errors.New("new error")).Info("msg", "info")
	l.With("fields1", "1", "fields2", 3.01).Warn("msg", "warning", "key1", "val1", "key2", "val2")
	output := stringutils.ByteToString(buf.Bytes())
	assert.Contains(t, output, "field1=1 field2=2 msg=error")
	assert.Contains(t, output, "error=\"new error\"")
	assert.Contains(t, output, "fields1=1 fields2=3.01 msg=warning key1=val1 key2=val2")
	assert.Contains(t, output, "level=error")
	assert.Contains(t, output, "level=info")
	assert.Contains(t, output, "level=warn")
}
