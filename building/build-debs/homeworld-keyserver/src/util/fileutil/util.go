package fileutil

import (
	"errors"
	"fmt"
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

func CreateFile(filename string, contents []byte, permissions os.FileMode) error {
	file_out, err := os.OpenFile(filename, os.O_WRONLY|os.O_EXCL|os.O_CREATE, permissions)
	if err != nil {
		return err
	}
	_, err = file_out.Write(contents)
	if err != nil {
		file_out.Close() // ignore failure: nothing more we can do...
	} else {
		err = file_out.Close()
	}
	if err != nil {
		err2 := os.Remove(filename) // do our best to remove it...
		if err2 != nil && !os.IsNotExist(err2) {
			err = fmt.Errorf("multiple errors while creating file: %s | %s", err.Error(), err2.Error())
		}
	}
	return err
}
