package stringutils

import (
	"gopkg.in/stretchr/testify.v1/assert"
	"testing"
)

func TestStringList_Scan(t *testing.T) {
	strList := &StringList{}
	assert := assert.New(t)
	err := strList.Scan("hello,world,boo")
	assert.Nil(err)
	assert.Contains(*strList, "hello")
	assert.Contains(*strList, "boo")
	assert.Contains(*strList, "world")
	assert.Len(*strList, 3)
}

func TestStringList_String(t *testing.T) {
	strList := &StringList{}
	assert := assert.New(t)
	err := strList.Scan("hello,world,boo")
	assert.Nil(err)
	assert.Equal("hello,world,boo", strList.String())
}
