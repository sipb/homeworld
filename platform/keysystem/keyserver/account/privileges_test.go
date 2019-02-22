package account

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/token"
)

const (
	TEST_TLSCERT = "-----BEGIN CERTIFICATE-----\nMIIC8TCCAdmgAwIBAgIJAPQ/RJLL04WxMA0GCSqGSIb3DQEBCwUAMA8xDTALBgNV\nBAMMBHRlc3QwHhcNMTcwODA2MDMzODM4WhcNMTcwOTA1MDMzODM4WjAPMQ0wCwYD\nVQQDDAR0ZXN0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwB+3zCKU\nUkUtpeq8MBt6kmblE7CAUsBEtJGGg/esfg8w+hBIXp7DivudXgCcXxJJ3dVpJJKW\nRIlxWSBGyyNfSVig35cOeFl5n1b7K6RIfbtW5Mh2MqGMwkV9Of00sIZSGP3Ql1Ox\nfG1AFuWuIYVAXZ8R4y5sFpGqH4KHOrH/dc2FuaeQ1nyIEZzrfoChmdLycrqPavRT\nKnAHrB5sTtjvBCELzJd0PmVyTpTacnJHf2GpWU1iFWB0UYKxDNUWtYj8dBfbLR4e\n90ojzNCCkSYwMEGmQ41SgfbaUDotqq4DcD7qVqXPxmmsAilSvO+P/I6XThgOQ/9j\n3KF/SS/1AhsiLQIDAQABo1AwTjAdBgNVHQ4EFgQU4yZVYno76oZTa1u+55GL7boX\nQwEwHwYDVR0jBBgwFoAU4yZVYno76oZTa1u+55GL7boXQwEwDAYDVR0TBAUwAwEB\n/zANBgkqhkiG9w0BAQsFAAOCAQEAG0mGjtGFGdocvfqQtClRcYWaetMBR0e1NchS\nbaiy1it5BSdjrHTR6HoAGEJhPOKcJEphM7kjKRuXzhPYGqFzgzli8NY3w3qlV/ls\n/BAfFBsA2p5aRAYW1buLlj9EbV/BNyu3Rbs4EF6an0t/CWDlvw/VKFOPbic6j5sW\ne23rwwHkSQiIldQe29QeSNguqdNgGt2HSoc49TFLIbfsImfb1B6rz3ttsl+ownX4\nqPNt25Yzac9eELe5XfMRLEucwxmSoS47VrJaDHHZVTGYLNb0VEMwqSyKvJY8+3tC\neoMF8TPIXXgnhX2Xv2rnG1k0/h8w8uzQF+y7aOo/5nCyk903yw==\n-----END CERTIFICATE-----"
	TEST_PRIVKEY = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAwB+3zCKUUkUtpeq8MBt6kmblE7CAUsBEtJGGg/esfg8w+hBI\nXp7DivudXgCcXxJJ3dVpJJKWRIlxWSBGyyNfSVig35cOeFl5n1b7K6RIfbtW5Mh2\nMqGMwkV9Of00sIZSGP3Ql1OxfG1AFuWuIYVAXZ8R4y5sFpGqH4KHOrH/dc2FuaeQ\n1nyIEZzrfoChmdLycrqPavRTKnAHrB5sTtjvBCELzJd0PmVyTpTacnJHf2GpWU1i\nFWB0UYKxDNUWtYj8dBfbLR4e90ojzNCCkSYwMEGmQ41SgfbaUDotqq4DcD7qVqXP\nxmmsAilSvO+P/I6XThgOQ/9j3KF/SS/1AhsiLQIDAQABAoIBAGaTCKBWffYGtT52\nOw88PI7ZnMiMXZbQzF3TrIvcuh17otx/wQOzpBcaC4TasqIXs5Rako5SLSRedUPu\ndZ2TPxZ72ThHABTFQKgP2n4Mch+e29++H00c73fxfdBuHal5rW9mY+3HY2VZNvSC\noBuJdzoE6ZnveQn7r2avW9+8lPWhXOLXkX2FNnsWDyDPI2B2V5TrNEPvBtzgCC0B\nRS++mP90OfUx2lzKiQdLzJXvZz9/aDKlxQvBiniOoU1Y9wJvjcgRzGxR4Ie0g4pr\n0ATCZ4EsxGQ7ji0S97HQfKBpeaVI+4SwoxRKzT9CDySzKR1UPt2tCQx8cRnGsdUq\n+TkS78ECgYEA90UJbP0t43lHFmsr8XY3WPOBLQjwp/gXMx9Krl8ewTGiAVbzgSq+\nLhbB0MSZ/nZAb0QD9pLgxe5S2dDwXuf4LdCM4NXvpKIH6piewE1E9GScqdhUZ8//\nAZS1xIhW9oWDWFVcorzx/KT8S7aYLh1RZAwFPcJST0ScDBm1QjyCI0kCgYEAxug+\nDxd2RLgVm4w9Qngzqa+q3gVz5WgOqe6T9NTCu9UuAkx3GnajK64fxdf1P950tX7R\nhXyCeXIH8/GOncLV8gJLA7gizzEBduODnHqtJ1eL1eIsH/BtUA6OiIeKFs2YI3i7\nW2MBlmKrEi8bIEFTu7WG0VCaAJqZIGT0wwsLI8UCgYEA3OhqkVpngtA4uEirC5/n\ntqplf4x7JDU61Mth9wK4ATWMXNIH3iAHpDlkklTylymiS0VinQl/kpVmo35NIRzw\n1k15buzymgzAMdCEE510uzqf1AWW8uAaHJl1As4jkz6Yp3QrvKA9OM9VL3dD4f8D\nVfR/Qju3OWY8W3skOrbANTECgYBBSvE8MP5wtmDZa6KcVCrZU8HqGa4eqxbNL3TA\nFKtLz0HIHWOnezQ63XCumCJ4ccSr41JR2DpYNVdo+21OWiuywo/vS52Zl8OcTDji\nv95hILrVXeYQIfMwKWceaCerLpf3ZOVTrV9TB1aSpIXqA6fB4We9BBFZi2YinSE/\neTuR5QKBgQCW5lWO3l5+dshaGFr8STENKoLiTMA0bNc1JTWoUr0zc85f+LyuZfOV\nwP/ySBmN8Cj/8MqNQSIKY8ah7av/gzeFKjet1uIaj2+/sppAL1nIEOngFjT9LetT\nlZx0SIMHFp7t+2W5gxocd/kS97V+yNmkGH/JXH1yYQ2BWq8xtRPyhw==\n-----END RSA PRIVATE KEY-----"
	TEST_SSHPUB  = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDAH7fMIpRSRS2l6rww" + "G3qSZuUTsIBSwES0kYaD96x+DzD6EEhensOK+51eAJxfEknd1WkkkpZEiXFZIEbLI19JWKDflw54WXmfVvsrpEh9u1bkyHYyoYzCRX05/TSwhlIY/dCXU7F8bUAW5a4hhUBdnxHjLmwWkaofgoc6sf91zYW5p5DWfIgRnOt+gKGZ0vJyuo9q9FMqcAesHmxO2O8EIQvMl3Q+ZXJOlNpyckd/YalZTWIVYHRRgrEM1Ra1iPx0F9stHh73SiPM0IKRJjAwQaZDjVKB9tpQOi2qrgNwPupWpc/GaawCKVK874/8jpdOGA5D/2PcoX9JL/UCGyIt"

	TEST_CLIENTKEY    = "-----BEGIN RSA PRIVATE KEY-----\nMIICWwIBAAKBgQC0q6RPYtP88n+LWlup97hWb2I3bIwWiIqPR6bsUU6sB5T/mier\nx9kReFSu4346GMyv4rVzarLueipvMPcXP++LZ+sH0NQUwD2uSPe15EgRcoEEPNvV\nxsNJMvQfEfjv+1RHHPHMYJV9MJzXFRj52oyx3xK+jDG4Sm1ThI70fwJHYwIDAQAB\nAoGAClAl7/YnPbAmAbFlvB0M47o19A35LSwcJLOlXqYBhKZmJfUJwK+Gv42L3/PS\nd8SEoqGhU/ZKQnyswW4dHLGkncr+RAAQ5UGRUHr7wsP1c+9kZpkaj1hmyLQvEbL6\nLPFxvno6AGxbURznIBu1hCQUu0aS/QZYfpaYrjo/9N3dg7ECQQDe4HAsUMYah+3b\nGu2q2oTqFOdLU+LA7ZloX338uIXbXCwiZz43b40uNqoXYXRZQQB7qT+zwseqDXWZ\npmWjBTeZAkEAz4VtH9Ug511V7idjlOe0k1kois4ydfvurniUoBtDE6xKD6dR/EZ6\nf5yCVfM0GZAq+BgomYKEBTklo1EuUMYkWwJAF7M0GnJIbp/PukHlzgpIof+xDMCR\n10Qs0P1+jzYr/cSSaOIjqo9xKt3jPnM9hRQ1cfDwdjQbOUkPHVSlcC1o2QJAekup\nWZ8ievbYUzdHSlOaaVObvuFxf3Ju4McS35/xUcCxDLSQblmii13SuZBP3djGWdry\n4jS2VNWuxqZq4xNCDQJAEZ7djTVtLEghjof27CuXMkopZZ4RhYTsAbZAwMBNBhds\npQQS+O5pIVDD8ou3QfifB6G5OmZr0PaKld/99H52LA==\n-----END RSA PRIVATE KEY-----"
	TEST_TLSCLIENTCSR = "-----BEGIN CERTIFICATE REQUEST-----\nMIIBVTCBvwIBADAWMRQwEgYDVQQDDAtjbGllbnQtdGVzdDCBnzANBgkqhkiG9w0B\nAQEFAAOBjQAwgYkCgYEAtKukT2LT/PJ/i1pbqfe4Vm9iN2yMFoiKj0em7FFOrAeU\n/5onq8fZEXhUruN+OhjMr+K1c2qy7noqbzD3Fz/vi2frB9DUFMA9rkj3teRIEXKB\nBDzb1cbDSTL0HxH47/tURxzxzGCVfTCc1xUY+dqMsd8SvowxuEptU4SO9H8CR2MC\nAwEAAaAAMA0GCSqGSIb3DQEBCwUAA4GBALCOKX+QHmNLGrrSCWB8p2iMuS+aPOcW\nYI9c1VaaTSQ43HOjF1smvGIa1iicM2L5zTBOEG36kI+sKFDOF2cXclhQF1WfLcxC\nIi/JSV+W7hbS6zWvJOnmoi15hzvVa1MRk8HZH+TpiMxO5uqQdDiEkV1sJ50v0ZtR\nTMuSBjdmmJ1t\n-----END CERTIFICATE REQUEST-----"
	TEST_SSHCLIENTPUB = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQC0q6RPYtP88n+LWlup97hWb2I3bIwWiIqPR6bsUU6sB5T/mierx9kReFSu4346GMyv4rVzarLueipvMPcXP++LZ+sH0NQUwD2uSPe15EgRcoEEPNvVxsNJMvQfEfjv+1RHHPHMYJV9MJzXFRj52oyx3xK+jDG4Sm1ThI70fwJHYw=="
)

func getTLSAuthority(t *testing.T) authorities.Authority {
	authority, err := authorities.LoadTLSAuthority([]byte(TEST_PRIVKEY), []byte(TEST_TLSCERT))
	if err != nil {
		t.Fatalf("Could not create TLS authority: %s", err)
	}
	return authority
}

func getSSHAuthority(t *testing.T) authorities.Authority {
	authority, err := authorities.LoadSSHAuthority([]byte(TEST_PRIVKEY), []byte(TEST_SSHPUB))
	if err != nil {
		t.Fatalf("Could not create SSH authority: %s", err)
	}
	return authority
}

func TestTLSGrantPrivilege(t *testing.T) {
	authority := getTLSAuthority(t)
	priv, err := NewTLSGrantPrivilege(authority, false, time.Hour, "test-cert", []string{"localhost"})
	if err != nil {
		t.Error(err)
	} else {
		cert, err := priv(nil, TEST_TLSCLIENTCSR)
		if err != nil {
			t.Error(err)
		}
		// check well-formation
		_, err = tls.X509KeyPair([]byte(cert), []byte(TEST_CLIENTKEY))
		if err != nil {
			t.Error(err)
		}
	}
}

func TestTLSGrantPrivilege_ShortLifetime(t *testing.T) {
	authority := getTLSAuthority(t)
	_, err := NewTLSGrantPrivilege(authority, false, time.Millisecond*500, "test-cert", []string{"localhost"})
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "parameter") {
		t.Error("Expected error about bad parameter, not", err)
	}
}

func TestTLSGrantPrivilege_NoAuthority(t *testing.T) {
	_, err := NewTLSGrantPrivilege(nil, false, time.Hour, "test-cert", []string{"localhost"})
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "parameter") {
		t.Error("Expected error about bad parameter, not", err)
	}
}

func TestTLSGrantPrivilege_NoKeyID(t *testing.T) {
	authority := getTLSAuthority(t)
	_, err := NewTLSGrantPrivilege(authority, false, time.Hour, "", []string{"localhost"})
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "parameter") {
		t.Error("Expected error about bad parameter, not", err)
	}
}

func TestTLSGrantPrivilege_WithSSH(t *testing.T) {
	authority := getSSHAuthority(t)
	_, err := NewTLSGrantPrivilege(authority, false, time.Hour, "test-cert", []string{"localhost"})
	if err == nil {
		t.Error("Expected error about wrong authority type")
	} else if !strings.Contains(err.Error(), "expects a TLS authority") {
		t.Error("Expected error about wrong authority type, not", err)
	}
}

// Some of these are copied from ssh.go and ssh_test.go

func arePublicKeysEqual(pk1 ssh.PublicKey, pk2 ssh.PublicKey) bool {
	if pk1.Type() != pk2.Type() {
		return false
	} else {
		return bytes.Equal(pk1.Marshal(), pk2.Marshal())
	}
}

func parseSingleSSHKey(data []byte) (ssh.PublicKey, error) {
	pubkey, _, _, rest, err := ssh.ParseAuthorizedKey(data)
	if err != nil {
		return nil, err
	}
	if rest != nil && len(rest) > 0 {
		return nil, fmt.Errorf("Trailing data after end of public key file")
	}
	return pubkey, nil
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

func checkSSHCert(t *testing.T, authorityPubkeyStr []byte, certstr string, pubstr string) {
	if !strings.HasPrefix(certstr, "ssh-rsa-cert-v01@openssh.com ") || !strings.HasSuffix(certstr, "\n") {
		t.Fatal("Invalid wrapping on certificate")
	}
	bin_str := certstr[len("ssh-rsa-cert-v01@openssh.com ") : len(certstr)-1]
	bin, err := base64.StdEncoding.DecodeString(bin_str)
	if err != nil {
		t.Fatalf("Failed to parse base64 data (%s): %s", err, bin_str)
	}
	cert_untyped, err := ssh.ParsePublicKey(bin)
	if err != nil {
		t.Fatal(err)
	}
	authorityPubkey, _, _, _, err := ssh.ParseAuthorizedKey(authorityPubkeyStr)
	if err != nil {
		t.Fatal(err)
	}
	cert := cert_untyped.(*ssh.Certificate)
	checker := ssh.CertChecker{
		IsUserAuthority: func(auth ssh.PublicKey) bool {
			return arePublicKeysEqual(auth, authorityPubkey)
		},
	}
	for _, princ := range []string{"principal1", "principal2", "principal3"} {
		_, err := checker.Authenticate(FakeConnMetadata{princ}, cert)
		if err != nil {
			t.Error(err)
		}
	}
	_, err = checker.Authenticate(FakeConnMetadata{"principal4"}, cert)
	if err == nil {
		t.Error("Should not have succeeded.")
	}
	tempkey, err := parseSingleSSHKey([]byte(pubstr))
	if err != nil {
		t.Error(err)
	} else if !arePublicKeysEqual(cert.Key, tempkey) {
		t.Error("Certificate key mismatch")
	}
}

func TestSSHGrantPrivilege(t *testing.T) {
	authority := getSSHAuthority(t)
	priv, err := NewSSHGrantPrivilege(authority, false, time.Hour, "test-cert", []string{"principal1", "principal2", "principal3"})
	if err != nil {
		t.Error(err)
	} else {
		cert, err := priv(nil, TEST_SSHCLIENTPUB)
		if err != nil {
			t.Error(err)
		}
		// check well-formation
		checkSSHCert(t, authority.GetPublicKey(), cert, TEST_SSHCLIENTPUB)
	}
}

func TestSSHGrantPrivilege_ShortLifetime(t *testing.T) {
	authority := getSSHAuthority(t)
	_, err := NewSSHGrantPrivilege(authority, false, time.Millisecond*500, "test-cert", []string{"localhost"})
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "parameter") {
		t.Error("Expected error about bad parameter, not", err)
	}
}

func TestSSHGrantPrivilege_NoAuthority(t *testing.T) {
	_, err := NewSSHGrantPrivilege(nil, false, time.Hour, "test-cert", []string{"localhost"})
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "parameter") {
		t.Error("Expected error about bad parameter, not", err)
	}
}

func TestSSHGrantPrivilege_NoNames(t *testing.T) {
	authority := getSSHAuthority(t)
	_, err := NewSSHGrantPrivilege(authority, false, time.Hour, "test-cert", nil)
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "parameter") {
		t.Error("Expected error about bad parameter, not", err)
	}
	_, err = NewSSHGrantPrivilege(authority, false, time.Hour, "test-cert", []string{})
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "parameter") {
		t.Error("Expected error about bad parameter, not", err)
	}
}

func TestSSHGrantPrivilege_NoKeyID(t *testing.T) {
	authority := getSSHAuthority(t)
	_, err := NewSSHGrantPrivilege(authority, false, time.Hour, "", []string{"localhost"})
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "parameter") {
		t.Error("Expected error about bad parameter, not", err)
	}
}

func TestSSHGrantPrivilege_WithSSH(t *testing.T) {
	authority := getTLSAuthority(t)
	_, err := NewSSHGrantPrivilege(authority, false, time.Hour, "test-cert", []string{"localhost"})
	if err == nil {
		t.Error("Expected error about wrong authority type")
	} else if !strings.Contains(err.Error(), "expects a SSH authority") {
		t.Error("Expected error about wrong authority type, not", err)
	}
}

func TestBootstrapPrivilege(t *testing.T) {
	registry := token.NewTokenRegistry()
	priv, err := NewBootstrapPrivilege([]string{"princ1", "princ2", "princ3"}, time.Millisecond*13, registry)
	if err != nil {
		t.Fatal(err)
	}
	checklose1, err := priv(nil, "princ1")
	if err != nil {
		t.Error(err)
	}
	checklose2, err := priv(nil, "princ1")
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 5; i++ {
		for _, princ := range []string{"princ3", "princ2", "princ1"} {
			tok, err := priv(nil, princ)
			if err != nil {
				t.Error(err)
			} else {
				stok, err := registry.LookupToken(tok)
				if err != nil {
					t.Error(err)
				} else {
					if stok.Claim() != nil {
						t.Error("Should have been able to claim token!")
					}
				}
			}
		}
		_, err := priv(nil, "princ4")
		if err == nil {
			t.Error("Expected error when requesting disallowed principal")
		} else if !strings.Contains(err.Error(), "not allowed") {
			t.Error("Expected error about disallowed principal, not", err)
		}
	}
	regain1, err := registry.LookupToken(checklose1)
	if err != nil {
		t.Error(err)
	} else if regain1.Claim() != nil {
		t.Error("Should have been able to claim")
	}
	time.Sleep(13 * time.Millisecond)
	regain2, err := registry.LookupToken(checklose2)
	if err != nil {
		t.Error(err)
	} else if regain2.Claim() == nil {
		t.Error("Should not have been able to claim")
	}
}

func TestBootstrapPrivilege_NoPrincipals(t *testing.T) {
	registry := token.NewTokenRegistry()
	_, err := NewBootstrapPrivilege(nil, time.Hour, registry)
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "at least one allowed principal") {
		t.Error("Expected error about bad parameter, not", err)
	}
	_, err = NewBootstrapPrivilege([]string{}, time.Hour, registry)
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "at least one allowed principal") {
		t.Error("Expected error about bad parameter, not", err)
	}
}

func TestBootstrapPrivilege_NoLifespan(t *testing.T) {
	registry := token.NewTokenRegistry()
	_, err := NewBootstrapPrivilege([]string{"principal"}, time.Nanosecond, registry)
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "parameter") {
		t.Error("Expected error about bad parameter, not", err)
	}
}

func TestBootstrapPrivilege_NoRegistry(t *testing.T) {
	_, err := NewBootstrapPrivilege([]string{"principal"}, time.Hour, nil)
	if err == nil {
		t.Error("Expected error about bad parameter")
	} else if !strings.Contains(err.Error(), "parameter") {
		t.Error("Expected error about bad parameter, not", err)
	}
}

func TestImpersonatePrivilege(t *testing.T) {
	test_account := &Account{Principal: "testy-tester"}
	broken_account := &Account{Principal: "wrong-name"}
	scope := &Group{AllMembers: []string{"testy-tester", "missing-account", "broken-account"}}
	get_account := func(name string) (*Account, error) {
		if name == "testy-tester" {
			return test_account, nil
		} else if name == "broken-account" {
			return broken_account, nil
		} else {
			return nil, fmt.Errorf("No such test user %s", name)
		}
	}
	priv, err := NewImpersonatePrivilege(get_account, scope)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &OperationContext{Account: nil}
	s, err := priv(ctx, "invalid-account")
	if err == nil {
		t.Error("Expected scope failure")
	} else if !strings.Contains(err.Error(), "outside of allowed scope") {
		t.Errorf("Expected disallowed scope error, not: %s", err)
	}
	s, err = priv(ctx, "missing-account")
	if err == nil {
		t.Error("Expected account failure")
	} else if err.Error() != "No such test user missing-account" {
		t.Errorf("Expected account error, not: %s", err)
	}
	s, err = priv(ctx, "broken-account")
	if err == nil {
		t.Error("Expected account mismatch")
	} else if !strings.Contains(err.Error(), "Wrong account") {
		t.Errorf("Expected account mismatch, not: %s", err)
	}
	s, err = priv(ctx, "testy-tester")
	if err != nil {
		t.Error("Expected to delegate account successfully, not:", err)
	} else if s != "" {
		t.Error("Expected no result from delegation")
	} else if ctx.Account != test_account {
		t.Error("Expected matching account pointer")
	}
}

func TestDelegateAuthorityPrivilege_NoScope(t *testing.T) {
	get_account := func(name string) (*Account, error) {
		return nil, nil
	}
	_, err := NewImpersonatePrivilege(get_account, nil)
	if err == nil || !strings.Contains(err.Error(), "Missing parameter") {
		t.Errorf("Expected bad parameter error, not %v", err)
	}
}

func TestDelegateAuthorityPrivilege_NoAccessor(t *testing.T) {
	scope := &Group{AllMembers: []string{"testy-tester"}}
	_, err := NewImpersonatePrivilege(nil, scope)
	if err == nil || !strings.Contains(err.Error(), "Missing parameter") {
		t.Errorf("Expected bad parameter error, not %v", err)
	}
}

func TestConfigurationPrivilege(t *testing.T) {
	for _, contents := range []string{"abcd", "", "NOATHUSEOCUH", "AOE AE", "more text"} {
		priv, err := NewConfigurationPrivilege(contents)
		if err != nil {
			t.Error(err)
		}
		nc, err := priv(nil, "")
		if err != nil {
			t.Error(err)
		}
		if nc != contents {
			t.Errorf("Contents mismatch: %s instead of %s", nc, contents)
		}
	}
}

func TestConfigurationPrivilegeFail(t *testing.T) {
	for _, contents := range []string{"abcd", "", "NOATHUSEOCUH", "AOE AE", "more text"} {
		priv, err := NewConfigurationPrivilege(contents)
		if err != nil {
			t.Error(err)
		}
		_, err = priv(nil, "a")
		if err == nil {
			t.Error("Expected error from providing unnecessary parameter")
		}
	}
}
