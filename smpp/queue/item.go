package queue

import (
	"encoding/json"
)

// Item holds a message that's parsed to/from rabbitmq queue
// Transmission is in json format
type Item struct {
	Msg      string
	Dst      string
	Src      string
	Enc      string
	Priority int
}

// FromJSON parses json and sets attributes in Item struct
func (q *Item) FromJSON(b []byte) error {
	return json.Unmarshal(b, q)
}

// ToJSON parses json and sets attributes in Item struct
func (q *Item) ToJSON() ([]byte, error) {
	return json.Marshal(q)
}
