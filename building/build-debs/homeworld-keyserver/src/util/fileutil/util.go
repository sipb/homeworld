package fileutil

import (
	"errors"
	"os"
)

func Exists(path string) bool {
	f, err := os.Open(path)
	if err == nil {
		f.Close()
		return true
	} else {
		return false
	}
}

func EnsureIsFolder(dirname string) error {
	fileinfo, err := os.Stat(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(dirname, 0755)
		}
		if err != nil {
			return err
		}
		fileinfo, err = os.Stat(dirname)
		if err != nil {
			return err // hard to test in practice, but still necessary
		}
	}
	if !fileinfo.IsDir() {
		return errors.New("not a directory")
	}
	return nil
}
