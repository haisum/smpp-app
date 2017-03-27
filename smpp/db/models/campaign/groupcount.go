package campaigns

//GroupCount is data structure to save results of .group(field).count() queries.
type GroupCount struct {
	Name  string `gorethink:"group"`
	Count int64  `gorethink:"reduction"`
}
