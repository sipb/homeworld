package keygen

import (
	"io/ioutil"
	"keyclient/config"
	"os"
	"testing"
	"util/fileutil"
	"util/testutil"
	"util/wraputil"
)

func TestPrepareKeygenAction_TLSPubkey_Disallowed(t *testing.T) {
	key := config.ConfigKey{
		Type: "tls-pubkey",
	}
	_, err := PrepareKeygenAction(key)
	testutil.CheckError(t, err, "unrecognized key type: tls-pubkey")
}

func TestPrepareKeygenAction_SSHPubkey(t *testing.T) {
	key := config.ConfigKey{
		Type: "ssh-pubkey",
	}
	action, err := PrepareKeygenAction(key)
	if err != nil {
		t.Fatal(err)
	}
	if action != nil {
		t.Error("action should be nil")
	}
}

func TestPrepareKeygenAction_TLSKey(t *testing.T) {
	key := config.ConfigKey{
		Type: "tls",
		Key:  "testdir/crypto.key",
	}
	action, err := PrepareKeygenAction(key)
	if err != nil {
		t.Fatal(err)
	}
	if action.(TLSKeygenAction).Keypath != "testdir/crypto.key" {
		t.Error("wrong key path")
	}
	if action.(TLSKeygenAction).Bits != 4096 {
		t.Error("wrong number of Bits")
	}
}

func TestPrepareKeygenAction_SSHKey_Unimplemented(t *testing.T) {
	key := config.ConfigKey{
		Type: "ssh",
	}
	_, err := PrepareKeygenAction(key)
	testutil.CheckError(t, err, "unimplemented operation")
}

func TestPrepareKeygenAction_Invalid(t *testing.T) {
	key := config.ConfigKey{
		Type: "pin-tumbler-key",
	}
	_, err := PrepareKeygenAction(key)
	testutil.CheckError(t, err, "unrecognized key type: pin-tumbler-key")
}

func TestTLSKeygenAction_Pending_NoKey(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	ispending, err := TLSKeygenAction{Keypath: "testdir/nonexistent.key"}.Pending()
	if err != nil {
		t.Fatal(err)
	}
	if !ispending {
		t.Error("should be pending")
	}
}

func TestTLSKeygenAction_Pending_YesKey(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/existent.key", []byte("test"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	ispending, err := TLSKeygenAction{Keypath: "testdir/existent.key"}.Pending()
	if err != nil {
		t.Fatal(err)
	}
	if ispending {
		t.Error("should not be pending")
	}

	os.Remove("testdir/existent.key")
}

func TestTLSKeygenAction_CheckBlocker(t *testing.T) {
	value := TLSKeygenAction{}.CheckBlocker()
	if value != nil {
		t.Error("keygen should never be blocked")
	}
}

func TestTLSKeygenAction_Perform(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/output.key")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	// weird number to make sure that the number is actually taken into account
	err = TLSKeygenAction{"testdir/output.key", 519}.Perform(nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadFile("testdir/output.key")
	if err != nil {
		t.Fatal(err)
	}
	key, err := wraputil.LoadRSAKeyFromPEM(data)
	if err != nil {
		t.Fatal(err)
	}

	err = key.Validate()
	if err != nil {
		t.Fatal(err)
	}

	if key.N.BitLen() != 519 {
		t.Fatal("Wrong number of Bits in key")
	}

	os.Remove("testdir/output.key")
}

func TestTLSKeygenAction_Perform_NoCreateDirectories(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/testinvalid")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = os.Mkdir("testdir/testinvalid", os.FileMode(0))
	if err != nil {
		t.Fatal(err)
	}

	// weird number to make sure that the number is actually taken into account
	err = TLSKeygenAction{"testdir/testinvalid/subdir/output.key", 519}.Perform(nil)
	testutil.CheckError(t, err, "failed to prepare directory")
	testutil.CheckError(t, err, "permission denied")

	os.Remove("testdir/testinvalid")
}

func TestTLSKeygenAction_Perform_CannotGen(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/output.key")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	// weird number to make sure that the number is actually taken into account
	err = TLSKeygenAction{"testdir/output.key", 0}.Perform(nil)
	testutil.CheckError(t, err, "too few primes of given length")
}

func TestTLSKeygenAction_Perform_NoCreateFile(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/testinvalid")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = os.Mkdir("testdir/testinvalid", os.FileMode(0))
	if err != nil {
		t.Fatal(err)
	}

	// weird number to make sure that the number is actually taken into account
	err = TLSKeygenAction{"testdir/testinvalid/output.key", 519}.Perform(nil)
	testutil.CheckError(t, err, "failed to create file")
	testutil.CheckError(t, err, "permission denied")

	os.Remove("testdir/testinvalid")
}
