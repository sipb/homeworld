package actloop

import (
	"log"
	"sync"
	"time"
)

type ActLoop struct {
	actions    NewAction
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

type NewAction func(nac *NewActionContext)

type NewActionContext struct {
	Logger    *log.Logger
	BlockedBy []error
	Performed bool
}

func (nac *NewActionContext) Errored(info string, err error) {
	nac.Logger.Printf("action stop error: %s (in %s)\n", err.Error(), info)
}

func (nac *NewActionContext) Blocked(err error) {
	nac.BlockedBy = append(nac.BlockedBy, err)
}

func (nac *NewActionContext) NotifyPerformed(info string) {
	nac.Logger.Printf("action performed: %s\n", info)
	nac.Performed = true
}

func ActionToNew(action Action) NewAction {
	return func(nac *NewActionContext) {
		pending, err := action.Pending()
		if err != nil {
			nac.Errored(action.Info(), err)
		} else if pending {
			blockerr := action.CheckBlocker()
			if blockerr != nil {
				nac.Blocked(blockerr)
			} else {
				err = action.Perform(nac.Logger)
				if err != nil {
					nac.Errored(action.Info(), err)
				} else {
					nac.NotifyPerformed(action.Info())
				}
			}
		}
	}
}

func MergeActions(newactions []NewAction) NewAction {
	return func(nac *NewActionContext) {
		for _, action := range newactions {
			action(nac)
			if nac.Performed {
				break
			}
		}
	}
}

func NewActLoop(actions NewAction, logger *log.Logger) ActLoop {
	return ActLoop{actions: actions, logger: logger}
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
		nac := NewActionContext{
			Logger: m.logger,
		}
		m.actions(&nac)

		// TODO: report current status somewhere -- health checker endpoint?

		if nac.Performed {
			time.Sleep(cycletime) // usually two seconds
		} else {
			if !wasStabilized {
				if len(nac.BlockedBy) > 0 {
					m.logger.Printf("ACTLOOP BLOCKED (%d)\n", len(nac.BlockedBy))
					for _, blockerr := range nac.BlockedBy {
						m.logger.Printf("actloop blocked by: %s\n", blockerr.Error())
					}
					m.logger.Printf("ACTLOOP BLOCKED BUT STABLE\n")
				} else {
					m.logger.Printf("ACTLOOP STABLE\n")
					if onReady != nil {
						onReady(m.logger)
					}
				}
			}
			time.Sleep(pausetime) // usually five minutes
		}
		wasStabilized = !nac.Performed
	}
}
