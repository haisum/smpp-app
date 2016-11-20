package models

import (
	"strings"

	r "github.com/dancannon/gorethink"
)

func filterBetweenInt(fields map[string]map[string]int64, t r.Term) (r.Term, bool) {
	var filtered bool
	for field, vals := range fields {
		if vals["after"] > 0 && vals["before"] > 0 {
			t = t.Between(vals["after"], vals["before"], r.BetweenOpts{
				Index:      field,
				RightBound: "closed",
			})
			filtered = true
		} else {
			if vals["after"] > 0 {
				t = t.Between(vals["after"], r.MaxVal, r.BetweenOpts{
					Index:      field,
					RightBound: "closed",
				})
				filtered = true
			}
			if vals["before"] > 0 {
				t = t.Between(r.MinVal, vals["before"], r.BetweenOpts{
					Index:      field,
					RightBound: "closed",
				})
				filtered = true
			}
		}
	}
	return t, filtered
}

func filterEqStr(fields map[string]string, t r.Term) (r.Term, bool) {
	var filtered bool
	for field, val := range fields {
		if val != "" {
			t = t.Filter(map[string]string{field: val})
			filtered = true
		}
	}
	return t, filtered
}

func orderBy(key, dir string, from interface{}, t r.Term, indexUsed, filterUsed bool) r.Term {
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
		indexUsed = true
	}
	if !indexUsed && !filterUsed {
		t = t.OrderBy(r.OrderByOpts{
			Index: order(key),
		})
	} else {
		t = t.OrderBy(order(key))
	}
	return t
}
