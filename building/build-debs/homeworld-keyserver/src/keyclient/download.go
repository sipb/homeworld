package keyclient

import (
	"keycommon"
	"time"
	"os"
	"io/ioutil"
)

type DownloadFetcher interface {
	PrereqsSatisfied() error
	Fetch() ([]byte, error)
}

type AuthorityFetcher struct {
	Keyserver     *keycommon.Keyserver
	AuthorityName string
}

type StaticFetcher struct {
	Keyserver     *keycommon.Keyserver
	StaticName string
}

type APIFetcher struct {
	Mainloop     *Mainloop
	API           string
}

type DownloadAction struct {
	Fetcher DownloadFetcher
	Path    string
	Refresh time.Duration
	Mode    uint64
}

func (da *DownloadAction) Perform() error {
	if statinfo, err := os.Stat(da.Path); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// doesn't exist -- create it!
	} else {
		staleness := time.Now().Sub(statinfo.ModTime())
		if staleness <= da.Refresh {
			return ErrNothingToDo
		}
		// stale! we should refresh it, if possible.
	}
	err := da.Fetcher.PrereqsSatisfied()
	if err != nil {
		return err
	}
	data, err := da.Fetcher.Fetch()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(da.Path, data, os.FileMode(da.Mode))
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
	if df.Mainloop.keygrant != nil {
		return nil
	} else {
		return errBlockedAction{"No keygranting certificate ready."}
	}
}

func (df *APIFetcher) Fetch() ([]byte, error) {
	rt, err := df.Mainloop.ks.AuthenticateWithCert(*df.Mainloop.keygrant)
	if err != nil {
		return nil, err
	}
	resp, err := keycommon.SendRequest(rt, df.API, "")
	if err != nil {
		return nil, err
	}
	return []byte(resp), nil
}
