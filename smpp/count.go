package smpp

import (
	"sync"
)

//Count mutex holds number of messages sent in t seconds
type CountMutex struct {
	sync.RWMutex
	Count int32
}
