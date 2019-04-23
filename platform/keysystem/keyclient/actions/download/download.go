package download

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/sipb/homeworld/platform/util/fileutil"
)

type DownloadAction struct {
	Fetcher DownloadFetcher
	Path    string
	Refresh time.Duration
	Mode    uint64
}

func (da *DownloadAction) Info() string {
	return fmt.Sprintf("download to file %s (mode %o) every %v: %s", da.Path, da.Mode, da.Refresh, da.Fetcher.Info())
}

func (da *DownloadAction) Pending() (bool, error) {
	if !da.Fetcher.CanRetry() {
		return false, nil
	}
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
