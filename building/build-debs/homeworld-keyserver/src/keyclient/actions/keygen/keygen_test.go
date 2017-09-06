package keygen

import (
	"testing"
	"keyclient/config"
	"util/testutil"
)

func TestPrepareKeygenAction_TLSPubkey(t *testing.T) {
	key := config.ConfigKey{
		Type: "tls-pubkey",
	}
	action, err := PrepareKeygenAction(key)
	if err != nil {
		t.Fatal(err)
	}
	if action != nil {
		t.Error("action should be nil")
	}
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
		Key: "testdir/crypto.key",
	}
	action, err := PrepareKeygenAction(key)
	if err != nil {
		t.Fatal(err)
	}
	if action.(TLSKeygenAction).keypath != "testdir/crypto.key" {
		t.Error("wrong key path")
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
