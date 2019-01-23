package setup

import (
	"bytes"
	"io/ioutil"
	"keysystem/keyclient/actions/bootstrap"
	"keysystem/keyclient/actions/download"
	"keysystem/keyclient/actions/keygen"
	"keysystem/keyclient/actions/keyreq"
	"log"
	"os"
	"testing"
	"time"
	"util/testkeyutil"
	"util/testutil"
)

func TestLoad_Minimal(t *testing.T) {
	_, _, certpem := testkeyutil.GenerateTLSRootPEMsForTests(t, "keyservertls", nil, nil)
	err := ioutil.WriteFile("testdir/keyservertls.pem", certpem, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}

	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	acts, err := Load("testdir/nearlyempty.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	if len(acts) != 0 {
		t.Error("should be no actions")
	}

	err = os.Remove("testdir/keyservertls.pem")
	if err != nil {
		t.Error(err)
	}
	if logbuf.String() != "[load] keygranting cert not yet available: no keygranting certificate found\n" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoad_KeygrantingCert(t *testing.T) {
	_, _, certpem := testkeyutil.GenerateTLSRootPEMsForTests(t, "keyservertls", nil, nil)
	err := ioutil.WriteFile("testdir/keyservertls.pem", certpem, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	grantkey, _, grantcert := testkeyutil.GenerateTLSRootPEMsForTests(t, "grant", nil, nil)
	err = ioutil.WriteFile("testdir/granting.key", grantkey, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/granting.pem", grantcert, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}

	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	acts, err := Load("testdir/nearlyempty.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	if len(acts) != 0 {
		t.Error("should be no actions")
	}

	err = os.Remove("testdir/keyservertls.pem")
	if err != nil {
		t.Error(err)
	}
	err = os.Remove("testdir/granting.pem")
	if err != nil {
		t.Error(err)
	}
	err = os.Remove("testdir/granting.key")
	if err != nil {
		t.Error(err)
	}
	if logbuf.String() != "" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoad_Full_Actions(t *testing.T) {
	_, _, certpem := testkeyutil.GenerateTLSRootPEMsForTests(t, "keyservertls", nil, nil)
	err := ioutil.WriteFile("testdir/keyservertls.pem", certpem, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}

	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	acts, err := Load("testdir/test.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	if len(acts) != 5 {
		t.Fatal("wrong number of actions")
	}

	// pre-bootstrap keygen action
	kgen := acts[0].(keygen.TLSKeygenAction)
	if kgen.Bits != 4096 {
		t.Error("wrong number of bits")
	}
	if kgen.Keypath != "testdir/granting.key" {
		t.Error("wrong keygen target")
	}

	// bootstrap action
	bstrap := acts[1].(*bootstrap.BootstrapAction)
	state := bstrap.State
	if bstrap.TokenAPI != "renew-keygrant" {
		t.Error("wrong token API")
	}
	if bstrap.TokenFilePath != "testdir/bootstrap.token" {
		t.Error("wrong token path")
	}

	// keygen action
	kgen = acts[2].(keygen.TLSKeygenAction)
	if kgen.Bits != 4096 {
		t.Error("wrong number of bits")
	}
	if kgen.Keypath != "testdir/granting2.key" {
		t.Error("wrong keygen target")
	}

	// keyreq action
	kreq := acts[3].(*keyreq.RequestOrRenewAction)
	if kreq.State != state {
		t.Error("mismatch of state")
	}
	if kreq.API != "renew-keygrant2" {
		t.Error("wrong grant API")
	}
	if kreq.CertFile != "testdir/granting2.pem" {
		t.Error("wrong cert file")
	}
	if kreq.KeyFile != "testdir/granting2.key" {
		t.Error("wrong cert file")
	}
	if kreq.InAdvance != time.Hour*336 {
		t.Error("wrong in-advance interval")
	}
	if kreq.Name != "keygranting" {
		t.Error("wrong name")
	}

	// download action
	down := acts[4].(*download.DownloadAction)
	if down.Path != "testdir/etcd-client.pem" {
		t.Error("wrong path")
	}
	if down.Mode != 0644 {
		t.Error("wrong file mode")
	}
	if down.Refresh != time.Hour*24 {
		t.Error("wrong refresh interval")
	}
	fetcher := down.Fetcher.(*download.AuthorityFetcher)
	if fetcher.Keyserver != state.Keyserver {
		t.Error("wrong keyserver")
	}
	if fetcher.AuthorityName != "etcd-client" {
		t.Error("wrong client")
	}

	// state
	if state.Keygrant != nil {
		t.Error("should be no keygrant currently")
	}
	if state.Config.KeyPath != "testdir/granting.key" {
		t.Error("keygranting path is wrong")
	}

	err = os.Remove("testdir/keyservertls.pem")
	if err != nil {
		t.Error(err)
	}
	if logbuf.String() != "[load] keygranting cert not yet available: no keygranting certificate found\n" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoad_MissingConfig(t *testing.T) {
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	_, err := Load("testdir/nonexistent.yaml", logger)
	testutil.CheckError(t, err, "testdir/nonexistent.yaml: no such file or directory")
	if logbuf.String() != "" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoad_NoAuthority(t *testing.T) {
	err := os.Remove("testdir/keyservertls.pem")
	if err != nil && !os.IsNotExist(err) {
		t.Error(err)
	}

	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	_, err = Load("testdir/nearlyempty.yaml", logger)
	testutil.CheckError(t, err, "while loading authority: open testdir/keyservertls.pem: no such file or directory")

	if logbuf.String() != "" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoad_BrokenAuthority(t *testing.T) {
	err := ioutil.WriteFile("testdir/keyservertls.pem", []byte("not a key"), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}

	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	_, err = Load("testdir/nearlyempty.yaml", logger)
	testutil.CheckError(t, err, "Missing expected PEM header")

	err = os.Remove("testdir/keyservertls.pem")
	if err != nil {
		t.Error(err)
	}
	if logbuf.String() != "" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoad_BrokenBootstrap(t *testing.T) {
	_, _, certpem := testkeyutil.GenerateTLSRootPEMsForTests(t, "keyservertls", nil, nil)
	err := ioutil.WriteFile("testdir/keyservertls.pem", certpem, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}

	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	_, err = Load("testdir/broken_bootstrap.yaml", logger)
	testutil.CheckError(t, err, "no bootstrap api provided")

	err = os.Remove("testdir/keyservertls.pem")
	if err != nil {
		t.Error(err)
	}
	if logbuf.String() != "" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoad_BrokenKeygen(t *testing.T) {
	_, _, certpem := testkeyutil.GenerateTLSRootPEMsForTests(t, "keyservertls", nil, nil)
	err := ioutil.WriteFile("testdir/keyservertls.pem", certpem, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}

	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	_, err = Load("testdir/broken_keygen.yaml", logger)
	testutil.CheckError(t, err, "unrecognized key type: pin-tumbler")

	err = os.Remove("testdir/keyservertls.pem")
	if err != nil {
		t.Error(err)
	}
	if logbuf.String() != "" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoad_BrokenKeyreq(t *testing.T) {
	_, _, certpem := testkeyutil.GenerateTLSRootPEMsForTests(t, "keyservertls", nil, nil)
	err := ioutil.WriteFile("testdir/keyservertls.pem", certpem, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}

	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	_, err = Load("testdir/broken_keyreq.yaml", logger)
	testutil.CheckError(t, err, "invalid in-advance interval for key renewal")

	err = os.Remove("testdir/keyservertls.pem")
	if err != nil {
		t.Error(err)
	}
	if logbuf.String() != "" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoad_BrokenDownload(t *testing.T) {
	_, _, certpem := testkeyutil.GenerateTLSRootPEMsForTests(t, "keyservertls", nil, nil)
	err := ioutil.WriteFile("testdir/keyservertls.pem", certpem, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}

	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	_, err = Load("testdir/broken_download.yaml", logger)
	testutil.CheckError(t, err, "unrecognized download type: ftp")

	err = os.Remove("testdir/keyservertls.pem")
	if err != nil {
		t.Error(err)
	}
	if logbuf.String() != "" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLaunch(t *testing.T) {
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	stop := Launch(nil, logger)
	defer stop()
	time.Sleep(time.Millisecond * 50)
	stop()
	if logbuf.String() != "[load] ACTLOOP STABILIZED\n" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoadAndLaunch_Missing(t *testing.T) {
	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	_, err := LoadAndLaunch("testdir/nonexistent.yaml", logger)
	testutil.CheckError(t, err, "testdir/nonexistent.yaml: no such file or directory")
	if logbuf.String() != "" {
		t.Error("wrong log output:", logbuf.String())
	}
}

func TestLoadAndLaunch(t *testing.T) {
	_, _, certpem := testkeyutil.GenerateTLSRootPEMsForTests(t, "keyservertls", nil, nil)
	err := ioutil.WriteFile("testdir/keyservertls.pem", certpem, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}

	logbuf := bytes.NewBuffer(nil)
	logger := log.New(logbuf, "[load] ", 0)
	stop, err := LoadAndLaunch("testdir/nearlyempty.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	time.Sleep(time.Millisecond * 50)
	stop()
	if logbuf.String() != "[load] keygranting cert not yet available: no keygranting certificate found\n[load] ACTLOOP STABILIZED\n" {
		t.Error("wrong log output:", logbuf.String())
	}

	err = os.Remove("testdir/keyservertls.pem")
	if err != nil {
		t.Error(err)
	}
}
