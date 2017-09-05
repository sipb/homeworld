package download

import (
	"io/ioutil"
	"os"
	"time"
	"keycommon/server"
	"keycommon/reqtarget"
	"strconv"
	"fmt"
	"keyclient/state"
	"keyclient/config"
	"keyclient/actloop"
	"log"
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
	Mainloop *state.ClientState
	API      string
}

type DownloadAction struct {
	Fetcher DownloadFetcher
	Path    string
	Refresh time.Duration
	Mode    uint64
}

func PrepareDownloadAction(m *state.ClientState, d config.ConfigDownload) (actloop.Action, error) {
	refresh_period, err := time.ParseDuration(d.Refresh)
	if err != nil {
		return nil, err
	}
	mode, err := strconv.ParseUint(d.Mode, 8, 9)
	if err != nil {
		return nil, err
	}
	if mode&0002 != 0 {
		return nil, fmt.Errorf("Disallowed mode: %o (will not grant world-writable access)", mode)
	}
	switch d.Type {
	case "authority":
		return &DownloadAction{Fetcher: &AuthorityFetcher{Keyserver: m.Keyserver, AuthorityName: d.Name}, Path: d.Path, Refresh: refresh_period, Mode: mode}, nil
	case "static":
		return &DownloadAction{Fetcher: &StaticFetcher{Keyserver: m.Keyserver, StaticName: d.Name}, Path: d.Path, Refresh: refresh_period, Mode: mode}, nil
	case "api":
		return &DownloadAction{Fetcher: &APIFetcher{Mainloop: m, API: d.Name}, Path: d.Path, Refresh: refresh_period, Mode: mode}, nil
	default:
		return nil, fmt.Errorf("Unrecognized download type: %s", d.Type)
	}
}

func (da *DownloadAction) Pending() (bool, error) {
	if statinfo, err := os.Stat(da.Path); err != nil {
		if os.IsNotExist(err) {
			// doesn't exist -- create it!
			return true, nil
		} else {
			// broken -- try creating it (and probably fail!)
			return true, err
		}
	} else {
		staleness := time.Now().Sub(statinfo.ModTime())
		if staleness <= da.Refresh {
			// still valid
			return false, nil
		} else {
			// stale! we should refresh it, if possible.
			return true, nil
		}
	}
}

func (da *DownloadAction) CheckBlocker() error {
	return da.Fetcher.PrereqsSatisfied()
}

func (da *DownloadAction) Perform(logger *log.Logger) error {
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
	if df.Mainloop.Keygrant != nil {
		return nil
	} else {
		return errors.New("No keygranting certificate ready.")
	}
}

func (df *APIFetcher) Fetch() ([]byte, error) {
	rt, err := df.Mainloop.Keyserver.AuthenticateWithCert(*df.Mainloop.Keygrant)
	if err != nil {
		return nil, err
	}
	resp, err := reqtarget.SendRequest(rt, df.API, "")
	if err != nil {
		return nil, err
	}
	return []byte(resp), nil
}
