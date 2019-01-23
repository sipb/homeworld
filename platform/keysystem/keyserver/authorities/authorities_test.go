package authorities

import (
	"bytes"
	"github.com/sipb/homeworld/platform/util/testkeyutil"
	"strings"
	"testing"
)

func TestLoadAuthority(t *testing.T) {
	key, _, cert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	tlsa, err := LoadAuthority("TLS", key, cert)
	if err != nil {
		t.Fatal(err)
	}
	tlst := tlsa.(*TLSAuthority)
	if !bytes.Equal(tlst.GetPublicKey(), cert) {
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
