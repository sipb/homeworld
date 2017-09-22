package actloop

import (
	"log"
	"sync"
	"time"
)

type ActLoop struct {
	actions     []Action
	stoplock    sync.Mutex
	should_stop bool
	logger      *log.Logger
}

type Action interface {
	Pending() (bool, error)
	CheckBlocker() error // error means "this can't happen yet"
	Perform(logger *log.Logger) error
	Info() string
}

func NewActLoop(actions []Action, logger *log.Logger) ActLoop {
	return ActLoop{actions: actions, logger: logger}
}

func (m *ActLoop) Step() (stabilized bool) {
	blocked_by := []error{}
	for _, action := range m.actions {
		pending, err := action.Pending()
		if err != nil {
			m.logger.Printf("actloop check error: %s (in %s)\n", err.Error(), action.Info())
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
			m.logger.Printf("actloop step error: %s (in %s)\n", err.Error(), action.Info())
		} else {
			m.logger.Printf("action performed: %s\n", action.Info())
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
		// we're calling this 'stable' because the problems won't resolve themselves; if no actions were executed, we're stuck.
		return true
	}
}

func (m *ActLoop) Cancel() {
	m.stoplock.Lock()
	defer m.stoplock.Unlock()
	m.should_stop = true
}

func (m *ActLoop) IsCancelled() bool {
	m.stoplock.Lock()
	defer m.stoplock.Unlock()
	return m.should_stop
}

func (m *ActLoop) Run(cycletime time.Duration, pausetime time.Duration) {
	was_stabilized := false
	for !m.IsCancelled() {
		// TODO: report current status somewhere -- health checker endpoint?
		stabilized := m.Step()
		if stabilized {
			if !was_stabilized {
				m.logger.Printf("ACTLOOP STABILIZED\n")
			}
			time.Sleep(pausetime) // usually five minutes
		} else {
			time.Sleep(cycletime) // usually two seconds
		}
		was_stabilized = stabilized
	}
}
