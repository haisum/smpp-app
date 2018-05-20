package permission

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	"strings"

	"database/sql/driver"

	"gopkg.in/stretchr/testify.v1/assert"
)

func TestGetList(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	// Parse permission.go file in this directory
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Dir(filename)+string(filepath.Separator)+strings.Replace(filename, "_test", "", 1), nil, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	permList := GetList()
	var expPerms []string
	// find out all constants declared in permission.go
	// We assume no other variable or constant will be declared in this file other than Permission
	// Yes I agree this is crazy!
	for _, d := range f.Decls {
		switch d.(type) {
		case *ast.GenDecl:
			gd := d.(*ast.GenDecl)
			for _, s := range gd.Specs {
				switch s.(type) {
				case *ast.ValueSpec:
					vs := s.(*ast.ValueSpec)
					val := strings.Trim(vs.Values[0].(*ast.BasicLit).Value, "\"")
					expPerms = append(expPerms, val)
				}
			}
		}
	}
	// Make sure all constants are returned from GetList
	for _, perm := range expPerms {
		assert.Contains(t, permList, Permission(perm))
	}
}

func TestList_Scan(t *testing.T) {
	l := List{}
	l.Scan("Perm1,Perm2, Perm3,,      ,Perm4")
	assert := assert.New(t)
	assert.Contains(l, Permission("Perm1"))
	assert.Contains(l, Permission("Perm2"))
	assert.Contains(l, Permission("Perm3"))
	assert.Contains(l, Permission("Perm4"))
	assert.Len(l, 4)
}

func TestList_String(t *testing.T) {
	l := List{}
	l.Scan("Perm1,Perm2, Perm3,,      ,Perm4")
	assert := assert.New(t)
	assert.Equal("Perm1,Perm2,Perm3,Perm4", l.String())
}

func TestList_Value(t *testing.T) {
	l := List{}
	l.Scan("Perm1,Perm2, Perm3,,      ,Perm4")
	assert := assert.New(t)
	val, err := l.Value()
	assert.Nil(err)
	_ = driver.Value(val)
	assert.Equal(fmt.Sprintf("%s", val), "Perm1,Perm2,Perm3,Perm4")
}

func TestList_Validate(t *testing.T) {
	l := List{}
	l.Scan(" Perm3,," + ShowConfig + "      ,Perm4," + Mask)
	assert := assert.New(t)
	err := l.Validate()
	assert.Equal("one or more permissions are invalid:Perm3,Perm4", err.Error())
}
