package keyclient

import (
	"keycommon"
	"sync"
	"time"
	"crypto/tls"
	"log"
	"keyclient/util"
	"fmt"
	"strings"
)

const RSA_BITS = 4096

type Mainloop struct {
	ks          *keycommon.Keyserver
	config      Config
	actions     []Action
	stoponce    sync.Once
	should_stop bool
	keygrant    *tls.Certificate
	logger      log.Logger
}

type Action interface {
	Perform() error
}

type errNothingToDo struct {}

func (errNothingToDo) Error() string {
	return "Nothing to do!"
}

type errBlockedAction struct {
	Message string
}

func (e errBlockedAction) Error() string {
	return fmt.Sprintf("action blocked: %s", e.Message)
}

var ErrNothingToDo = errNothingToDo{}

func IsBlockedAction(err error) bool {
	_, match := err.(errBlockedAction)
	return match
}

// TODO: key rotation, not just getting new certs

func Load(configpath string) (*Mainloop, error) {
	ks, config, err := LoadConfig(configpath)
	if err != nil {
		return nil, err
	}
	m := &Mainloop{config: config, ks: ks}
	actions := []Action{}
	if config.TokenPath != "" {
		act, err := PrepareBootstrapAction(m, config.TokenPath, config.TokenAPI)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	for _, key := range config.Keys {
		inadvance, err := time.ParseDuration(key.InAdvance)
		if err != nil {
			return nil, err
		}
		act, err := PrepareKeygenAction(m, key)
		if err != nil {
			return nil, err
		}
		if act != nil {
			actions = append(actions, act)
		}
		act, err = PrepareRequestOrRenewKeys(m, key, inadvance)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	for _, download := range config.Downloads {
		act, err := PrepareDownloadAction(m, download)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act)
	}
	m.actions = actions
	m.reloadKeygrantingCert()
	return m, nil
}

func (m *Mainloop) Stop() {
	m.stoponce.Do(func() {
		m.should_stop = true
	})
}

func (m *Mainloop) reloadKeygrantingCert() {
	if util.Exists(m.config.KeyPath) && util.Exists(m.config.CertPath) {
		cert, err := tls.LoadX509KeyPair(m.config.CertPath, m.config.KeyPath)
		if err != nil {
			m.logger.Printf("Failed to reload keygranting certificate: %s", err)
		} else {
			m.keygrant = &cert
		}
	} else {
		m.logger.Printf("No keygranting certificate found.")
	}
}

func (m *Mainloop) tryPerformStep() error {
	blocked_actions := []string{}
	for _, action := range m.actions {
		err := action.Perform()
		if err == nil {
			return nil
		} else if err == ErrNothingToDo {
			// no need to warn about this
		} else if IsBlockedAction(err) {
			blocked_actions = append(blocked_actions, err.(errBlockedAction).Message)
		} else {
			m.logger.Printf("step action failed: %s", err)
		}
	}
	if len(blocked_actions) == 0 {
		return ErrNothingToDo
	} else if len(blocked_actions) == 1 {
		return errBlockedAction{blocked_actions[0]}
	} else {
		return errBlockedAction{"all of [" + strings.Join(blocked_actions, ", ") + "]"}
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
				m.logger.Printf("step: %s", err)
			}
			time.Sleep(time.Second * 10)
		}
	}
}
