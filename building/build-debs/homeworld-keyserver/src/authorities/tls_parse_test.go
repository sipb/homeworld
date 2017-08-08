package authorities

import (
	"testing"
	"bytes"
	"strings"
	"crypto/tls"
	"crypto/rsa"
)

const (
	TLS_TEST_CERT          = "-----BEGIN CERTIFICATE-----\nMIIC8TCCAdmgAwIBAgIJAPQ/RJLL04WxMA0GCSqGSIb3DQEBCwUAMA8xDTALBgNV\nBAMMBHRlc3QwHhcNMTcwODA2MDMzODM4WhcNMTcwOTA1MDMzODM4WjAPMQ0wCwYD\nVQQDDAR0ZXN0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwB+3zCKU\nUkUtpeq8MBt6kmblE7CAUsBEtJGGg/esfg8w+hBIXp7DivudXgCcXxJJ3dVpJJKW\nRIlxWSBGyyNfSVig35cOeFl5n1b7K6RIfbtW5Mh2MqGMwkV9Of00sIZSGP3Ql1Ox\nfG1AFuWuIYVAXZ8R4y5sFpGqH4KHOrH/dc2FuaeQ1nyIEZzrfoChmdLycrqPavRT\nKnAHrB5sTtjvBCELzJd0PmVyTpTacnJHf2GpWU1iFWB0UYKxDNUWtYj8dBfbLR4e\n90ojzNCCkSYwMEGmQ41SgfbaUDotqq4DcD7qVqXPxmmsAilSvO+P/I6XThgOQ/9j\n3KF/SS/1AhsiLQIDAQABo1AwTjAdBgNVHQ4EFgQU4yZVYno76oZTa1u+55GL7boX\nQwEwHwYDVR0jBBgwFoAU4yZVYno76oZTa1u+55GL7boXQwEwDAYDVR0TBAUwAwEB\n/zANBgkqhkiG9w0BAQsFAAOCAQEAG0mGjtGFGdocvfqQtClRcYWaetMBR0e1NchS\nbaiy1it5BSdjrHTR6HoAGEJhPOKcJEphM7kjKRuXzhPYGqFzgzli8NY3w3qlV/ls\n/BAfFBsA2p5aRAYW1buLlj9EbV/BNyu3Rbs4EF6an0t/CWDlvw/VKFOPbic6j5sW\ne23rwwHkSQiIldQe29QeSNguqdNgGt2HSoc49TFLIbfsImfb1B6rz3ttsl+ownX4\nqPNt25Yzac9eELe5XfMRLEucwxmSoS47VrJaDHHZVTGYLNb0VEMwqSyKvJY8+3tC\neoMF8TPIXXgnhX2Xv2rnG1k0/h8w8uzQF+y7aOo/5nCyk903yw==\n-----END CERTIFICATE-----"
	TLS_TEST_PRIVKEY       = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAwB+3zCKUUkUtpeq8MBt6kmblE7CAUsBEtJGGg/esfg8w+hBI\nXp7DivudXgCcXxJJ3dVpJJKWRIlxWSBGyyNfSVig35cOeFl5n1b7K6RIfbtW5Mh2\nMqGMwkV9Of00sIZSGP3Ql1OxfG1AFuWuIYVAXZ8R4y5sFpGqH4KHOrH/dc2FuaeQ\n1nyIEZzrfoChmdLycrqPavRTKnAHrB5sTtjvBCELzJd0PmVyTpTacnJHf2GpWU1i\nFWB0UYKxDNUWtYj8dBfbLR4e90ojzNCCkSYwMEGmQ41SgfbaUDotqq4DcD7qVqXP\nxmmsAilSvO+P/I6XThgOQ/9j3KF/SS/1AhsiLQIDAQABAoIBAGaTCKBWffYGtT52\nOw88PI7ZnMiMXZbQzF3TrIvcuh17otx/wQOzpBcaC4TasqIXs5Rako5SLSRedUPu\ndZ2TPxZ72ThHABTFQKgP2n4Mch+e29++H00c73fxfdBuHal5rW9mY+3HY2VZNvSC\noBuJdzoE6ZnveQn7r2avW9+8lPWhXOLXkX2FNnsWDyDPI2B2V5TrNEPvBtzgCC0B\nRS++mP90OfUx2lzKiQdLzJXvZz9/aDKlxQvBiniOoU1Y9wJvjcgRzGxR4Ie0g4pr\n0ATCZ4EsxGQ7ji0S97HQfKBpeaVI+4SwoxRKzT9CDySzKR1UPt2tCQx8cRnGsdUq\n+TkS78ECgYEA90UJbP0t43lHFmsr8XY3WPOBLQjwp/gXMx9Krl8ewTGiAVbzgSq+\nLhbB0MSZ/nZAb0QD9pLgxe5S2dDwXuf4LdCM4NXvpKIH6piewE1E9GScqdhUZ8//\nAZS1xIhW9oWDWFVcorzx/KT8S7aYLh1RZAwFPcJST0ScDBm1QjyCI0kCgYEAxug+\nDxd2RLgVm4w9Qngzqa+q3gVz5WgOqe6T9NTCu9UuAkx3GnajK64fxdf1P950tX7R\nhXyCeXIH8/GOncLV8gJLA7gizzEBduODnHqtJ1eL1eIsH/BtUA6OiIeKFs2YI3i7\nW2MBlmKrEi8bIEFTu7WG0VCaAJqZIGT0wwsLI8UCgYEA3OhqkVpngtA4uEirC5/n\ntqplf4x7JDU61Mth9wK4ATWMXNIH3iAHpDlkklTylymiS0VinQl/kpVmo35NIRzw\n1k15buzymgzAMdCEE510uzqf1AWW8uAaHJl1As4jkz6Yp3QrvKA9OM9VL3dD4f8D\nVfR/Qju3OWY8W3skOrbANTECgYBBSvE8MP5wtmDZa6KcVCrZU8HqGa4eqxbNL3TA\nFKtLz0HIHWOnezQ63XCumCJ4ccSr41JR2DpYNVdo+21OWiuywo/vS52Zl8OcTDji\nv95hILrVXeYQIfMwKWceaCerLpf3ZOVTrV9TB1aSpIXqA6fB4We9BBFZi2YinSE/\neTuR5QKBgQCW5lWO3l5+dshaGFr8STENKoLiTMA0bNc1JTWoUr0zc85f+LyuZfOV\nwP/ySBmN8Cj/8MqNQSIKY8ah7av/gzeFKjet1uIaj2+/sppAL1nIEOngFjT9LetT\nlZx0SIMHFp7t+2W5gxocd/kS97V+yNmkGH/JXH1yYQ2BWq8xtRPyhw==\n-----END RSA PRIVATE KEY-----"
	TLS_TEST_PKCS8_PRIVKEY = "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDAH7fMIpRSRS2l\n6rwwG3qSZuUTsIBSwES0kYaD96x+DzD6EEhensOK+51eAJxfEknd1WkkkpZEiXFZ\nIEbLI19JWKDflw54WXmfVvsrpEh9u1bkyHYyoYzCRX05/TSwhlIY/dCXU7F8bUAW\n5a4hhUBdnxHjLmwWkaofgoc6sf91zYW5p5DWfIgRnOt+gKGZ0vJyuo9q9FMqcAes\nHmxO2O8EIQvMl3Q+ZXJOlNpyckd/YalZTWIVYHRRgrEM1Ra1iPx0F9stHh73SiPM\n0IKRJjAwQaZDjVKB9tpQOi2qrgNwPupWpc/GaawCKVK874/8jpdOGA5D/2PcoX9J\nL/UCGyItAgMBAAECggEAZpMIoFZ99ga1PnY7Dzw8jtmcyIxdltDMXdOsi9y6HXui\n3H/BA7OkFxoLhNqyohezlFqSjlItJF51Q+51nZM/FnvZOEcAFMVAqA/afgxyH57b\n374fTRzvd/F90G4dqXmtb2Zj7cdjZVk29IKgG4l3OgTpme95CfuvZq9b37yU9aFc\n4teRfYU2exYPIM8jYHZXlOs0Q+8G3OAILQFFL76Y/3Q59THaXMqJB0vMle9nP39o\nMqXFC8GKeI6hTVj3Am+NyBHMbFHgh7SDimvQBMJngSzEZDuOLRL3sdB8oGl5pUj7\nhLCjFErNP0IPJLMpHVQ+3a0JDHxxGcax1Sr5ORLvwQKBgQD3RQls/S3jeUcWayvx\ndjdY84EtCPCn+BczH0quXx7BMaIBVvOBKr4uFsHQxJn+dkBvRAP2kuDF7lLZ0PBe\n5/gt0Izg1e+kogfqmJ7ATUT0ZJyp2FRnz/8BlLXEiFb2hYNYVVyivPH8pPxLtpgu\nHVFkDAU9wlJPRJwMGbVCPIIjSQKBgQDG6D4PF3ZEuBWbjD1CeDOpr6reBXPlaA6p\n7pP01MK71S4CTHcadqMrrh/F1/U/3nS1ftGFfIJ5cgfz8Y6dwtXyAksDuCLPMQF2\n44Oceq0nV4vV4iwf8G1QDo6Ih4oWzZgjeLtbYwGWYqsSLxsgQVO7tYbRUJoAmpkg\nZPTDCwsjxQKBgQDc6GqRWmeC0Di4SKsLn+e2qmV/jHskNTrUy2H3ArgBNYxc0gfe\nIAekOWSSVPKXKaJLRWKdCX+SlWajfk0hHPDWTXlu7PKaDMAx0IQTnXS7Op/UBZby\n4BocmXUCziOTPpindCu8oD04z1Uvd0Ph/wNV9H9CO7c5ZjxbeyQ6tsA1MQKBgEFK\n8Tww/nC2YNlropxUKtlTweoZrh6rFs0vdMAUq0vPQcgdY6d7NDrdcK6YInhxxKvj\nUlHYOlg1V2j7bU5aK7LCj+9LnZmXw5xMOOK/3mEgutVd5hAh8zApZx5oJ6sul/dk\n5VOtX1MHVpKkheoDp8HhZ70EEVmLZiKdIT95O5HlAoGBAJbmVY7eXn52yFoYWvxJ\nMQ0qguJMwDRs1zUlNahSvTNzzl/4vK5l85XA//JIGY3wKP/wyo1BIgpjxqHtq/+D\nN4UqN63W4hqPb7+ymkAvWcgQ6eAWNP0t61OVnHRIgwcWnu37ZbmDGhx3+RL3tX7I\n2aQYf8lcfXJhDYFarzG1E/KH\n-----END PRIVATE KEY-----"
	TLS_TEST_ECDSA_KEY     = "-----BEGIN PRIVATE KEY-----\nMIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIArv14RWE1eJPwJMLY\nXbvXU5XufK9B/h1fp+23g2E/AqHqv8VNcQZy/a1IVFLrR8pxqE6pg4/vYiS+7BFJ\nwobGh/ahgYkDgYYABAFAIh6CwZTseH+WNsmHtl1d5BH29BjInF32fePbAPON1bXk\nTljhH2xwV5h/iJJf8zXHflxe9jIXRYF0ulL6vjV7TwHuO7MZkNi25oIwYj+rNj/j\nSWMHgdkSvSkWxfgogAWG0SxdiyUXOMeBlMZit1G3LijAubn9shTxPxNzLnHvbvVV\ngQ==\n-----END PRIVATE KEY-----"
	TLS_TEST_ECDSA_CERT    = "-----BEGIN CERTIFICATE-----\nMIIB7DCCAU6gAwIBAgIJAPrNxR4/otK9MAoGCCqGSM49BAMCMA8xDTALBgNVBAMM\nBHRlc3QwHhcNMTcwODA2MDQ0NzU2WhcNMTcwOTA1MDQ0NzU2WjAPMQ0wCwYDVQQD\nDAR0ZXN0MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQBQCIegsGU7Hh/ljbJh7Zd\nXeQR9vQYyJxd9n3j2wDzjdW15E5Y4R9scFeYf4iSX/M1x35cXvYyF0WBdLpS+r41\ne08B7juzGZDYtuaCMGI/qzY/40ljB4HZEr0pFsX4KIAFhtEsXYslFzjHgZTGYrdR\nty4owLm5/bIU8T8Tcy5x7271VYGjUDBOMB0GA1UdDgQWBBTnOngkY/2UtUdDgWUu\n/UQRBt2KbTAfBgNVHSMEGDAWgBTnOngkY/2UtUdDgWUu/UQRBt2KbTAMBgNVHRME\nBTADAQH/MAoGCCqGSM49BAMCA4GLADCBhwJCAOVYNUxX44Ex9+5nVTY8R5g0OQAW\nbDUTXqhgSNSr3bKybkmVhzbzKlZqLolloJMYvSw4nSGn3nF1WsDn/xyYLR0oAkEP\nraLAbWX42jMY9EC/l2++o0pLVkZs7tLok/w1cTaoSth2IGtPtMlL8N4prIWd84O7\nqrF9anZocHGTr9iD23E8fw==\n-----END CERTIFICATE-----"

	TLS_TEST2_PRIVKEY = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpgIBAAKCAQEAuc1psL/DYin1ZLfSh2erwFg5yqJ1BlW4xPkRA4T2cFOKG8zB\nI3MfB2vJkB2Ug821vN+QxfX77CQwEQZwiIf9gAdYlC9I3TNnESipqncxq2x0Jnre\nBYCaFgru60CoLeCUwMUFXNKhkgagZSp9kAirR1LPTyThAx0s+JeC8ImfxNscQ/uf\nzOxCnZXEV8gTmROqAMCHVfSTYy9BYqjf0D1oo2XBTdAFk6pdDUcTKvRGoDs/V4Ry\n13X96z0MyfyD/8yumQegwEOd0Ay3yfOzwHgv2JhVrAduZRxjlel+W3Mk9Bw8Xy9W\n2DQgdKDQONe4g6efsqmKBXQ1tSKI+T9kCW2JtQIDAQABAoIBAQCtU/uhn/KT05KR\nZ45lNIgbgfI/nzfONg+M6NA/WT1QYg43itYtzMoIcTvyTjXqku9UB7cVhTiC/Os+\nJqS6KSqJ0dCHRGkTuU0Py8AjPtg+E4lzEDGoLmUP5RkmqwV47sW14tXy1qdVAwuD\n9JR31i559b1hFoU2E3SNX0IORESgLTLAujAnrCWvnal/O6d1Tzb9csVxj/8e2Ymw\nuONAnu6jOKyOxGt+VJAoMrZkutzhZixmdFfQyuYYV3ViLhKcgtpWOXjHvCJfJ/7o\nBCy74mHkti+i24Ssd5sgdiWOuoURjJgm3okQtTGSR4guPWkYAajLzeQYD9aSepzR\nHSIIRXWBAoGBAOxjHLpEHMIq9O0vshDcu2dq5pTEOBarVXr2sh5WBUUwZjqCuop5\n3KV/olSRENipnYYOIHiDMDSjcB05cFApmKmi7CIQOOnNSgZjpzdsKXQKEWLfEO2S\nYJ+lybcSYaKxONIAPU9PISxZgI6jYSc3nb/uEDStERm4cnB6MV2cXanhAoGBAMk3\n4ACoACgov/9wY0OHyXRfvEflo9AJvUHXgRvopX06lMSDM0j73TD4QM/OEdXEDdTn\nUgnK2altMfS6p0Y8z7vYpoJ0k/zz34FxNNY0pZI1pMgYEOOz2c+jFsNZkOvei33Rq7y6XsfObS3xLOx/zZ+lSIf5bFtAfPaPKvRkzGJVAoGBANPlaGwEAG+BSDqRZaI9\n63Oh3P4AAnM3tJFcMICHBYRnBUxvwT2+TS7BgccinqJJMR5o7Wx51K1q0GYyBd6l\n2uY9WESUnB/g2PlvPQauW15cZAdoA+miLCEP4QjNXl4TVObSNiMwwIDb3iR+iek4\nrpzMjxRZCxouP89ZiYTrVP6hAoGBALUQl3xfsMxyZtrX6irRXIFgyI814GOK8Af4\ngVB417nJZiczHIoXQiIXslKMT1Y5dmzXvuXa6GRiQyrCb1Vv0UpqmOMZPjXHyZ60\nHOSIOVlI9j+sED6mD2CdlBUzWoo1FvagHtbUKgfIBEzsEg26r3ByDcN1uYCflhNU\nH0YOEjCFAoGBAOUtlDMr6V455tpcTF03dQW4kZ07zY40TgghZwFjybYsM34KFC00\nO0Ot3Or50r4rYhZTw7snzTmP3ktGIdvZqFydwrwVhS8vXnx0/l/5Ka9gn95NAWFV\nKrwFtKrYCxQtNA+qlVODiLeMXgOQVj9HM7ad4favGxlyfZRV2FF4ROug\n-----END RSA PRIVATE KEY-----"
)

func getTLSAuthority(t *testing.T) *TLSAuthority {
	authority, err := LoadTLSAuthority([]byte(TLS_TEST_PRIVKEY), []byte(TLS_TEST_CERT))
	if err != nil {
		t.Fatalf("Could not create TLS authority: %s", err)
	}
	return authority.(*TLSAuthority)
}

func TestLoadTLSAuthority(t *testing.T) {
	_ = getTLSAuthority(t)
}

func TestTLSEqual(t *testing.T) {
	a := getTLSAuthority(t)
	b := getTLSAuthority(t)
	if !a.Equal(b) {
		t.Errorf("Loaded authorities were mismatched")
	}
}

func TestGetTLSPublicKey(t *testing.T) {
	pubkey := getTLSAuthority(t).GetPublicKey()
	if !bytes.Equal(pubkey, []byte(TLS_TEST_CERT)) {
		t.Error("Mismatch between pubkey and cert: %s versus %s", string(pubkey), TLS_TEST_CERT)
	}
}

func TestTLSAuthority_ToCertPool(t *testing.T) {
	a := getTLSAuthority(t)
	pool := a.ToCertPool()
	subjects := pool.Subjects()
	if len(subjects) != 1 {
		t.Error("Wrong number of subjects in cert pool")
	} else if !bytes.Equal(a.cert.RawSubject, subjects[0]) {
		t.Error("Mismatched subjects of certificates.")
	}
}

func TestParseTLSCertAsPrivkey(t *testing.T) {
	_, err := LoadTLSAuthority([]byte(TLS_TEST_CERT), []byte(TLS_TEST_CERT))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Found PEM block of type") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
}

func TestParseTLSPrivkeyAsCert(t *testing.T) {
	_, err := LoadTLSAuthority([]byte(TLS_TEST_PRIVKEY), []byte(TLS_TEST_PRIVKEY))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Found PEM block of type") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
}

func TestParseTLSMalformedLeadingData(t *testing.T) {
	_, err := LoadTLSAuthority([]byte("JUNK\n"+TLS_TEST_PRIVKEY), []byte(TLS_TEST_CERT))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "PEM header") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
	_, err = LoadTLSAuthority([]byte(TLS_TEST_PRIVKEY), []byte("JUNK\n"+TLS_TEST_CERT))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "PEM header") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
}

func TestParseTLSMalformedMiddleData(t *testing.T) {
	_, err := LoadTLSAuthority([]byte(strings.Replace(TLS_TEST_PRIVKEY, "Z", "Y", 1)), []byte(TLS_TEST_CERT))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "not load PEM private key") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
	_, err = LoadTLSAuthority([]byte(TLS_TEST_PRIVKEY), []byte(strings.Replace(TLS_TEST_CERT, "K", "Y", -1)))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "don't match") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
}

func TestParseTLSMalformedTruncatedData(t *testing.T) {
	_, err := LoadTLSAuthority([]byte(TLS_TEST_PRIVKEY[:1000]), []byte(TLS_TEST_CERT))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Could not parse") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
	_, err = LoadTLSAuthority([]byte(TLS_TEST_PRIVKEY), []byte(TLS_TEST_CERT[:1000]))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Could not parse") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
}

func TestParseTLSMalformedTrailingData(t *testing.T) {
	_, err := LoadTLSAuthority([]byte(TLS_TEST_PRIVKEY+"\nJUNK"), []byte(TLS_TEST_CERT))
	if err == nil {
		t.Errorf("Expected creation of TLS authority to be broken")
	} else if !strings.Contains(err.Error(), "Trailing data") {
		t.Errorf("Incorrect error, instead of PEM error: %s", err)
	}
	_, err = LoadTLSAuthority([]byte(TLS_TEST_PRIVKEY), []byte(TLS_TEST_CERT+"\nJUNK"))
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
	authority1r, err := LoadTLSAuthority([]byte(TLS_TEST_PRIVKEY), []byte(TLS_TEST_CERT))
	if err != nil {
		t.Fatalf("Could not create TLS authority: %s", err)
	}
	authority2r, err := LoadTLSAuthority([]byte(TLS_TEST_PKCS8_PRIVKEY), []byte(TLS_TEST_CERT))
	if err != nil {
		t.Fatalf("Could not create TLS authority: %s", err)
	}
	authority1 := authority1r.(*TLSAuthority)
	authority2 := authority2r.(*TLSAuthority)
	if !authority1.Equal(authority2) {
		t.Error("PKCS8 authority is different from PKCS1 authority")
	}
	if !privateKeysEqual(authority1.key, authority2.key) {
		t.Error("PKCS8 key is different from PKCS1 key: %v versus %v", authority1.key, authority2.key)
	}
}

func TestLoadECDSAKey(t *testing.T) {
	_, err := LoadTLSAuthority([]byte(TLS_TEST_ECDSA_KEY), []byte(TLS_TEST_CERT))
	if err == nil {
		t.Error("Should not be able to create ECDSA authority (YET)")
	} else if !strings.Contains(err.Error(), "Non-RSA") {
		t.Errorf("Expected ECDSA key to fail for not being RSA, not: %s", err)
	}
}

func TestLoadECDSACert(t *testing.T) {
	_, err := LoadTLSAuthority([]byte(TLS_TEST_PRIVKEY), []byte(TLS_TEST_ECDSA_CERT))
	if err == nil {
		t.Error("Should not be able to create ECDSA authority (YET)")
	} else if !strings.Contains(err.Error(), "expected RSA") {
		t.Errorf("Expected ECDSA cert to fail for not being RSA, not: %s", err)
	}
}

func TestLoadTLSAuthority_WrongPrivKey(t *testing.T) {
	_, err := LoadTLSAuthority([]byte(TLS_TEST2_PRIVKEY), []byte(TLS_TEST_CERT))
	if err == nil {
		t.Error("Should not be able to create mismatched authority")
	} else if !strings.Contains(err.Error(), "mismatched") {
		t.Errorf("Expected authority to fail for mismatch, not: %s", err)
	}
}

func TestTLSAuthority_ToHTTPSCert(t *testing.T) {
	cert := getTLSAuthority(t).ToHTTPSCert()
	baseline, err := tls.X509KeyPair([]byte(TLS_TEST_CERT), []byte(TLS_TEST_PRIVKEY))
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
