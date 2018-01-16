package user

// Token represents a token given produced against valid token request
type Token struct {
	ID           int64  `db:"id" goqu:"skipinsert"`
	LastAccessed int64  `db:"lastaccessed"`
	Token        string `db:"token"`
	Username     string `db:"username"`
	Validity     int    `db:"validity"`
}

// tokenStorer is interface for token store
type tokenStorer interface {
	Create(username string, validity int) (string, error)
}
