package wraputil

import (
	"testing"
	"golang.org/x/crypto/ssh"
	"util/testutil"
)

const (
	TEST_PUBKEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCY+MUkm8b9t2EkbOd6wjkNIFhqE6IUDKi6jj/ZWS4E73vcc1bbCXtk7QtPPRuU8fLcJtdU8DzzgI93eIbci8YQk0FNHWdf2qn4OEH+Ss0NehJC1kjKkdcxgks7HOM4BEUZjmsYkXw4zj69u2PXKWew+cGUnK9ZWysowL7463P6aVjOEbJgakzJ8dzY28d0VWetXoWhqr7SWVPpHRS4/+34Bgt4tLZmIvK9I0S0MXLws0Z2SsdWrLtryyLnJP54eSWIJSC58o8P2GD51V58/Z+6KaZaflePGjuAclyFqUDBIjXgwS0EKvoiJGkSChZKcgEeVKCFCQ8rPFLJU/PTzvhn user@hyades"
)

func TestParseSSHTextPubkey(t *testing.T) {
	key, err := ParseSSHTextPubkey([]byte(TEST_PUBKEY))
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := ssh.FingerprintLegacyMD5(key)
	if fingerprint != "88:c7:32:f3:0f:cb:91:63:cb:dd:18:e5:e0:be:bb:b5" {
		t.Error("Wrong fingerprint.")
	}
}

func TestParseSSHTextPubkey_Invalid(t *testing.T) {
	_, err := ParseSSHTextPubkey([]byte("invalid"))
	testutil.CheckError(t, err, "no key found")
}

func TestParseSSHTextPubkey_Empty(t *testing.T) {
	_, err := ParseSSHTextPubkey([]byte(""))
	testutil.CheckError(t, err, "no key found")
}

func TestParseSSHTextPubkey_Prefix(t *testing.T) {
	_, err := ParseSSHTextPubkey([]byte("invalid\n" + TEST_PUBKEY))
	testutil.CheckError(t, err, "extraneous leading data")
}

func TestParseSSHTextPubkey_Suffix(t *testing.T) {
	_, err := ParseSSHTextPubkey([]byte(TEST_PUBKEY + "\ninvalid"))
	testutil.CheckError(t, err, "extraneous trailing data")
}
