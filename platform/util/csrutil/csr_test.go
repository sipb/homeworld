package csrutil

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/sipb/homeworld/platform/util/testutil"
	"github.com/sipb/homeworld/platform/util/wraputil"
)

const (
	SSH_TEST_PRIVKEY = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAwqhw4zh0nlccLF1vpdj2iGfHYOwb3T9EIlt/Axv44mQlLNjs\nY7dgSQ+qqI5XahRBxVo4SfQDT4ptnFs6whklWrHBGCNHH/oGAh0Tt5to3So75TZm\nLdefI4Fq7BgMEZARrtS45iWlVQkscMeyMdUKp4CCHeqYM673RdKtC/0IgdhIn0f3\nUqjTzyKq/kuDgfIM9nTf+UceJdqiqRHtD+G00zUTE5jrSGCjgHYMqwS2oVWxnSc5\nNF/n5KDJxaAh7ji76/H4i0SVloDrn4z3AK/F5mUieo2s3qM70k/2M82VLo5fSe7m\nA+sn78wQhycbD705+VLCzRlp2NNt/4hXDQQdpwIDAQABAoIBAQCyIn8cEJ/3/vOT\nTfZMKV6CuaXldmyRbcImRuDWsZRzaP30Kpc5Maj1c1bTZV9tfhgqSEPWuW0GL0Hb\nokkFMwnSE3UHZ9FA3Ab/jChtD9VI/8tMGRosvXOuhFKat+7ja5ojChwi0TSZuwlm\nM/lITROw8ZMhWXvrYCR9Syx9GhPc7br7zDaGerS2MZRkfzCOgvRE3tm0rIH6yl+K\nuLFD3r1KaN2ay/b0I7jf9/BMK1poGEGUgP3SePs0OyjUd2sZ+G3wQYtEyOcegg0E\nH8mGoBWSWHBAa8yDbMw3o7Tcn88UphHepwdlmSxrYXP1WP7DwY6bngc5PPPnDcTp\nIKUu5mmhAoGBAPwoVFT9adBuyz1G5PuJnPRl0uV/Ua9x+Wex2TY+oJ/cFIPyAw48\npXcjCIDy3fenxWXqZR3CaTLbNNCFx9DEQdM3JmP7d+wGznTPF0v/EToSgpBizNPS\nHAo96vFTms4bYZEo/ts8waEahHsKyUUgx54JfaqtPX7c6We5YWdgPWNvAoGBAMWf\nzgkf29vn4ftsGwz8oO1VjTshHxch/SuuA/a/s7qyyZ4u9MvcPsr+LUCTb2BpmdDd\nPG7WUlYmqwg7xZJSN/YqjHoCmKEp1AikpOem0jP0RBYuwHoNWcCokWR0eadBY70g\n22QMC4toF23pykf3yQtU4e4rBlVjvAKS2/ZyW+1JAoGAD7ib+WiLTll6BmoDIMOl\nq38ltPVJLH0YpaRq/HzPGuhnxwoxspOJZXIjt5ZszGIDZqVEhKR4VplgI5gTqypx\nSC/qDtXA1lBeUt4Of8h5VHuO9F2Uk6hH40OVAFLMFgmS/a/mo9iX4el7VQiJH+w5\nRdsloJyIdv5i9vqR3hYb/bUCgYBeE4rjcRUahDJhm77s2b5J/PX0dfn06ys4BejB\nJ9UJRV8RPE0wVrJVs9Ya7ZSRkvO0J/1Czif39wRoMPwGgbk+KFcjJeU+o0jarHYM\nCK/8J4XaAXuDHqPhQN2lsoTPCCPQvrlx0QIV5QFyQ18WD3DXQhsjY7vqHkY7+2lW\n0m3McQKBgQD7kQdYzeX51HMFxDnKFHQ/mEwLTj0af8BChZvH/RP0VZZkJDHaXbia\nUh8Q/WWlFRUkY5EV/K8uvrUSwkgMuqHltRRXhmLmXd3l8zCikLYyp/Eb+1Vgi4fZ\n60hYAUcbr/Q23FMrW8WoavKwvFW2VG/lQagquny0Qt8dXwFG5wIxqA==\n-----END RSA PRIVATE KEY-----\n"
	SSH_TEST_PUBKEY  = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDCqHDjOHSeVxwsXW+l2PaIZ8dg7BvdP0QiW38DG/jiZCUs2Oxjt2BJD6qojldqFEHFWjhJ9ANPim2cWzrCGSVascEYI0cf+gYCHRO3m2jdKjvlNmYt158jgWrsGAwRkBGu1LjmJaVVCSxwx7Ix1QqngIId6pgzrvdF0q0L/QiB2EifR/dSqNPPIqr+S4OB8gz2dN/5Rx4l2qKpEe0P4bTTNRMTmOtIYKOAdgyrBLahVbGdJzk0X+fkoMnFoCHuOLvr8fiLRJWWgOufjPcAr8XmZSJ6jazeozvST/YzzZUujl9J7uYD6yfvzBCHJxsPvTn5UsLNGWnY023/iFcNBB2n user@hyades"
	SSH_TEST_CERT    = "ssh-rsa-cert-v01@openssh.com AAAAHHNzaC1yc2EtY2VydC12MDFAb3BlbnNzaC5jb20AAAAgMyh6jYuu1dB+aqNbJfl/9BqAtpZKvO8wXltqsmZpAkYAAAADAQABAAABAQDCqHDjOHSeVxwsXW+l2PaIZ8dg7BvdP0QiW38DG/jiZCUs2Oxjt2BJD6qojldqFEHFWjhJ9ANPim2cWzrCGSVascEYI0cf+gYCHRO3m2jdKjvlNmYt158jgWrsGAwRkBGu1LjmJaVVCSxwx7Ix1QqngIId6pgzrvdF0q0L/QiB2EifR/dSqNPPIqr+S4OB8gz2dN/5Rx4l2qKpEe0P4bTTNRMTmOtIYKOAdgyrBLahVbGdJzk0X+fkoMnFoCHuOLvr8fiLRJWWgOufjPcAr8XmZSJ6jazeozvST/YzzZUujl9J7uYD6yfvzBCHJxsPvTn5UsLNGWnY023/iFcNBB2nAAAAAAAAAAAAAAACAAAABHRlc3QAAAAAAAAAAAAAAAD//////////wAAAAAAAAAAAAAAAAAAARcAAAAHc3NoLXJzYQAAAAMBAAEAAAEBAMKocOM4dJ5XHCxdb6XY9ohnx2DsG90/RCJbfwMb+OJkJSzY7GO3YEkPqqiOV2oUQcVaOEn0A0+KbZxbOsIZJVqxwRgjRx/6BgIdE7ebaN0qO+U2Zi3XnyOBauwYDBGQEa7UuOYlpVUJLHDHsjHVCqeAgh3qmDOu90XSrQv9CIHYSJ9H91Ko088iqv5Lg4HyDPZ03/lHHiXaoqkR7Q/htNM1ExOY60hgo4B2DKsEtqFVsZ0nOTRf5+SgycWgIe44u+vx+ItElZaA65+M9wCvxeZlInqNrN6jO9JP9jPNlS6OX0nu5gPrJ+/MEIcnGw+9OflSws0ZadjTbf+IVw0EHacAAAEPAAAAB3NzaC1yc2EAAAEAJs0SNZlyv1QQTuX80a5AJ7ptfpiAaMaM9JupGuW1BJqZUMB2t+Eqzfu2JFwSAp7UBYYQ00gNwhoOep/rZYl/E+wV+7ppjM39si+kF1RUAaOBSHHQzIZA/aCZbS/OomeeSDJF3Y10ZT5KTX1uk85oOC5k+uVo/S3rTF4JegWnRc00PtsaYruqY6DMx2OuPVS+6pqzKshUi+9Ofc9iu5cD/anmNvQAY5inCcgJmgxZNHiRJdvvjmr0Y8cmIOkigWzQ0UpkBArjUOQX0COJ7Ph94x2nFtTCNfVafGEIEYJ7vS3YfrXbk3A5EzJzU9fuuKnqjv1zF5RRM5iZw31ych0C2g== test.key.pub"

	TLS_TEST_KEY = "-----BEGIN RSA PRIVATE KEY-----\nMIICWwIBAAKBgQC0q6RPYtP88n+LWlup97hWb2I3bIwWiIqPR6bsUU6sB5T/mier\nx9kReFSu4346GMyv4rVzarLueipvMPcXP++LZ+sH0NQUwD2uSPe15EgRcoEEPNvV\nxsNJMvQfEfjv+1RHHPHMYJV9MJzXFRj52oyx3xK+jDG4Sm1ThI70fwJHYwIDAQAB\nAoGAClAl7/YnPbAmAbFlvB0M47o19A35LSwcJLOlXqYBhKZmJfUJwK+Gv42L3/PS\nd8SEoqGhU/ZKQnyswW4dHLGkncr+RAAQ5UGRUHr7wsP1c+9kZpkaj1hmyLQvEbL6\nLPFxvno6AGxbURznIBu1hCQUu0aS/QZYfpaYrjo/9N3dg7ECQQDe4HAsUMYah+3b\nGu2q2oTqFOdLU+LA7ZloX338uIXbXCwiZz43b40uNqoXYXRZQQB7qT+zwseqDXWZ\npmWjBTeZAkEAz4VtH9Ug511V7idjlOe0k1kois4ydfvurniUoBtDE6xKD6dR/EZ6\nf5yCVfM0GZAq+BgomYKEBTklo1EuUMYkWwJAF7M0GnJIbp/PukHlzgpIof+xDMCR\n10Qs0P1+jzYr/cSSaOIjqo9xKt3jPnM9hRQ1cfDwdjQbOUkPHVSlcC1o2QJAekup\nWZ8ievbYUzdHSlOaaVObvuFxf3Ju4McS35/xUcCxDLSQblmii13SuZBP3djGWdry\n4jS2VNWuxqZq4xNCDQJAEZ7djTVtLEghjof27CuXMkopZZ4RhYTsAbZAwMBNBhds\npQQS+O5pIVDD8ou3QfifB6G5OmZr0PaKld/99H52LA==\n-----END RSA PRIVATE KEY-----"
)

func TestBuildSSHCSR_PrivateKey(t *testing.T) {
	csr, err := BuildSSHCSR([]byte(SSH_TEST_PRIVKEY))
	if err != nil {
		t.Fatal(err)
	}
	pkfound, err := wraputil.ParseSSHTextPubkey(csr)
	if err != nil {
		t.Fatal(err)
	}
	pkexpect, err := wraputil.ParseSSHTextPubkey([]byte(SSH_TEST_PUBKEY))
	if err != nil {
		t.Fatal(err)
	}
	if pkfound.Type() != pkexpect.Type() {
		t.Error("type mismatch")
	}
	if !bytes.Equal(pkfound.Marshal(), pkexpect.Marshal()) {
		t.Errorf("Data mismatch")
	}
}

func TestBuildSSHCSR_PublicKey(t *testing.T) {
	csr, err := BuildSSHCSR([]byte(SSH_TEST_PUBKEY))
	if err != nil {
		t.Fatal(err)
	}
	pkfound, err := wraputil.ParseSSHTextPubkey(csr)
	if err != nil {
		t.Fatal(err)
	}
	pkexpect, err := wraputil.ParseSSHTextPubkey([]byte(SSH_TEST_PUBKEY))
	if err != nil {
		t.Fatal(err)
	}
	if pkfound.Type() != pkexpect.Type() {
		t.Error("type mismatch")
	}
	if !bytes.Equal(pkfound.Marshal(), pkexpect.Marshal()) {
		t.Errorf("Data mismatch")
	}
}

func TestBuildSSHCSR_NoCert(t *testing.T) {
	_, err := BuildSSHCSR([]byte(SSH_TEST_CERT))
	testutil.CheckError(t, err, "not have certificate type")
}

func TestBuildSSHCSR_NoType(t *testing.T) {
	_, err := BuildSSHCSR([]byte("neither"))
	testutil.CheckError(t, err, "could not parse key as")
}

func TestBuildSSHCSR_NoData(t *testing.T) {
	_, err := BuildSSHCSR([]byte(""))
	testutil.CheckError(t, err, "could not parse key as")
}

func TestBuildTLSCSR(t *testing.T) {
	csrdata, err := BuildTLSCSR([]byte(TLS_TEST_KEY))
	if err != nil {
		t.Fatal(err)
	}
	csr, err := wraputil.LoadX509CSRFromPEM(csrdata)
	if err != nil {
		t.Fatal(err)
	}
	err = csr.CheckSignature()
	if err != nil {
		t.Error(err)
	}
	if csr.SignatureAlgorithm != x509.SHA256WithRSA {
		t.Errorf("Wrong signature algorithm")
	}
	key, err := wraputil.LoadRSAKeyFromPEM([]byte(TLS_TEST_KEY))
	if err != nil {
		t.Fatal(err)
	}
	if key.PublicKey.N.Cmp(csr.PublicKey.(*rsa.PublicKey).N) != 0 {
		t.Error("Mismatched keys")
	}
	if !strings.HasPrefix(csr.Subject.CommonName, "invalid-") {
		t.Errorf("Expected 'invalid-' prefix")
	}
}

func TestBuildTLSCSR_NoKey(t *testing.T) {
	_, err := BuildTLSCSR([]byte("notakey"))
	testutil.CheckError(t, err, "Missing expected PEM header")
}

func TestBuildTLSCSR_NoData(t *testing.T) {
	_, err := BuildTLSCSR([]byte("notakey"))
	testutil.CheckError(t, err, "Missing expected PEM header")
}

func TestBuildTLSCSR_Invalid(t *testing.T) {
	pkey, err := rsa.GenerateKey(rand.Reader, 12)
	if err != nil {
		t.Fatal(err)
	}
	encoded := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pkey)})
	_, err = BuildTLSCSR(encoded)
	testutil.CheckError(t, err, "message too long for RSA public key size")
}
