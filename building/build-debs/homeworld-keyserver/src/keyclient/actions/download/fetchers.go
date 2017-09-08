package download

import (
	"keycommon/reqtarget"
	"keycommon/server"
	"keyclient/state"
	"errors"
)

type DownloadFetcher interface {
	PrereqsSatisfied() error
	Fetch() ([]byte, error)
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

func (af *AuthorityFetcher) Fetch() ([]byte, error) {
	return af.Keyserver.GetPubkey(af.AuthorityName)
}

func (sf *StaticFetcher) PrereqsSatisfied() error {
	return nil // so, yes
}

func (sf *StaticFetcher) Fetch() ([]byte, error) {
	return sf.Keyserver.GetStatic(sf.StaticName)
}

func (df *APIFetcher) PrereqsSatisfied() error {
	if df.State.Keygrant != nil {
		return nil
	} else {
		return errors.New("No keygranting certificate ready.")
	}
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
