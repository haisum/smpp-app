package utils

//Query builder is a simple utility to create complex sql queries
// this is obsolete, please use goqu in future.
type QueryBuilder struct {
	query   string
	where   string
	limit   string
	orderBy string
	groupBy string
}

func (q *QueryBuilder) Select(s string) *QueryBuilder {
	q.query = "SELECT " + s + " FROM "
	return q
}

func (q *QueryBuilder) From(table string) *QueryBuilder {
	if q.query == "" {
		q.query = "SELECT * FROM "
	}
	q.query += table
	return q
}

func (q *QueryBuilder) WhereAnd(condition string) *QueryBuilder {
	if q.where == "" {
		q.where = condition
	} else {
		q.where = q.where + " AND " + condition
	}
	return q
}

func (q *QueryBuilder) WhereOr(condition string) *QueryBuilder {
	if q.where == "" {
		q.where = condition
	} else {
		q.where = q.where + " OR " + condition
	}
	return q
}

func (q *QueryBuilder) Limit(limit string) *QueryBuilder {
	q.limit = limit
	return q
}

func (q *QueryBuilder) OrderBy(orderBy string) *QueryBuilder {
	q.orderBy = orderBy
	return q
}

func (q *QueryBuilder) GroupBy(groupBy string) *QueryBuilder {
	q.groupBy = groupBy
	return q
}

func (q *QueryBuilder) GetQuery() string {
	query := q.query
	if q.where != "" {
		query = query + " WHERE " + q.where
	}
	if q.groupBy != "" {
		query = query + " GROUP BY " + q.groupBy
	}
	if q.orderBy != "" {
		query = query + " ORDER BY " + q.orderBy
	}
	if q.limit != "" {
		query = query + " LIMIT " + q.limit
	}
	return query
}
