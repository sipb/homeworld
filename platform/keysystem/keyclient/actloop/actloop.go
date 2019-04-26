package actloop

import (
	"log"
	"sync"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
)

type ActLoop struct {
	actions    NewAction
	stoplock   sync.Mutex
	shouldStop bool
	logger     *log.Logger
}

type NewAction func(nac *NewActionContext)

type NewActionContext struct {
	State     *state.ClientState
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

func (m *ActLoop) Run(state *state.ClientState, cycletime time.Duration, pausetime time.Duration, onReady func(*log.Logger)) {
	wasStabilized := false
	for !m.IsCancelled() {
		nac := NewActionContext{
			Logger: m.logger,
			State:  state,
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
