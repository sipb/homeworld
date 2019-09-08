package download

import (
	"errors"
	"fmt"

	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
)

type DownloadFetcher interface {
	PrereqsSatisfied() error
	Fetch() ([]byte, error)
	Info() string
}

type AuthorityFetcher struct {
	Keyserver     *server.Keyserver
	AuthorityName string
}

type StaticFetcher struct {
	Keyserver  *server.Keyserver
	StaticName string
}

type APIFetcher struct {
	State *state.ClientState
	API   string
}

func (af *AuthorityFetcher) PrereqsSatisfied() error {
	return nil // so, yes
}

func (af *AuthorityFetcher) Info() string {
	return fmt.Sprintf("pubkey for authority %s", af.AuthorityName)
}

func (af *AuthorityFetcher) Fetch() ([]byte, error) {
	return af.Keyserver.GetPubkey(af.AuthorityName)
}

func (sf *StaticFetcher) PrereqsSatisfied() error {
	return nil // so, yes
}

func (sf *StaticFetcher) Info() string {
	return fmt.Sprintf("static file %s", sf.StaticName)
}

func (sf *StaticFetcher) Fetch() ([]byte, error) {
	return sf.Keyserver.GetStatic(sf.StaticName)
}

func (df *APIFetcher) PrereqsSatisfied() error {
	if df.State.Keygrant != nil {
		return nil
	} else {
		return errors.New("no keygranting certificate ready")
	}
}

func (df *APIFetcher) Info() string {
	return fmt.Sprintf("result from api %s", df.API)
}

func (df *APIFetcher) Fetch() ([]byte, error) {
	rt, err := df.State.Keyserver.AuthenticateWithCert(*df.State.Keygrant)
	if err != nil {
		return nil, err // no actual way for this part to fail
	}
	resp, err := reqtarget.SendRequest(rt, df.API, "")
	if err != nil {
		return nil, err
	}
	return []byte(resp), nil
}
