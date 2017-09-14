package download

import (
	"fmt"
	"io/ioutil"
	"keyclient/actloop"
	"keyclient/config"
	"keyclient/state"
	"log"
	"os"
	"strconv"
	"time"
	"util/fileutil"
	"path"
)

type DownloadAction struct {
	Fetcher DownloadFetcher
	Path    string
	Refresh time.Duration
	Mode    uint64
}

func PrepareDownloadAction(s *state.ClientState, d config.ConfigDownload) (actloop.Action, error) {
	refresh_period, err := time.ParseDuration(d.Refresh)
	if err != nil {
		return nil, err
	}
	mode, err := strconv.ParseUint(d.Mode, 8, 9)
	if err != nil {
		return nil, err
	}
	if mode&0002 != 0 {
		return nil, fmt.Errorf("disallowed mode: %o (will not grant world-writable access)", mode)
	}
	switch d.Type {
	case "authority":
		return &DownloadAction{Fetcher: &AuthorityFetcher{Keyserver: s.Keyserver, AuthorityName: d.Name}, Path: d.Path, Refresh: refresh_period, Mode: mode}, nil
	case "static":
		return &DownloadAction{Fetcher: &StaticFetcher{Keyserver: s.Keyserver, StaticName: d.Name}, Path: d.Path, Refresh: refresh_period, Mode: mode}, nil
	case "api":
		return &DownloadAction{Fetcher: &APIFetcher{State: s, API: d.Name}, Path: d.Path, Refresh: refresh_period, Mode: mode}, nil
	default:
		return nil, fmt.Errorf("unrecognized download type: %s", d.Type)
	}
}

func (da *DownloadAction) Info() string {
	return fmt.Sprintf("download to file %s (mode %o) every %v: %s", da.Path, da.Mode, da.Refresh, da.Fetcher.Info())
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
	err = fileutil.EnsureIsFolder(path.Dir(da.Path))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(da.Path, data, os.FileMode(da.Mode))
	if err != nil {
		return err
	}
	return nil
}
