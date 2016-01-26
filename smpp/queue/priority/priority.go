package priority

import (
	"github.com/haisum/smpp/queue"
)

const (
	Low queue.Priority = iota
	Normal
	Medium
	High
)
