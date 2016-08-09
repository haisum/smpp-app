package queue

import (
	"encoding/json"
)

// Item holds a message that's parsed to/from rabbitmq queue
// Transmission is in json format
type Item struct {
	MsgID string
	Total int
	// Msg parameter is required so that, masked msgs are recorded in db and
	// real messages are sent via worker
	Msg string
}

// FromJSON parses json and sets attributes in Item struct
func (q *Item) FromJSON(b []byte) error {
	return json.Unmarshal(b, q)
}

// ToJSON parses json and sets attributes in Item struct
func (q *Item) ToJSON() ([]byte, error) {
	return json.Marshal(q)
}
