package priority

import (
	"bitbucket.com/codefreak/hsmpp/smpp/queue"
)

const (
	Low queue.Priority = iota
	Normal
	Medium
	High
)
