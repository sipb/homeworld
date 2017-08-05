package util

import (
	"testing"
	"time"
	"sync"
	"math/rand"
)

func TestSingleThreadedFlag(t *testing.T) {
	for i := 0; i < 1000; i++ {
		flag := NewOnceFlag()
		if !flag.Set() {
			t.Errorf("Flag did not get set the first time!")
		}
		if flag.Set() {
			t.Errorf("Flag got set twice!")
		}
	}
}

func TestIndependentFlags(t *testing.T) {
	for i := 0; i < 1000; i++ {
		flag1 := NewOnceFlag()
		flag2 := NewOnceFlag()
		if !flag1.Set() {
			t.Errorf("Flag did not get set the first time!")
		}
		if !flag2.Set() {
			t.Errorf("Flag did not get set the first time!")
		}
		if flag1.Set() {
			t.Errorf("Flag got set twice!")
		}
		if flag2.Set() {
			t.Errorf("Flag got set twice!")
		}
	}
}

func TestParallelFlags(t *testing.T) {
	boxes := make([]OnceFlag, 1000)
	count := 0
	countsync := sync.Mutex{}
	done := make(chan int)
	for i := 0; i < 1000; i++ {
		go func() {
			time.Sleep(time.Millisecond * 10)
			for j := 0; j < 3000; j++ {
				idx := rand.Intn(len(boxes))
				if boxes[idx].Set() {
					countsync.Lock()
					count++
					countsync.Unlock()
				}
			}
			done <- 0
		}()
	}
	for i := 0; i < 1000; i++ {
		_ = <-done
	}
	if count != len(boxes) {
		t.Errorf("Got a count besides %s: %s", len(boxes), count)
	}
}
