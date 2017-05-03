package campaign

//GroupCount is data structure to save results of .group(field).count() queries.
type GroupCount struct {
	Name  string `db:"name"`
	Count int64  `db:"count"`
}
