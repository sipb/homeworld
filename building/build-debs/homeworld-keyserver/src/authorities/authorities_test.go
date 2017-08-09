package authorities

import (
	"testing"
	"bytes"
	"strings"
)

func TestLoadAuthority(t *testing.T) {
	tlsa, err := LoadAuthority("TLS", []byte(TLS_TEST_PRIVKEY), []byte(TLS_TEST_CERT))
	if err != nil {
		t.Fatal(err)
	}
	tlst := tlsa.(*TLSAuthority)
	if !bytes.Equal(tlst.GetPublicKey(), []byte(TLS_TEST_CERT)) {
		t.Error("Mismatch!")
	}

	ssha, err := LoadAuthority("SSH", []byte(SSH_TEST_PRIVKEY), []byte(SSH_TEST_PUBKEY))
	if err != nil {
		t.Fatal(err)
	}
	ssht := ssha.(*SSHAuthority)
	if !bytes.Equal(ssht.GetPublicKey(), []byte(SSH_TEST_PUBKEY)) {
		t.Error("Mismatch!")
	}

	_, err = LoadAuthority("neither", []byte(SSH_TEST_PRIVKEY), []byte(SSH_TEST_PUBKEY))
	if err == nil {
		t.Error("Expected failure to load invalid kind of authority")
	} else if !strings.Contains(err.Error(), "Unrecognized kind") {
		t.Error("Expected unrecognized kind error.")
	}
}
