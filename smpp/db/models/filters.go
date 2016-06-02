package models

import (
	"strings"

	r "github.com/dancannon/gorethink"
)

func filterBetweenInt(fields map[string]map[string]int64, t r.Term) r.Term {
	for field, vals := range fields {
		if vals["after"] > 0 && vals["before"] > 0 {
			t = t.Between(vals["after"], vals["before"], r.BetweenOpts{
				Index: field,
			})
		}
		if vals["after"] > 0 {
			t = t.Filter(r.Row.Field(field).Gt(vals["after"]))
		}
		if vals["before"] > 0 {
			t = t.Filter(r.Row.Field(field).Lt(vals["before"]))
		}
	}
	return t
}

func filterEqStr(fields map[string]string, t r.Term) r.Term {
	for field, val := range fields {
		if val != "" {
			t = t.Filter(map[string]string{field: val})
		}
	}
	return t
}

func orderBy(key, dir string, from interface{}, t r.Term, indexUsed bool) r.Term {
	var order func(args ...interface{}) r.Term
	if strings.ToUpper(dir) == "ASC" {
		order = r.Asc
	} else {
		order = r.Desc
	}
	if from != nil {
		if dir == "ASC" {
			t = t.Between(from, r.MaxVal, r.BetweenOpts{
				Index:     key,
				LeftBound: "open",
			})
		} else {
			t = t.Between(r.MinVal, from, r.BetweenOpts{
				Index:     key,
				LeftBound: "open",
			})
		}
	}
	if !indexUsed {
		t = t.OrderBy(r.OrderByOpts{
			Index: order(key),
		})
	} else {
		t = t.OrderBy(order(key))
	}
	return t
}
