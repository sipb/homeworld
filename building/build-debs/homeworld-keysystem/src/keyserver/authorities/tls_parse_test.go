package authorities

import (
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"strings"
	"testing"
	"util/testkeyutil"
)

const (
	TLS_TEST_ECDSA_KEY  = "-----BEGIN PRIVATE KEY-----\nMIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIArv14RWE1eJPwJMLY\nXbvXU5XufK9B/h1fp+23g2E/AqHqv8VNcQZy/a1IVFLrR8pxqE6pg4/vYiS+7BFJ\nwobGh/ahgYkDgYYABAFAIh6CwZTseH+WNsmHtl1d5BH29BjInF32fePbAPON1bXk\nTljhH2xwV5h/iJJf8zXHflxe9jIXRYF0ulL6vjV7TwHuO7MZkNi25oIwYj+rNj/j\nSWMHgdkSvSkWxfgogAWG0SxdiyUXOMeBlMZit1G3LijAubn9shTxPxNzLnHvbvVV\ngQ==\n-----END PRIVATE KEY-----"
	TLS_TEST_ECDSA_CERT = "-----BEGIN CERTIFICATE-----\nMIIB7DCCAU6gAwIBAgIJAPrNxR4/otK9MAoGCCqGSM49BAMCMA8xDTALBgNVBAMM\nBHRlc3QwHhcNMTcwODA2MDQ0NzU2WhcNMTcwOTA1MDQ0NzU2WjAPMQ0wCwYDVQQD\nDAR0ZXN0MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQBQCIegsGU7Hh/ljbJh7Zd\nXeQR9vQYyJxd9n3j2wDzjdW15E5Y4R9scFeYf4iSX/M1x35cXvYyF0WBdLpS+r41\ne08B7juzGZDYtuaCMGI/qzY/40ljB4HZEr0pFsX4KIAFhtEsXYslFzjHgZTGYrdR\nty4owLm5/bIU8T8Tcy5x7271VYGjUDBOMB0GA1UdDgQWBBTnOngkY/2UtUdDgWUu\n/UQRBt2KbTAfBgNVHSMEGDAWgBTnOngkY/2UtUdDgWUu/UQRBt2KbTAMBgNVHRME\nBTADAQH/MAoGCCqGSM49BAMCA4GLADCBhwJCAOVYNUxX44Ex9+5nVTY8R5g0OQAW\nbDUTXqhgSNSr3bKybkmVhzbzKlZqLolloJMYvSw4nSGn3nF1WsDn/xyYLR0oAkEP\nraLAbWX42jMY9EC/l2++o0pLVkZs7tLok/w1cTaoSth2IGtPtMlL8N4prIWd84O7\nqrF9anZocHGTr9iD23E8fw==\n-----END CERTIFICATE-----"

	TLS_TEST2_PRIVKEY = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpgIBAAKCAQEAuc1psL/DYin1ZLfSh2erwFg5yqJ1BlW4xPkRA4T2cFOKG8zB\nI3MfB2vJkB2Ug821vN+QxfX77CQwEQZwiIf9gAdYlC9I3TNnESipqncxq2x0Jnre\nBYCaFgru60CoLeCUwMUFXNKhkgagZSp9kAirR1LPTyThAx0s+JeC8ImfxNscQ/uf\nzOxCnZXEV8gTmROqAMCHVfSTYy9BYqjf0D1oo2XBTdAFk6pdDUcTKvRGoDs/V4Ry\n13X96z0MyfyD/8yumQegwEOd0Ay3yfOzwHgv2JhVrAduZRxjlel+W3Mk9Bw8Xy9W\n2DQgdKDQONe4g6efsqmKBXQ1tSKI+T9kCW2JtQIDAQABAoIBAQCtU/uhn/KT05KR\nZ45lNIgbgfI/nzfONg+M6NA/WT1QYg43itYtzMoIcTvyTjXqku9UB7cVhTiC/Os+\nJqS6KSqJ0dCHRGkTuU0Py8AjPtg+E4lzEDGoLmUP5RkmqwV47sW14tXy1qdVAwuD\n9JR31i559b1hFoU2E3SNX0IORESgLTLAujAnrCWvnal/O6d1Tzb9csVxj/8e2Ymw\nuONAnu6jOKyOxGt+VJAoMrZkutzhZixmdFfQyuYYV3ViLhKcgtpWOXjHvCJfJ/7o\nBCy74mHkti+i24Ssd5sgdiWOuoURjJgm3okQtTGSR4guPWkYAajLzeQYD9aSepzR\nHSIIRXWBAoGBAOxjHLpEHMIq9O0vshDcu2dq5pTEOBarVXr2sh5WBUUwZjqCuop5\n3KV/olSRENipnYYOIHiDMDSjcB05cFApmKmi7CIQOOnNSgZjpzdsKXQKEWLfEO2S\nYJ+lybcSYaKxONIAPU9PISxZgI6jYSc3nb/uEDStERm4cnB6MV2cXanhAoGBAMk3\n4ACoACgov/9wY0OHyXRfvEflo9AJvUHXgRvopX06lMSDM0j73TD4QM/OEdXEDdTn\nUgnK2altMfS6p0Y8z7vYpoJ0k/zz34FxNNY0pZI1pMgYEOOz2c+jFsNZkOvei33Rq7y6XsfObS3xLOx/zZ+lSIf5bFtAfPaPKvRkzGJVAoGBANPlaGwEAG+BSDqRZaI9\n63Oh3P4AAnM3tJFcMICHBYRnBUxvwT2+TS7BgccinqJJMR5o7Wx51K1q0GYyBd6l\n2uY9WESUnB/g2PlvPQauW15cZAdoA+miLCEP4QjNXl4TVObSNiMwwIDb3iR+iek4\nrpzMjxRZCxouP89ZiYTrVP6hAoGBALUQl3xfsMxyZtrX6irRXIFgyI814GOK8Af4\ngVB417nJZiczHIoXQiIXslKMT1Y5dmzXvuXa6GRiQyrCb1Vv0UpqmOMZPjXHyZ60\nHOSIOVlI9j+sED6mD2CdlBUzWoo1FvagHtbUKgfIBEzsEg26r3ByDcN1uYCflhNU\nH0YOEjCFAoGBAOUtlDMr6V455tpcTF03dQW4kZ07zY40TgghZwFjybYsM34KFC00\nO0Ot3Or50r4rYhZTw7snzTmP3ktGIdvZqFydwrwVhS8vXnx0/l/5Ka9gn95NAWFV\nKrwFtKrYCxQtNA+qlVODiLeMXgOQVj9HM7ad4favGxlyfZRV2FF4ROug\n-----END RSA PRIVATE KEY-----"
)

func getTLSAuthority(t *testing.T) (auth *TLSAuthority, key []byte, cert []byte) {
	pemkey, _, pemcert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	authority, err := LoadTLSAuthority(pemkey, pemcert)
	if err != nil {
		t.Fatalf("Could not create TLS authority: %s", err)
	}
	return authority.(*TLSAuthority), pemkey, pemcert
}

func TestLoadTLSAuthority(t *testing.T) {
	_, _, _ = getTLSAuthority(t)
}

func TestTLSEqual(t *testing.T) {
	a, pemkey, pemcert := getTLSAuthority(t)
	b, err := LoadTLSAuthority(pemkey, pemcert)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Equal(b.(*TLSAuthority)) {
		t.Errorf("Loaded authorities were mismatched")
	}
}

func TestGetTLSPublicKey(t *testing.T) {
	auth, _, cert := getTLSAuthority(t)
	pubkey := auth.GetPublicKey()
	if !bytes.Equal(pubkey, cert) {
		t.Errorf("Mismatch between pubkey and cert: %s versus %s", string(pubkey), string(cert))
	}
}

func TestTLSAuthority_ToCertPool(t *testing.T) {
	a, _, _ := getTLSAuthority(t)
	pool := a.ToCertPool()
	subjects := pool.Subjects()
	if len(subjects) != 1 {
		t.Error("Wrong number of subjects in cert pool")
	} else if !bytes.Equal(a.cert.RawSubject, subjects[0]) {
		t.Error("Mismatched subjects of certificates.")
	}
}

func TestParseTLSCertAsPrivkey(t *testing.T) {
	_, _, pemcert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	_, err := LoadTLSAuthority(pemcert, pemcert)
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Found PEM block of type") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
}

func TestParseTLSPrivkeyAsCert(t *testing.T) {
	pemkey, _, _ := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	_, err := LoadTLSAuthority(pemkey, pemkey)
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Found PEM block of type") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
}

func TestParseTLSMalformedLeadingData(t *testing.T) {
	pemkey, _, pemcert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	_, err := LoadTLSAuthority([]byte("JUNK\n"+string(pemkey)), pemcert)
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "PEM header") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
	_, err = LoadTLSAuthority(pemkey, []byte("JUNK\n"+string(pemcert)))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "PEM header") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
}

func TestParseTLSMalformedMiddleData(t *testing.T) {
	pemkey, _, pemcert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	skey := string(pemkey)
	_, err := LoadTLSAuthority([]byte(skey[:100]+"AAAA"+skey[104:]), pemcert)
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "not load PEM private key") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
	scert := string(pemcert)
	_, err = LoadTLSAuthority(pemkey, []byte(scert[:100]+"AAAA"+scert[104:]))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "structure error") {
		t.Errorf("Incorrect error, instead of structure error: %s", err)
	}
}

func TestParseTLSMalformedTruncatedData(t *testing.T) {
	pemkey, _, pemcert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	_, err := LoadTLSAuthority(pemkey, pemcert)
	if err != nil {
		t.Fatal(err)
	}
	_, err = LoadTLSAuthority(pemkey[:299], pemcert)
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Could not parse") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
	_, err = LoadTLSAuthority(pemkey, pemcert[:299])
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Could not parse") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
}

func TestParseTLSMalformedTrailingData(t *testing.T) {
	pemkey, _, pemcert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	_, err := LoadTLSAuthority([]byte(string(pemkey)+"\nJUNK"), pemcert)
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Trailing data") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
	_, err = LoadTLSAuthority(pemkey, []byte(string(pemcert)+"\nJUNK"))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Trailing data") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
}

func privateKeysEqual(k1 *rsa.PrivateKey, k2 *rsa.PrivateKey) bool {
	return k1.N.Cmp(k2.N) == 0 && k1.E == k2.E && k1.D.Cmp(k2.D) == 0
}

func TestLoadPKCS8Key(t *testing.T) {
	pemkey, pemkeypkcs8, pemcert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	authority1r, err := LoadTLSAuthority(pemkey, pemcert)
	if err != nil {
		t.Fatalf("Could not create TLS authority: %s", err)
	}
	authority2r, err := LoadTLSAuthority(pemkeypkcs8, pemcert)
	if err != nil {
		t.Fatalf("Could not create TLS authority: %s", err)
	}
	authority1 := authority1r.(*TLSAuthority)
	authority2 := authority2r.(*TLSAuthority)
	if !authority1.Equal(authority2) {
		t.Error("PKCS8 authority is different from PKCS1 authority")
	}
	if !privateKeysEqual(authority1.key, authority2.key) {
		t.Errorf("PKCS8 key is different from PKCS1 key: %v versus %v", authority1.key, authority2.key)
	}
}

func TestLoadECDSAKey(t *testing.T) {
	_, _, pemcert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	_, err := LoadTLSAuthority([]byte(TLS_TEST_ECDSA_KEY), pemcert)
	if err == nil {
		t.Error("Should not be able to create ECDSA authority (YET)")
	} else if !strings.Contains(err.Error(), "non-RSA") {
		t.Errorf("Expected ECDSA key to fail for not being RSA, not: %s", err)
	}
}

func TestLoadECDSACert(t *testing.T) {
	pemkey, _, _ := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	_, err := LoadTLSAuthority(pemkey, []byte(TLS_TEST_ECDSA_CERT))
	if err == nil {
		t.Error("Should not be able to create ECDSA authority (YET)")
	} else if !strings.Contains(err.Error(), "expected RSA") {
		t.Errorf("Expected ECDSA cert to fail for not being RSA, not: %s", err)
	}
}

func TestLoadTLSAuthority_WrongPrivKey(t *testing.T) {
	_, _, pemcert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	_, err := LoadTLSAuthority([]byte(TLS_TEST2_PRIVKEY), pemcert)
	if err == nil {
		t.Error("Should not be able to create mismatched authority")
	} else if !strings.Contains(err.Error(), "mismatched") {
		t.Errorf("Expected authority to fail for mismatch, not: %s", err)
	}
}

func TestTLSAuthority_ToHTTPSCert(t *testing.T) {
	auth, pemkey, pemcert := getTLSAuthority(t)
	cert := auth.ToHTTPSCert()
	baseline, err := tls.X509KeyPair(pemcert, pemkey)
	if err != nil {
		t.Fatal(err)
	}
	if len(cert.Certificate) != 1 || len(baseline.Certificate) != 1 {
		t.Error("Mismatched cert count")
	} else if !bytes.Equal(cert.Certificate[0], baseline.Certificate[0]) {
		t.Error("Mismatch between generated cert and baseline cert arrays")
	}
	certrsa := cert.PrivateKey.(*rsa.PrivateKey)
	baselinersa := baseline.PrivateKey.(*rsa.PrivateKey)
	if !privateKeysEqual(certrsa, baselinersa) {
		t.Error("Mismatched private keys between generated cert and baseline cert")
	}
}
