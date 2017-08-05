package util

import (
	"sync"
)

type OnceFlag struct {
	mut   sync.Mutex
	value bool
}

// returns true if we were the first, false otherwise
func (f *OnceFlag) Set() bool {
	f.mut.Lock()
	defer f.mut.Unlock()
	if f.value {
		return false
	} else {
		f.value = true
		return true
	}
}

func NewOnceFlag() *OnceFlag {
	return &OnceFlag{value: false}
}
