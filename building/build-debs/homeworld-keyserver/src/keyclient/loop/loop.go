package loop

import (
	"crypto/tls"
	"fmt"
	"util/fileutil"
	"log"
	"strings"
	"sync"
	"time"
	"keycommon/server"
	"keyclient/config"
)

type Mainloop struct {
	Keyserver   *server.Keyserver
	Config      config.Config
	Actions     []Action
	stoponce    sync.Once
	should_stop bool
	Keygrant    *tls.Certificate
	Logger      log.Logger
}

type Action interface {
	Perform() error
}

type errNothingToDo struct{}

func (errNothingToDo) Error() string {
	return "Nothing to do!"
}

type ErrBlockedAction struct {
	Message string
}

func (e ErrBlockedAction) Error() string {
	return fmt.Sprintf("action blocked: %s", e.Message)
}

var ErrNothingToDo = errNothingToDo{}

func IsBlockedAction(err error) bool {
	_, match := err.(ErrBlockedAction)
	return match
}

func (m *Mainloop) Stop() {
	m.stoponce.Do(func() {
		m.should_stop = true
	})
}

func (m *Mainloop) ReloadKeygrantingCert() {
	if fileutil.Exists(m.Config.KeyPath) && fileutil.Exists(m.Config.CertPath) {
		cert, err := tls.LoadX509KeyPair(m.Config.CertPath, m.Config.KeyPath)
		if err != nil {
			m.Logger.Printf("Failed to reload keygranting certificate: %s", err)
		} else {
			m.Keygrant = &cert
		}
	} else {
		m.Logger.Printf("No keygranting certificate found.")
	}
}

func (m *Mainloop) tryPerformStep() error {
	blocked_actions := []string{}
	for _, action := range m.Actions {
		err := action.Perform()
		if err == nil {
			return nil
		} else if err == ErrNothingToDo {
			// no need to warn about this
		} else if IsBlockedAction(err) {
			blocked_actions = append(blocked_actions, err.(ErrBlockedAction).Message)
		} else {
			m.Logger.Printf("step action failed: %s", err)
		}
	}
	if len(blocked_actions) == 0 {
		return ErrNothingToDo
	} else if len(blocked_actions) == 1 {
		return ErrBlockedAction{blocked_actions[0]}
	} else {
		return ErrBlockedAction{"all of [" + strings.Join(blocked_actions, ", ") + "]"}
	}
}

func (m *Mainloop) Run() {
	for !m.should_stop {
		// TODO: report current status somewhere -- health checker endpoint?
		err := m.tryPerformStep()
		if err == ErrNothingToDo {
			time.Sleep(time.Minute * 5)
		} else {
			if err != nil {
				m.Logger.Printf("step: %s", err)
			}
			time.Sleep(time.Second * 10)
		}
	}
}
