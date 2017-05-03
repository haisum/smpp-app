package stringutils

import (
	"fmt"
	"strings"
)

// StringList is a list of strings that's represented as comma separated values in database
type StringList []string

// Scan implements scanner interface
func (s *StringList) Scan(vals interface{}) error {
	sl := strings.Split(fmt.Sprintf("%s", vals), ",")
	for _, v := range sl {
		*s = append(*s, v)
	}
	return nil
}

// String implements String interface
func (s *StringList) String() string {
	var vals []string
	for _, v := range *s {
		vals = append(vals, string(v))
	}
	return strings.Join(vals, ",")
}
