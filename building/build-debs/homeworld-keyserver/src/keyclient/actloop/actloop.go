package actloop

import (
	"sync"
	"log"
	"time"
)

type ActLoop struct {
	actions     []Action
	stoponce    sync.Once
	should_stop bool
	logger      *log.Logger
}

type Action interface {
	Pending() (bool, error)
	CheckBlocker() error   // error means "this can't happen yet"
	Perform(logger *log.Logger) error
}

func NewActLoop(actions []Action, logger *log.Logger) ActLoop {
	return ActLoop{actions: actions, logger: logger}
}

func (m *ActLoop) Step() (stabilized bool) {
	blocked_by := []error{}
	for _, action := range m.actions {
		pending, err := action.Pending()
		if err != nil {
			m.logger.Print("actloop check error: %s\n", err.Error())
		}
		if !pending {
			continue
		}
		blockerr := action.CheckBlocker()
		if blockerr != nil {
			blocked_by = append(blocked_by, blockerr)
			continue
		}
		err = action.Perform(m.logger)
		if err != nil {
			m.logger.Printf("actloop step error: %s\n", err.Error())
		} else {
			return false
		}
	}
	if len(blocked_by) == 0 {
		return true
	} else {
		m.logger.Printf("ACTLOOP BLOCKED (%d)\n", len(blocked_by))
		for _, blockerr := range blocked_by {
			m.logger.Printf("actloop blocked by: %s\n", blockerr.Error())
		}
		return false
	}
}

func (m *ActLoop) Cancel() {
	m.stoponce.Do(func() {
		m.should_stop = true
	})
}

func (m *ActLoop) Run() {
	was_stabilized := false
	for !m.should_stop {
		// TODO: report current status somewhere -- health checker endpoint?
		stabilized := m.Step()
		if stabilized {
			if !was_stabilized {
				m.logger.Printf("ACTLOOP STABILIZED\n")
			}
			time.Sleep(time.Minute * 5)
		} else {
			time.Sleep(time.Second * 10)
		}
		was_stabilized = stabilized
	}
}
