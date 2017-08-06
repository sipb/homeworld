package authorities

import (
	"testing"
	"bytes"
	"time"
	"strings"
	"golang.org/x/crypto/ssh"
	"encoding/base64"
	"net"
	"io"
	"errors"
)

const (
	TEST_PUBKEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC5zWmwv8NiKfVkt9KHZ6vAWDnKonUGVbjE+REDhPZwU4obzMEjcx8Ha8mQHZSDzbW835DF9fvsJDARBnCIh/2AB1iUL0jdM2cRKKmqdzGrbHQmet4FgJoWCu7rQKgt4JTAxQVc0qGSBqBlKn2QCKtHUs9PJOEDHSz4l4LwiZ/E2xxD+5/M7EKdlcRXyBOZE6oAwIdV9JNjL0FiqN/QPWijZcFN0AWTql0NRxMq9EagOz9XhHLXdf3rPQzJ/IP/zK6ZB6DAQ53QDLfJ87PAeC/YmFWsB25lHGOV6X5bcyT0HDxfL1bYNCB0oNA417iDp5+yqYoFdDW1Ioj5P2QJbYm1 user@host"
	TEST_PRIVKEY = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpgIBAAKCAQEAuc1psL/DYin1ZLfSh2erwFg5yqJ1BlW4xPkRA4T2cFOKG8zB\nI3MfB2vJkB2Ug821vN+QxfX77CQwEQZwiIf9gAdYlC9I3TNnESipqncxq2x0Jnre\nBYCaFgru60CoLeCUwMUFXNKhkgagZSp9kAirR1LPTyThAx0s+JeC8ImfxNscQ/uf\nzOxCnZXEV8gTmROqAMCHVfSTYy9BYqjf0D1oo2XBTdAFk6pdDUcTKvRGoDs/V4Ry\n13X96z0MyfyD/8yumQegwEOd0Ay3yfOzwHgv2JhVrAduZRxjlel+W3Mk9Bw8Xy9W\n2DQgdKDQONe4g6efsqmKBXQ1tSKI+T9kCW2JtQIDAQABAoIBAQCtU/uhn/KT05KR\nZ45lNIgbgfI/nzfONg+M6NA/WT1QYg43itYtzMoIcTvyTjXqku9UB7cVhTiC/Os+\nJqS6KSqJ0dCHRGkTuU0Py8AjPtg+E4lzEDGoLmUP5RkmqwV47sW14tXy1qdVAwuD\n9JR31i559b1hFoU2E3SNX0IORESgLTLAujAnrCWvnal/O6d1Tzb9csVxj/8e2Ymw\nuONAnu6jOKyOxGt+VJAoMrZkutzhZixmdFfQyuYYV3ViLhKcgtpWOXjHvCJfJ/7o\nBCy74mHkti+i24Ssd5sgdiWOuoURjJgm3okQtTGSR4guPWkYAajLzeQYD9aSepzR\nHSIIRXWBAoGBAOxjHLpEHMIq9O0vshDcu2dq5pTEOBarVXr2sh5WBUUwZjqCuop5\n3KV/olSRENipnYYOIHiDMDSjcB05cFApmKmi7CIQOOnNSgZjpzdsKXQKEWLfEO2S\nYJ+lybcSYaKxONIAPU9PISxZgI6jYSc3nb/uEDStERm4cnB6MV2cXanhAoGBAMk3\n4ACoACgov/9wY0OHyXRfvEflo9AJvUHXgRvopX06lMSDM0j73TD4QM/OEdXEDdTn\nUgnK2altMfS6p0Y8z7vYpoJ0k/zz34FxNNY0pZI1pMgYEOOz2c+jFsNZkOvei33Rq7y6XsfObS3xLOx/zZ+lSIf5bFtAfPaPKvRkzGJVAoGBANPlaGwEAG+BSDqRZaI9\n63Oh3P4AAnM3tJFcMICHBYRnBUxvwT2+TS7BgccinqJJMR5o7Wx51K1q0GYyBd6l\n2uY9WESUnB/g2PlvPQauW15cZAdoA+miLCEP4QjNXl4TVObSNiMwwIDb3iR+iek4\nrpzMjxRZCxouP89ZiYTrVP6hAoGBALUQl3xfsMxyZtrX6irRXIFgyI814GOK8Af4\ngVB417nJZiczHIoXQiIXslKMT1Y5dmzXvuXa6GRiQyrCb1Vv0UpqmOMZPjXHyZ60\nHOSIOVlI9j+sED6mD2CdlBUzWoo1FvagHtbUKgfIBEzsEg26r3ByDcN1uYCflhNU\nH0YOEjCFAoGBAOUtlDMr6V455tpcTF03dQW4kZ07zY40TgghZwFjybYsM34KFC00\nO0Ot3Or50r4rYhZTw7snzTmP3ktGIdvZqFydwrwVhS8vXnx0/l/5Ka9gn95NAWFV\nKrwFtKrYCxQtNA+qlVODiLeMXgOQVj9HM7ad4favGxlyfZRV2FF4ROug\n-----END RSA PRIVATE KEY-----"

	TEST2_PUBKEY = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDXEV53SU+URbdq5l062XBptEwreAJd9ejRXXhBrE2rHr40zqhksDtEJFqXTsVdBnSAopQsuXzw2SrvqFa8vAiYKL10sU51Xfj1W0AfGL4fuls6uAHjrXf+AmDkAb0r+w4LmLj8OgpVaA1F/CjntNNUm7cGfUdMVYHip0fzXhg5OkOFYGSh9SE7B0R39cZnqYXdk99L5c47RT6zG6pvKXf9nwo6OPdb3CpNNrt3or2y7+bNJskbXdUcnOehimACKJVwG1XS8T37CEhHsa6HIGO3/O00XpvOYHug+J4DgvFZyTgWsCPgHL3Im5Wx0mxHFZNaYBDuGyXpvT+a4qRfGr5/ user2@host2"
	TEST3_PUBKEY = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEvnwaQ9ql2V6OfGh94Xl0K5BOlozsfZeyMY7GJw7QjtNqPkI+02K5tSjMOzEwAMBFDgOKjBBhEgU7apjjPTkds= user@hyades"
)

func getAuthority(t *testing.T) *SSHAuthority {
	authority, err := LoadSSHAuthority([]byte(TEST_PRIVKEY), []byte(TEST_PUBKEY))
	if err != nil {
		t.Fatalf("Could not create SSH authority: %s", err)
	}
	return authority.(*SSHAuthority)
}

func TestLoadSSHAuthority(t *testing.T) {
	_ = getAuthority(t)
}

func TestGetSSHPublicKey(t *testing.T) {
	if !bytes.Equal([]byte(TEST_PUBKEY), getAuthority(t).GetPublicKey()) {
		t.Error("Pubkey bytes mismatch")
	}
}

type FakeConnMetadata struct {
	user string
}

func (f FakeConnMetadata) User() string {
	return f.user
}

func (f FakeConnMetadata) SessionID() []byte {
	panic("Mock")
}

func (f FakeConnMetadata) ClientVersion() []byte {
	panic("Mock")
}

func (f FakeConnMetadata) ServerVersion() []byte {
	panic("Mock")
}

func (f FakeConnMetadata) RemoteAddr() net.Addr {
	panic("Mock")
}

func (f FakeConnMetadata) LocalAddr() net.Addr {
	panic("Mock")
}

func TestSignSSHUserCertificate(t *testing.T) {
	a := getAuthority(t)
	s, err := a.Sign(TEST2_PUBKEY, false, time.Minute, "first-name", []string { "principal1", "principal2", "principal3" })
	if err != nil {
		t.Error(err)
	} else if !strings.HasPrefix(s, "ssh-rsa-cert-v01@openssh.com ") || !strings.HasSuffix(s, "\n") {
		t.Error("Invalid wrapping on certificate")
	} else {
		bin_str := s[len("ssh-rsa-cert-v01@openssh.com "):len(s)-1]
		bin, err := base64.StdEncoding.DecodeString(bin_str)
		if err != nil {
			t.Errorf("Failed to parse base64 data (%s): %s", err, bin_str)
		} else {
			cert_untyped, err := ssh.ParsePublicKey(bin)
			if err != nil {
				t.Error(err)
			} else {
				cert := cert_untyped.(*ssh.Certificate)
				checker := ssh.CertChecker{
					IsUserAuthority: func(auth ssh.PublicKey) bool {
						return arePublicKeysEqual(auth, a.key.PublicKey())
					},
				}
				for _, princ := range []string {"principal1", "principal2", "principal3"} {
					_, err := checker.Authenticate(FakeConnMetadata{princ}, cert)
					if err != nil {
						t.Error(err)
					}
				}
				_, err := checker.Authenticate(FakeConnMetadata{"principal4"}, cert)
				if err == nil {
					t.Error("Should not have succeeded.")
				}
				tempkey, err := parseSingleSSHKey([]byte(TEST2_PUBKEY))
				if err != nil {
					t.Error(err)
				} else if !arePublicKeysEqual(cert.Key, tempkey) {
					t.Error("Certificate key mismatch")
				}
				if cert.KeyId != "first-name" {
					t.Error("Mismatch on keyid")
				}
				if cert.Serial == 0 || len(cert.Nonce) < 16 || bytes.Count(cert.Nonce, []byte { 0 }) > 4 {
					t.Error("Lacking populated fields: %d %s", cert.Serial, cert.Nonce)
				}
			}
		}
	}
}

func TestSignSSHHostCertificate(t *testing.T) {
	a := getAuthority(t)
	s, err := a.Sign(TEST2_PUBKEY, true, time.Minute, "first-name", []string { "host1", "host2", "host3" })
	if err != nil {
		t.Error(err)
	} else if !strings.HasPrefix(s, "ssh-rsa-cert-v01@openssh.com ") || !strings.HasSuffix(s, "\n") {
		t.Error("Invalid wrapping on certificate")
	} else {
		bin_str := s[len("ssh-rsa-cert-v01@openssh.com "):len(s)-1]
		bin, err := base64.StdEncoding.DecodeString(bin_str)
		if err != nil {
			t.Errorf("Failed to parse base64 data (%s): %s", err, bin_str)
		} else {
			cert_untyped, err := ssh.ParsePublicKey(bin)
			if err != nil {
				t.Error(err)
			} else {
				cert := cert_untyped.(*ssh.Certificate)
				checker := ssh.CertChecker{
					IsHostAuthority: func(auth ssh.PublicKey, addr string) bool {
						return arePublicKeysEqual(auth, a.key.PublicKey())
					},
				}
				for _, princ := range []string {"host1:22", "host2:22", "host3:22"} {
					err := checker.CheckHostKey(princ, nil, cert)
					if err != nil {
						t.Error(err)
					}
				}
				err := checker.CheckHostKey("host4:22", nil, cert)
				if err == nil {
					t.Error("Should not have succeeded.")
				}
				tempkey, err := parseSingleSSHKey([]byte(TEST2_PUBKEY))
				if err != nil {
					t.Error(err)
				} else if !arePublicKeysEqual(cert.Key, tempkey) {
					t.Error("Certificate key mismatch")
				}
				if cert.KeyId != "first-name" {
					t.Error("Mismatch on keyid")
				}
				if cert.Serial == 0 || len(cert.Nonce) < 16 || bytes.Count(cert.Nonce, []byte { 0 }) > 4 {
					t.Error("Lacking populated fields: %d %s", cert.Serial, cert.Nonce)
				}
			}
		}
	}
}

func TestSSHNegativeLifespan(t *testing.T) {
	a := getAuthority(t)
	_, err := a.Sign(TEST2_PUBKEY, false, time.Hour, "name", []string { "princ" })
	if err != nil {
		t.Error("Positive lifespan should have signed")
	}
	_, err = a.Sign(TEST2_PUBKEY, false, 0, "name", []string { "princ" })
	if err == nil {
		t.Error("Zero lifespan should have failed to sign")
	} else if !strings.Contains(err.Error(), "Lifespan") {
		t.Error("Error should have talked about lifespans")
	}
	_, err = a.Sign(TEST2_PUBKEY, false, -time.Hour, "name", []string { "princ" })
	if err == nil {
		t.Error("Negative lifespan should have failed to sign")
	} else if !strings.Contains(err.Error(), "Lifespan") {
		t.Error("Error should have talked about lifespans")
	}
}

type FakeSigner struct {
}

func (f *FakeSigner) Type() string {
	return "FakeKey"
}

func (f *FakeSigner) Marshal() []byte {
	return []byte("test-marshal")
}

func (f *FakeSigner) Verify(data []byte, sig *ssh.Signature) error {
	return errors.New("Verify mock")
}

func (f *FakeSigner) PublicKey() ssh.PublicKey {
	return f
}

func (f *FakeSigner) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	return nil, errors.New("Mocked failure")
}

func TestSSHSigningFailed(t *testing.T) {
	a := &SSHAuthority{key: &FakeSigner{}, pubkey: []byte("fake")}
	_, err := a.Sign(TEST2_PUBKEY, false, time.Minute, "name", []string { "princ" })
	if err == nil || err.Error() != "Mocked failure" {
		t.Errorf("Expected mocked failure error, but got: %s", err)
	}
}

func TestSSHInvalidSSHKeyToSign(t *testing.T) {
	a := getAuthority(t)
	_, err := a.Sign("invalid key", false, time.Minute, "name", []string { "princ" })
	if err == nil || err.Error() != "ssh: no key found" {
		t.Errorf("Expected no key found failure error, but got: %s", err)
	}
	_, err = a.Sign(TEST2_PUBKEY + "\nbad suffix", false, time.Minute, "name", []string { "princ" })
	if err == nil || !strings.Contains(err.Error(), "Trailing data") {
		t.Errorf("Expected no key found failure error, but got: %s", err)
	}
}

func TestMismatchSSHKeys(t *testing.T) {
	_, err := LoadSSHAuthority([]byte(TEST_PRIVKEY), []byte(TEST2_PUBKEY))
	if err == nil {
		t.Errorf("Should not be able to create mismatched authority")
	}
	_, err = LoadSSHAuthority([]byte(TEST_PRIVKEY), []byte(TEST3_PUBKEY))
	if err == nil {
		t.Errorf("Should not be able to create mismatched authority")
	}
}

func TestMissingSSHKey(t *testing.T) {
	_, err := LoadSSHAuthority([]byte(TEST_PRIVKEY), []byte(TEST_PUBKEY))
	if err != nil {
		t.Errorf("Should be able to create normal authority")
	}
	_, err = LoadSSHAuthority([]byte{}, []byte(TEST_PUBKEY))
	if err == nil {
		t.Errorf("Should not be able to create malformed authority")
	}
	_, err = LoadSSHAuthority([]byte(TEST_PRIVKEY), []byte{})
	if err == nil {
		t.Errorf("Should not be able to create malformed authority")
	}
}

func TestMalformedSSHKey(t *testing.T) {
	_, err := LoadSSHAuthority([]byte(TEST_PRIVKEY), []byte(TEST_PUBKEY))
	if err != nil {
		t.Errorf("Should be able to create normal authority")
	}
	_, err = LoadSSHAuthority([]byte("." + TEST_PRIVKEY), []byte(TEST_PUBKEY))
	if err == nil {
		t.Errorf("Should not be able to create malformed authority")
	}
	_, err = LoadSSHAuthority([]byte(TEST_PRIVKEY), []byte(strings.Replace(TEST_PUBKEY, "Z", "Y", 1)))
	if err == nil {
		t.Errorf("Should not be able to create malformed authority")
	}
}
