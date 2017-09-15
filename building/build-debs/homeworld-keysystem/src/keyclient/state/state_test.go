package state

import (
	"crypto/rsa"
	"crypto/tls"
	"io/ioutil"
	"keyclient/config"
	"os"
	"testing"
	"util/fileutil"
	"util/testkeyutil"
	"util/testutil"
)

func TestClientState_ReloadKeygrantingCert_NoCerts(t *testing.T) {
	err := os.Remove("../testdir/test.pem")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = os.Remove("../testdir/test.key")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	state := &ClientState{Config: config.Config{
		CertPath: "../testdir/test.pem",
		KeyPath:  "../testdir/test.key",
	}}
	err = state.ReloadKeygrantingCert()
	testutil.CheckError(t, err, "no keygranting certificate found")
	if state.Keygrant != nil {
		t.Error("no keygrant should be present")
	}
}

func TestClientState_ReloadKeygrantingCert_NoCerts_Preserve(t *testing.T) {
	err := os.Remove("../testdir/test.pem")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = os.Remove("../testdir/test.key")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	state := &ClientState{Config: config.Config{
		CertPath: "../testdir/test.pem",
		KeyPath:  "../testdir/test.key",
	}}
	state.Keygrant = &tls.Certificate{OCSPStaple: []byte("not a valid cert")}
	err = state.ReloadKeygrantingCert()
	testutil.CheckError(t, err, "no keygranting certificate found")
	if string(state.Keygrant.OCSPStaple) != "not a valid cert" {
		t.Error("keygrant modified when it should have been preserved")
	}
}

func TestClientState_ReloadKeygrantingCert_NoCertYesKey(t *testing.T) {
	err := os.Remove("../testdir/test.pem")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("../testdir/test.key", []byte("hello world"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	state := &ClientState{Config: config.Config{
		CertPath: "../testdir/test.pem",
		KeyPath:  "../testdir/test.key",
	}}
	err = state.ReloadKeygrantingCert()
	testutil.CheckError(t, err, "no keygranting certificate found")
	if state.Keygrant != nil {
		t.Error("no keygrant should be present")
	}
	os.Remove("../testdir/test.key")
}

func TestClientState_ReloadKeygrantingCert_YesCertNoKey(t *testing.T) {
	err := os.Remove("../testdir/test.key")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("../testdir/test.pem", []byte("hello world"), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	state := &ClientState{Config: config.Config{
		CertPath: "../testdir/test.pem",
		KeyPath:  "../testdir/test.key",
	}}
	err = state.ReloadKeygrantingCert()
	testutil.CheckError(t, err, "no keygranting certificate found")
	if state.Keygrant != nil {
		t.Error("no keygrant should be present")
	}
	os.Remove("../testdir/test.pem")
}

func TestClientState_ReloadKeygrantingCert_InvalidData(t *testing.T) {
	err := ioutil.WriteFile("../testdir/test.key", []byte("hello world"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("../testdir/test.pem", []byte("hello world"), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	state := &ClientState{Config: config.Config{
		CertPath: "../testdir/test.pem",
		KeyPath:  "../testdir/test.key",
	}}
	err = state.ReloadKeygrantingCert()
	testutil.CheckError(t, err, "failed to reload keygranting certificate: tls: failed to find any PEM data")
	if state.Keygrant != nil {
		t.Error("no keygrant should be present")
	}
	os.Remove("../testdir/test.pem")
	os.Remove("../testdir/test.key")
}

func TestClientState_ReloadKeygrantingCert_InvalidData_Preserve(t *testing.T) {
	err := ioutil.WriteFile("../testdir/test.key", []byte("hello world"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("../testdir/test.pem", []byte("hello world"), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	state := &ClientState{Config: config.Config{
		CertPath: "../testdir/test.pem",
		KeyPath:  "../testdir/test.key",
	}}
	state.Keygrant = &tls.Certificate{OCSPStaple: []byte("not a valid cert")}
	err = state.ReloadKeygrantingCert()
	testutil.CheckError(t, err, "failed to reload keygranting certificate: tls: failed to find any PEM data")
	if string(state.Keygrant.OCSPStaple) != "not a valid cert" {
		t.Error("keygrant modified when it should have been preserved")
	}
	os.Remove("../testdir/test.pem")
	os.Remove("../testdir/test.key")
}

func TestClientState_ReloadKeygrantingCert_ValidData(t *testing.T) {
	key, _, cert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	err := ioutil.WriteFile("../testdir/test.key", key, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("../testdir/test.pem", cert, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	state := &ClientState{Config: config.Config{
		CertPath: "../testdir/test.pem",
		KeyPath:  "../testdir/test.key",
	}}
	err = state.ReloadKeygrantingCert()
	if err != nil {
		t.Fatal(err)
	}
	if state.Keygrant == nil {
		t.Error("expected keygrant")
	} else {
		if state.Keygrant.PrivateKey.(*rsa.PrivateKey).Validate() != nil {
			t.Error("Invalid private key")
		}
	}
	os.Remove("../testdir/test.pem")
	os.Remove("../testdir/test.key")
}

func TestClientState_ReloadKeygrantingCert_ValidData_NoPreserve(t *testing.T) {
	key, _, cert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	err := ioutil.WriteFile("../testdir/test.key", key, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("../testdir/test.pem", cert, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	state := &ClientState{Config: config.Config{
		CertPath: "../testdir/test.pem",
		KeyPath:  "../testdir/test.key",
	}}
	state.Keygrant = &tls.Certificate{OCSPStaple: []byte("not a valid cert")}
	err = state.ReloadKeygrantingCert()
	if err != nil {
		t.Fatal(err)
	}
	if state.Keygrant == nil {
		t.Error("expected keygrant")
	} else {
		if state.Keygrant.PrivateKey.(*rsa.PrivateKey).Validate() != nil {
			t.Error("Invalid private key")
		}
		if string(state.Keygrant.OCSPStaple) != "" {
			t.Error("keygrant preserved when it should have been modified")
		}
	}
	os.Remove("../testdir/test.pem")
	os.Remove("../testdir/test.key")
}

func TestClientState_ReplaceKeygrantingCert(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	state := &ClientState{Config: config.Config{
		CertPath: "testdir/testa.pem",
		KeyPath:  "testdir/testa.key",
	}}
	keypem, _, certpem := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	err = ioutil.WriteFile("testdir/testa.key", keypem, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = state.ReplaceKeygrantingCert(certpem)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/testa.key")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/testa.pem")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientState_ReplaceKeygrantingCert_NoFolder(t *testing.T) {
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
	state := &ClientState{Config: config.Config{
		CertPath: "testdir/brokendir/testa.pem",
		KeyPath:  "testdir/testa.key",
	}}
	keypem, _, certpem := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	err = ioutil.WriteFile("testdir/testa.key", keypem, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = state.ReplaceKeygrantingCert(certpem)
	testutil.CheckError(t, err, "testdir/brokendir/testa.pem: permission denied")
	err = os.Remove("testdir/testa.key")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientState_ReplaceKeygrantingCert_Invalid(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	state := &ClientState{Config: config.Config{
		CertPath: "testdir/testa.pem",
		KeyPath:  "testdir/testa.key",
	}}
	keypem, _, _ := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	err = ioutil.WriteFile("testdir/testa.key", keypem, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = state.ReplaceKeygrantingCert([]byte("invalid cert"))
	testutil.CheckError(t, err, "failed to reload keygranting certificate: tls: failed to find any PEM data in certificate input")
	err = os.Remove("testdir/testa.key")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/testa.pem")
	if err != nil {
		t.Fatal(err)
	}
}
