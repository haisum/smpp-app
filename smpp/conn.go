package smpp

// Conn represents configuration specific to a single smpp connection
type Conn struct {
	Id     string
	Url    string
	User   string
	Size   int32
	Time   int
	Passwd string
	Pfxs   []string
	Fields PduFields
}
