package certutil

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
	"testing"
	"time"

	"github.com/sipb/homeworld/platform/util/testkeyutil"
	"github.com/sipb/homeworld/platform/util/testutil"
)

func TestCheckSSHCertExpiration(t *testing.T) {
	signkey, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(signkey)
	if err != nil {
		t.Fatal(err)
	}
	testtime := time.Now().Add(time.Hour*13 + time.Second*20)
	pk, err := ssh.NewPublicKey(signkey.Public())
	if err != nil {
		t.Fatal(err)
	}
	cert := &ssh.Certificate{
		Key:         pk,
		ValidBefore: uint64(testtime.Unix()),
	}
	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		t.Fatal(err)
	}
	at, err := CheckSSHCertExpiration(ssh.MarshalAuthorizedKey(cert))
	if err != nil {
		t.Fatal(err)
	}
	expect := time.Unix(testtime.Unix(), 0)
	if !at.Equal(expect) {
		t.Error("wrong time:", at, expect)
	}
}

func TestCheckSSHCertExpiration_Missing(t *testing.T) {
	_, err := CheckSSHCertExpiration([]byte{})
	testutil.CheckError(t, err, "ssh: no key found")
}

func TestCheckSSHCertExpiration_Invalid(t *testing.T) {
	_, err := CheckSSHCertExpiration([]byte("invalid"))
	testutil.CheckError(t, err, "ssh: no key found")
}

func TestCheckSSHCertExpiration_PubkeyInstead(t *testing.T) {
	signkey, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}
	pubkey, err := ssh.NewPublicKey(&signkey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CheckSSHCertExpiration(ssh.MarshalAuthorizedKey(pubkey))
	testutil.CheckError(t, err, "found public key instead of certificate when checking expiration")
}

func TestCheckTLSCertExpiration(t *testing.T) {
	issueat := time.Now()
	endat := issueat.Add(time.Minute * 72)
	_, cert := testkeyutil.GenerateTLSKeypairForTests_WithTime(t, "test", nil, nil, nil, nil, issueat, endat.Sub(issueat))
	foundtime, err := CheckTLSCertExpiration(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
	if err != nil {
		t.Fatal(err)
	}
	expect := time.Unix(endat.Unix(), 0)
	if !foundtime.Equal(expect) {
		t.Error("wrong time:", foundtime, "instead of", expect)
	}
}

func TestCheckTLSCertExpiration_Missing(t *testing.T) {
	_, err := CheckTLSCertExpiration([]byte{})
	testutil.CheckError(t, err, "Missing expected PEM header")
}

func TestCheckTLSCertExpiration_Invalid(t *testing.T) {
	_, err := CheckTLSCertExpiration([]byte("invalid"))
	testutil.CheckError(t, err, "Missing expected PEM header")
}
