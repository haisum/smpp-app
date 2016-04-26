package priority

import (
	"bitbucket.org/codefreak/hsmpp/smpp/queue"
)

const (
	Low queue.Priority = iota
	Normal
	Medium
	High
)
