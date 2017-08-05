package util

import "sync"

type BooleanFlag struct {
	mut   sync.Mutex
	value bool
}

func (f *BooleanFlag) Set() bool {
	f.mut.Lock()
	defer f.mut.Unlock()
	if f.value {
		return false
	} else {
		f.value = true
		return true
	}
}

func (f *BooleanFlag) Unset() bool {
	f.mut.Lock()
	defer f.mut.Unlock()
	if f.value {
		f.value = false
		return true
	} else {
		return false
	}
}

func NewBooleanFlag() *BooleanFlag {
	return &BooleanFlag{value: false}
}
