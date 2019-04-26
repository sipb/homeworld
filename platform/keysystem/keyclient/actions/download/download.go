package download

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
	"github.com/sipb/homeworld/platform/util/fileutil"
)

type FetchFunc func(nac *actloop.NewActionContext) ([]byte, error)

type config struct {
	Path    string
	Refresh time.Duration
	Mode    uint64
}

func DownloadAuthority(name string, path string, refreshPeriod time.Duration, nac *actloop.NewActionContext) {
	act := &config{
		Path:    path,
		Refresh: refreshPeriod,
		Mode:    0644,
	}
	fetch, fetchInfo := fetchAuthority(name)
	act.Download(nac, fetch, fetchInfo)
}

func DownloadStatic(name string, path string, refreshPeriod time.Duration, nac *actloop.NewActionContext) {
	act := &config{
		Path:    path,
		Refresh: refreshPeriod,
		Mode:    0644,
	}
	fetch, fetchInfo := fetchStatic(name)
	act.Download(nac, fetch, fetchInfo)
}

func DownloadFromAPI(api string, path string, refreshPeriod time.Duration, mode uint64, nac *actloop.NewActionContext) {
	act := &config{
		Path:    path,
		Refresh: refreshPeriod,
		Mode:    mode,
	}
	fetch, fetchInfo := fetchAPI(api)
	act.Download(nac, fetch, fetchInfo)
}

func (da *config) Download(nac *actloop.NewActionContext, fetcher FetchFunc, fetchInfo string) {
	info := fmt.Sprintf("download to file %s (mode %o) every %v: %s", da.Path, da.Mode, da.Refresh, fetchInfo)
	if da.needsRefresh(nac, info) {
		err := da.refresh(nac, fetcher, info)
		if err != nil {
			nac.Errored(info, err)
		}
	}
}

func (da *config) needsRefresh(nac *actloop.NewActionContext, info string) bool {
	if statinfo, err := os.Stat(da.Path); err != nil {
		if os.IsNotExist(err) {
			return true
		}
		// somehow it's broken
		nac.Errored(info, err)
		return false
	} else {
		staleness := time.Now().Sub(statinfo.ModTime())
		return staleness > da.Refresh
	}
}

func (da *config) refresh(nac *actloop.NewActionContext, fetcher FetchFunc, info string) error {
	data, err := fetcher(nac)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		// no error, but nothing to fetch.
		return nil
	}
	err = fileutil.EnsureIsFolder(path.Dir(da.Path))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(da.Path, data, os.FileMode(da.Mode))
	if err != nil {
		return err
	}
	nac.NotifyPerformed(info)
	return nil
}
