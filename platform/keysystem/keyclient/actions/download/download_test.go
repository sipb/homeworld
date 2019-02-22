package download

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/config"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/state"
	"github.com/sipb/homeworld/platform/util/fileutil"
	"github.com/sipb/homeworld/platform/util/testutil"
)

func TestPrepareDownloadAction_Static(t *testing.T) {
	ml := &state.ClientState{Keyserver: &server.Keyserver{}}
	action, err := PrepareDownloadAction(ml, config.ConfigDownload{
		Type:    "static",
		Refresh: "24h",
		Name:    "teststatic",
		Mode:    "600",
		Path:    "testdir/test.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if action.(*DownloadAction).Path != "testdir/test.txt" {
		t.Error("wrong path")
	}
	if action.(*DownloadAction).Mode != 0600 {
		t.Error("wrong mode")
	}
	if action.(*DownloadAction).Refresh != time.Hour*24 {
		t.Error("wrong refresh interval")
	}
	fetcher := action.(*DownloadAction).Fetcher.(*StaticFetcher)
	if fetcher.Keyserver != ml.Keyserver {
		t.Error("wrong keyserver")
	}
	if fetcher.StaticName != "teststatic" {
		t.Error("wrong static name")
	}
}

func TestPrepareDownloadAction_Pubkey(t *testing.T) {
	ml := &state.ClientState{Keyserver: &server.Keyserver{}}
	action, err := PrepareDownloadAction(ml, config.ConfigDownload{
		Type:    "authority",
		Refresh: "24h",
		Name:    "testkey",
		Mode:    "600",
		Path:    "testdir/test.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if action.(*DownloadAction).Path != "testdir/test.txt" {
		t.Error("wrong path")
	}
	if action.(*DownloadAction).Mode != 0600 {
		t.Error("wrong mode")
	}
	if action.(*DownloadAction).Refresh != time.Hour*24 {
		t.Error("wrong refresh interval")
	}
	fetcher := action.(*DownloadAction).Fetcher.(*AuthorityFetcher)
	if fetcher.Keyserver != ml.Keyserver {
		t.Error("wrong keyserver")
	}
	if fetcher.AuthorityName != "testkey" {
		t.Error("wrong static name")
	}
}

func TestPrepareDownloadAction_API(t *testing.T) {
	ml := &state.ClientState{Keyserver: &server.Keyserver{}}
	action, err := PrepareDownloadAction(ml, config.ConfigDownload{
		Type:    "api",
		Refresh: "24h",
		Name:    "testapi",
		Mode:    "600",
		Path:    "testdir/test.txt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if action.(*DownloadAction).Path != "testdir/test.txt" {
		t.Error("wrong path")
	}
	if action.(*DownloadAction).Mode != 0600 {
		t.Error("wrong mode")
	}
	if action.(*DownloadAction).Refresh != time.Hour*24 {
		t.Error("wrong refresh interval")
	}
	fetcher := action.(*DownloadAction).Fetcher.(*APIFetcher)
	if fetcher.State != ml {
		t.Error("wrong keyserver")
	}
	if fetcher.API != "testapi" {
		t.Error("wrong API name")
	}
}

func TestPrepareDownloadAction_Unrecognized(t *testing.T) {
	ml := &state.ClientState{Keyserver: &server.Keyserver{}}
	_, err := PrepareDownloadAction(ml, config.ConfigDownload{
		Type:    "ftp",
		Refresh: "24h",
		Name:    "testapi",
		Mode:    "600",
		Path:    "testdir/test.txt",
	})
	testutil.CheckError(t, err, "unrecognized download type: ftp")
}

func TestPrepareDownloadAction_InvalidDuration_UnsupportedUnit(t *testing.T) {
	ml := &state.ClientState{Keyserver: &server.Keyserver{}}
	_, err := PrepareDownloadAction(ml, config.ConfigDownload{
		Type:    "static",
		Refresh: "24d",
		Name:    "testapi",
		Mode:    "600",
		Path:    "testdir/test.txt",
	})
	testutil.CheckError(t, err, "unknown unit d")
}

func TestPrepareDownloadAction_InvalidDuration_Invalid(t *testing.T) {
	ml := &state.ClientState{Keyserver: &server.Keyserver{}}
	_, err := PrepareDownloadAction(ml, config.ConfigDownload{
		Type:    "static",
		Refresh: "one hour",
		Name:    "testapi",
		Mode:    "600",
		Path:    "testdir/test.txt",
	})
	testutil.CheckError(t, err, "invalid duration one hour")
}

func TestPrepareDownloadAction_NotOctal(t *testing.T) {
	ml := &state.ClientState{Keyserver: &server.Keyserver{}}
	_, err := PrepareDownloadAction(ml, config.ConfigDownload{
		Type:    "static",
		Refresh: "24h",
		Name:    "testapi",
		Mode:    "800",
		Path:    "testdir/test.txt",
	})
	testutil.CheckError(t, err, "strconv.ParseUint: parsing \"800\": invalid syntax")
}

func TestPrepareDownloadAction_WorldWritable(t *testing.T) {
	ml := &state.ClientState{Keyserver: &server.Keyserver{}}
	_, err := PrepareDownloadAction(ml, config.ConfigDownload{
		Type:    "static",
		Refresh: "24h",
		Name:    "testapi",
		Mode:    "666",
		Path:    "testdir/test.txt",
	})
	testutil.CheckError(t, err, "will not grant world-writable access")
}

func TestDownloadAction_Pending(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	da := &DownloadAction{Refresh: time.Hour, Path: "testdir/nonexistent.txt"}
	b, err := da.Pending()
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("nonexistent file should be pending")
	}
}

func TestDownloadAction_Pending_Broken(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Mkdir("testdir/nonexistent", os.FileMode(0))
	if err != nil && !os.IsExist(err) {
		t.Fatal("could not create broken dir")
	}
	da := &DownloadAction{Refresh: time.Hour, Path: "testdir/nonexistent/nonexistent.txt"}
	b, err := da.Pending()
	if !b {
		t.Error("nonexistent file should be pending")
	}
	testutil.CheckError(t, err, "permission denied")
}

func TestDownloadAction_Pending_Recent(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/existent.txt", []byte("test"), os.FileMode(644))
	if err != nil {
		t.Fatal(err)
	}
	da := &DownloadAction{Refresh: time.Hour, Path: "testdir/existent.txt"}
	b, err := da.Pending()
	if err != nil {
		t.Fatal(err)
	}
	if b {
		t.Error("recent existent file should not be pending")
	}
	os.Remove("testdir/existent.txt")
}

func TestDownloadAction_Pending_Old(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/existent.txt", []byte("test"), os.FileMode(644))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond * 10)
	da := &DownloadAction{Refresh: time.Millisecond * 5, Path: "testdir/existent.txt"}
	b, err := da.Pending()
	if err != nil {
		t.Fatal(err)
	}
	if !b {
		t.Error("older existent file should be pending")
	}
	os.Remove("testdir/existent.txt")
}

type FakeFetcher struct {
	prereqs   error
	data      []byte
	datafail  error
	fetchinfo string
}

func (f *FakeFetcher) PrereqsSatisfied() error {
	return f.prereqs
}

func (f *FakeFetcher) Fetch() ([]byte, error) {
	return f.data, f.datafail
}

func (f *FakeFetcher) Info() string {
	return f.fetchinfo
}

func TestDownloadAction_CheckBlocker_Satisfied(t *testing.T) {
	da := &DownloadAction{Refresh: time.Hour, Path: "testdir/existent.txt",
		Fetcher: &FakeFetcher{nil, nil, errors.New("invalid"), ""}}
	if da.CheckBlocker() != nil {
		t.Error("should not be blocked")
	}
}

func TestDownloadAction_CheckBlocker_Unsatisfied(t *testing.T) {
	da := &DownloadAction{Refresh: time.Hour, Path: "testdir/existent.txt",
		Fetcher: &FakeFetcher{errors.New("purposeful error"), nil, errors.New("invalid"), ""}}
	err := da.CheckBlocker()
	testutil.CheckError(t, err, "purposeful error")
}

func TestDownloadAction_Perform(t *testing.T) {
	err := os.Remove("testdir/result.txt")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	da := &DownloadAction{Refresh: time.Hour, Path: "testdir/result.txt", Mode: 0765,
		Fetcher: &FakeFetcher{errors.New("purposeful error"), []byte("validated results\n"), nil, ""}}
	err = da.Perform(nil)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat("testdir/result.txt")
	if err != nil {
		t.Fatal(err)
	}
	if (info.Mode() & 0777) != 0745 { // 0745 because umask
		t.Error("invalid mode:", info.Mode())
	}
	data, err := ioutil.ReadFile("testdir/result.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "validated results\n" {
		t.Error("invalid results")
	}
	os.Remove("testdir/result.txt")
}

func TestDownloadAction_Perform_InvalidFilename(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/brokendir")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = os.Mkdir("testdir/brokendir", os.FileMode(0))
	if err != nil {
		t.Fatal(err)
	}
	da := &DownloadAction{Refresh: time.Hour, Path: "testdir/brokendir/result.txt", Mode: 0644,
		Fetcher: &FakeFetcher{errors.New("purposeful error"), []byte("validated results\n"), nil, ""}}
	err = da.Perform(nil)
	testutil.CheckError(t, err, "testdir/brokendir/result.txt: permission denied")
}

func TestDownloadAction_Perform_FetchFailed(t *testing.T) {
	da := &DownloadAction{Refresh: time.Hour, Path: "testdir/result.txt", Mode: 0644,
		Fetcher: &FakeFetcher{errors.New("purposeful error"), nil, errors.New("purposeful failure"), ""}}
	err := da.Perform(nil)
	testutil.CheckError(t, err, "purposeful failure")
}

func TestDownloadAction_Info(t *testing.T) {
	da := &DownloadAction{Refresh: time.Hour, Path: "testdir/result.txt", Mode: 0644, Fetcher: &FakeFetcher{
		errors.New("should not be used"), nil, errors.New("should not be used"), "test-fetchinfo-test",
	}}
	if da.Info() != "download to file testdir/result.txt (mode 644) every 1h0m0s: test-fetchinfo-test" {
		t.Error("wrong info for action: %s", da.Info())
	}
}
