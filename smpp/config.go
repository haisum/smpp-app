package smpp

// Config represents all settings defined in settings file
type Config struct {
	AmqpURL    string
	ConnGroups []ConnGroup
	HTTPSPort  int
}

// ConnGroup is a group of connections to be used by a single tenant
type ConnGroup struct {
	Conns      []Conn
	DefaultPfx string
	Name       string
}

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

// PduFields are fields that may be sent to smpp server
// when sending an sms. These are usually optional but some smpp providers
// require them.
type PduFields struct {
	ServiceType          string
	SourceAddrTON        uint8
	SourceAddrNPI        uint8
	DestAddrTON          uint8
	DestAddrNPI          uint8
	ProtocolID           uint8
	PriorityFlag         uint8
	ScheduleDeliveryTime string
	ReplaceIfPresentFlag uint8
	SMDefaultMsgID       uint8
}
