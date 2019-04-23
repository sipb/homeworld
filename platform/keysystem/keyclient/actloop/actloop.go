package actloop

import (
	"log"
	"sync"
	"time"
)

type ActLoop struct {
	actions    []Action
	stoplock   sync.Mutex
	shouldStop bool
	logger     *log.Logger
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

func (m *ActLoop) Step() (stabilized bool, blocked bool) {
	var blockedBy []error
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
			blockedBy = append(blockedBy, blockerr)
			continue
		}
		err = action.Perform(m.logger)
		if err != nil {
			m.logger.Printf("actloop step error: %s (in %s)\n", err.Error(), action.Info())
		} else {
			m.logger.Printf("action performed: %s\n", action.Info())
			return false, false
		}
	}
	if len(blockedBy) == 0 {
		return true, false
	} else {
		m.logger.Printf("ACTLOOP BLOCKED (%d)\n", len(blockedBy))
		for _, blockerr := range blockedBy {
			m.logger.Printf("actloop blocked by: %s\n", blockerr.Error())
		}
		// we're calling this 'stable' because the problems won't resolve themselves; if no actions were executed, we're stuck.
		return true, true
	}
}

func (m *ActLoop) Cancel() {
	m.stoplock.Lock()
	defer m.stoplock.Unlock()
	m.shouldStop = true
}

func (m *ActLoop) IsCancelled() bool {
	m.stoplock.Lock()
	defer m.stoplock.Unlock()
	return m.shouldStop
}

func (m *ActLoop) Run(cycletime time.Duration, pausetime time.Duration, onReady func(*log.Logger)) {
	wasStabilized := false
	for !m.IsCancelled() {
		// TODO: report current status somewhere -- health checker endpoint?
		stabilized, blocked := m.Step()
		if stabilized {
			if !wasStabilized {
				if blocked {
					m.logger.Printf("ACTLOOP BLOCKED BUT STABLE\n")
				} else {
					m.logger.Printf("ACTLOOP STABLE\n")
					if onReady != nil {
						onReady(m.logger)
					}
				}
			}
			time.Sleep(pausetime) // usually five minutes
		} else {
			time.Sleep(cycletime) // usually two seconds
		}
		wasStabilized = stabilized
	}
}
