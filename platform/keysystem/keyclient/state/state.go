package state

import (
	"crypto/tls"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/util/fileutil"
)

type ClientState struct {
	Keyserver *server.Keyserver
	Keygrant  *tls.Certificate

	// entries are present here if an API was inaccessible because we weren't authorized for that particular API
	// this lets us avoid constantly trying to request something from an API that isn't intended for us
	RetryAt map[string]time.Time
}

func NewClientState(keyserver *server.Keyserver) *ClientState {
	return &ClientState{
		Keyserver: keyserver,
		RetryAt:   make(map[string]time.Time),
	}
}

// how frequently to recheck whether our access is denied, in case it changes
const RetryAfter = time.Hour * 12

func (s *ClientState) RetryFailed(api string) {
	s.RetryAt[api] = time.Now().Add(RetryAfter)
}

func (s *ClientState) CanRetry(api string) bool {
	retryAt, found := s.RetryAt[api]
	return !found || time.Now().After(retryAt)
}

func (s *ClientState) ReloadKeygrantingCert() error {
	if fileutil.Exists(paths.GrantingKeyPath) && fileutil.Exists(paths.GrantingCertPath) {
		cert, err := tls.LoadX509KeyPair(paths.GrantingCertPath, paths.GrantingKeyPath)
		if err != nil {
			return errors.Wrap(err, "failed to reload keygranting certificate")
		} else {
			s.Keygrant = &cert
			return nil
		}
	} else {
		return errors.New("no keygranting certificate found")
	}
}

func (s *ClientState) ReplaceKeygrantingCert(data []byte) error {
	err := fileutil.EnsureIsFolder(path.Dir(paths.GrantingCertPath)) // TODO: unit test
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(paths.GrantingCertPath, data, os.FileMode(0644))
	if err != nil {
		return err
	}
	err = s.ReloadKeygrantingCert()
	if err != nil {
		return errors.Wrap(err, "expected properly loaded keygrant certificate")
	}
	return nil
}
