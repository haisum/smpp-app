package campaign

// groupCount is data structure to save results of .group(field).count() queries.
type groupCount struct {
	Name  string `db:"name"`
	Count int64  `db:"count"`
}
