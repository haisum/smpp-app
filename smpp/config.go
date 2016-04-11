package smpp

import (
	"fmt"
)

// Config represents all settings defined in settings file
type Config struct {
	ConnGroups []ConnGroup
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

// GetKeys returns all prefixes defined by all the connections
func (c *Config) GetKeys(group string) []string {
	var keys []string
	g, err := c.GetGroup(group)
	if err != nil {
		return keys
	}
	for _, con := range g.Conns {
		keys = append(keys, con.Pfxs...)
	}
	return keys
}

// GetConn returns a connection with given id
func (c *Config) GetConn(group, id string) (Conn, error) {
	var con Conn
	g, err := c.GetGroup(group)
	if err != nil {
		return con, err
	}
	for _, con = range g.Conns {
		if con.ID == id {
			return con, nil
		}
	}

	return con, fmt.Errorf("Couldn't find key for connection %s.", id)
}

func (c *Config) GetGroup(group string) (ConnGroup, error) {
	var cg ConnGroup
	for _, g := range c.ConnGroups {
		if g.Name == group {
			return g, nil
		}
	}
	return cg, fmt.Errorf("Couldn't find group with name %s.", group)
}
