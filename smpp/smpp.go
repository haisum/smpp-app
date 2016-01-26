package smpp

import (
	"encoding/json"
)

// Holds a message that's parsed to/from rabbitmq queue
// Transmission is in json format
type QueueItem struct {
	Msg      string
	Dst      string
	Src      string
	Priority int
}

// Parses json and sets attributes in QueueItem struct
func (q *QueueItem) FromJSON(b []byte) error {
	return json.Unmarshal(b, q)
}

// Parses json and sets attributes in QueueItem struct
func (q *QueueItem) ToJSON() ([]byte, error) {
	return json.Marshal(q)
}
