package fileutil

import (
	"io/ioutil"
	"os"
	"testing"
	"util/testutil"
)

func TestExists_Not(t *testing.T) {
	os.Mkdir("testdir", os.FileMode(0755)) // ignore errors
	if Exists("testdir/path_that_should_not_exist.txt") {
		t.Error("should not have existed")
	}
}

func TestExists(t *testing.T) {
	os.Mkdir("testdir", os.FileMode(0755)) // ignore errors
	err := ioutil.WriteFile("testdir/path_that_should_exist.txt", []byte("test"), os.FileMode(0755))
	if err != nil {
		t.Fatal(err)
	}
	if !Exists("testdir/path_that_should_exist.txt") {
		t.Error("should have existed")
	}
}

func TestEnsureIsFolder_NotDir(t *testing.T) {
	os.Mkdir("testdir", os.FileMode(0755))         // ignore errors
	os.Remove("testdir/should_not_be_a_directory") // ignore errors
	err := ioutil.WriteFile("testdir/should_not_be_a_directory", []byte("test"), os.FileMode(0755))
	if err != nil {
		t.Fatal(err)
	}
	err = EnsureIsFolder("testdir/should_not_be_a_directory")
	testutil.CheckError(t, err, "not a directory")
}

func TestEnsureIsFolder_AlreadyExists(t *testing.T) {
	os.Mkdir("testdir", os.FileMode(0755))     // ignore errors
	os.Remove("testdir/should_be_a_directory") // ignore errors
	err := os.Mkdir("testdir/should_be_a_directory", os.FileMode(0755))
	if err != nil {
		t.Fatal(err)
	}
	err = EnsureIsFolder("testdir/should_be_a_directory")
	if err != nil {
		t.Error(err)
	}
	info, err := os.Stat("testdir/should_be_a_directory")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("Not a directory.")
	}
}

func TestEnsureIsFolder_CreateOne(t *testing.T) {
	os.Mkdir("testdir", os.FileMode(0755))     // ignore errors
	os.Remove("testdir/should_be_a_directory") // ignore errors
	info, err := os.Stat("testdir/should_be_a_directory")
	if err == nil {
		t.Fatal("expected error")
	} else if !os.IsNotExist(err) {
		t.Fatal("expected nonexistence error")
	}
	err = EnsureIsFolder("testdir/should_be_a_directory")
	if err != nil {
		t.Error(err)
	}
	info, err = os.Stat("testdir/should_be_a_directory")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("Not a directory.")
	}
}

func TestEnsureIsFolder_CreateTwo(t *testing.T) {
	os.Mkdir("testdir", os.FileMode(0755)) // ignore errors
	os.Remove("testdir/parent_dir/subdir") // ignore errors
	os.Remove("testdir/parent_dir")        // ignore errors
	info, err := os.Stat("testdir/parent_dir")
	if err == nil {
		t.Fatal("expected error")
	} else if !os.IsNotExist(err) {
		t.Fatal("expected nonexistence error")
	}
	err = EnsureIsFolder("testdir/parent_dir/subdir")
	if err != nil {
		t.Error(err)
	}
	info, err = os.Stat("testdir/parent_dir/subdir")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("Not a directory.")
	}
}

func TestEnsureIsFolder_CannotStat(t *testing.T) {
	os.Mkdir("testdir", os.FileMode(0755)) // ignore errors
	os.Remove("testdir/brokendir")         // ignore errors
	err := ioutil.WriteFile("testdir/brokendir", []byte("test"), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	err = EnsureIsFolder("testdir/brokendir/invalidpath")
	testutil.CheckError(t, err, "not a directory")
}

func TestEnsureIsFolder_CannotMkdir(t *testing.T) {
	os.Mkdir("testdir", os.FileMode(0755)) // ignore errors
	os.Remove("testdir/limiteddir")        // ignore errors
	err := os.Mkdir("testdir/limiteddir", os.FileMode(0555))
	if err != nil {
		t.Fatal(err)
	}
	err = EnsureIsFolder("testdir/limiteddir/disallowed")
	testutil.CheckError(t, err, "permission denied")
}

func TestCreateFile(t *testing.T) {
	err := EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/testfile.txt")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = CreateFile("testdir/testfile.txt", []byte("hello world\n"), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	data, err := ioutil.ReadFile("testdir/testfile.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world\n" {
		t.Error("wrong file data")
	}
}

func TestCreateFile_Contended(t *testing.T) {
	err := EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/testfile.txt", []byte("original\n"), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	err = CreateFile("testdir/testfile.txt", []byte("hello world\n"), os.FileMode(0644))
	testutil.CheckError(t, err, "testdir/testfile.txt: file exists")
	data, err := ioutil.ReadFile("testdir/testfile.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original\n" {
		t.Error("wrong file data")
	}
}
