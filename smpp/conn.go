package smpp

// Conn represents configuration specific to a single smpp connection
type Conn struct {
	ID     string
	URL    string
	User   string
	Size   int32
	Time   int
	Passwd string
	Pfxs   []string
	Fields PduFields
}
