package wraputil

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"strings"
	"testing"
	"util/testutil"
)

const (
	TLS_TEST_CERT          = "-----BEGIN CERTIFICATE-----\nMIIC8TCCAdmgAwIBAgIJAPQ/RJLL04WxMA0GCSqGSIb3DQEBCwUAMA8xDTALBgNV\nBAMMBHRlc3QwHhcNMTcwODA2MDMzODM4WhcNMTcwOTA1MDMzODM4WjAPMQ0wCwYD\nVQQDDAR0ZXN0MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwB+3zCKU\nUkUtpeq8MBt6kmblE7CAUsBEtJGGg/esfg8w+hBIXp7DivudXgCcXxJJ3dVpJJKW\nRIlxWSBGyyNfSVig35cOeFl5n1b7K6RIfbtW5Mh2MqGMwkV9Of00sIZSGP3Ql1Ox\nfG1AFuWuIYVAXZ8R4y5sFpGqH4KHOrH/dc2FuaeQ1nyIEZzrfoChmdLycrqPavRT\nKnAHrB5sTtjvBCELzJd0PmVyTpTacnJHf2GpWU1iFWB0UYKxDNUWtYj8dBfbLR4e\n90ojzNCCkSYwMEGmQ41SgfbaUDotqq4DcD7qVqXPxmmsAilSvO+P/I6XThgOQ/9j\n3KF/SS/1AhsiLQIDAQABo1AwTjAdBgNVHQ4EFgQU4yZVYno76oZTa1u+55GL7boX\nQwEwHwYDVR0jBBgwFoAU4yZVYno76oZTa1u+55GL7boXQwEwDAYDVR0TBAUwAwEB\n/zANBgkqhkiG9w0BAQsFAAOCAQEAG0mGjtGFGdocvfqQtClRcYWaetMBR0e1NchS\nbaiy1it5BSdjrHTR6HoAGEJhPOKcJEphM7kjKRuXzhPYGqFzgzli8NY3w3qlV/ls\n/BAfFBsA2p5aRAYW1buLlj9EbV/BNyu3Rbs4EF6an0t/CWDlvw/VKFOPbic6j5sW\ne23rwwHkSQiIldQe29QeSNguqdNgGt2HSoc49TFLIbfsImfb1B6rz3ttsl+ownX4\nqPNt25Yzac9eELe5XfMRLEucwxmSoS47VrJaDHHZVTGYLNb0VEMwqSyKvJY8+3tC\neoMF8TPIXXgnhX2Xv2rnG1k0/h8w8uzQF+y7aOo/5nCyk903yw==\n-----END CERTIFICATE-----"
	TLS_TEST_PRIVKEY       = "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAwB+3zCKUUkUtpeq8MBt6kmblE7CAUsBEtJGGg/esfg8w+hBI\nXp7DivudXgCcXxJJ3dVpJJKWRIlxWSBGyyNfSVig35cOeFl5n1b7K6RIfbtW5Mh2\nMqGMwkV9Of00sIZSGP3Ql1OxfG1AFuWuIYVAXZ8R4y5sFpGqH4KHOrH/dc2FuaeQ\n1nyIEZzrfoChmdLycrqPavRTKnAHrB5sTtjvBCELzJd0PmVyTpTacnJHf2GpWU1i\nFWB0UYKxDNUWtYj8dBfbLR4e90ojzNCCkSYwMEGmQ41SgfbaUDotqq4DcD7qVqXP\nxmmsAilSvO+P/I6XThgOQ/9j3KF/SS/1AhsiLQIDAQABAoIBAGaTCKBWffYGtT52\nOw88PI7ZnMiMXZbQzF3TrIvcuh17otx/wQOzpBcaC4TasqIXs5Rako5SLSRedUPu\ndZ2TPxZ72ThHABTFQKgP2n4Mch+e29++H00c73fxfdBuHal5rW9mY+3HY2VZNvSC\noBuJdzoE6ZnveQn7r2avW9+8lPWhXOLXkX2FNnsWDyDPI2B2V5TrNEPvBtzgCC0B\nRS++mP90OfUx2lzKiQdLzJXvZz9/aDKlxQvBiniOoU1Y9wJvjcgRzGxR4Ie0g4pr\n0ATCZ4EsxGQ7ji0S97HQfKBpeaVI+4SwoxRKzT9CDySzKR1UPt2tCQx8cRnGsdUq\n+TkS78ECgYEA90UJbP0t43lHFmsr8XY3WPOBLQjwp/gXMx9Krl8ewTGiAVbzgSq+\nLhbB0MSZ/nZAb0QD9pLgxe5S2dDwXuf4LdCM4NXvpKIH6piewE1E9GScqdhUZ8//\nAZS1xIhW9oWDWFVcorzx/KT8S7aYLh1RZAwFPcJST0ScDBm1QjyCI0kCgYEAxug+\nDxd2RLgVm4w9Qngzqa+q3gVz5WgOqe6T9NTCu9UuAkx3GnajK64fxdf1P950tX7R\nhXyCeXIH8/GOncLV8gJLA7gizzEBduODnHqtJ1eL1eIsH/BtUA6OiIeKFs2YI3i7\nW2MBlmKrEi8bIEFTu7WG0VCaAJqZIGT0wwsLI8UCgYEA3OhqkVpngtA4uEirC5/n\ntqplf4x7JDU61Mth9wK4ATWMXNIH3iAHpDlkklTylymiS0VinQl/kpVmo35NIRzw\n1k15buzymgzAMdCEE510uzqf1AWW8uAaHJl1As4jkz6Yp3QrvKA9OM9VL3dD4f8D\nVfR/Qju3OWY8W3skOrbANTECgYBBSvE8MP5wtmDZa6KcVCrZU8HqGa4eqxbNL3TA\nFKtLz0HIHWOnezQ63XCumCJ4ccSr41JR2DpYNVdo+21OWiuywo/vS52Zl8OcTDji\nv95hILrVXeYQIfMwKWceaCerLpf3ZOVTrV9TB1aSpIXqA6fB4We9BBFZi2YinSE/\neTuR5QKBgQCW5lWO3l5+dshaGFr8STENKoLiTMA0bNc1JTWoUr0zc85f+LyuZfOV\nwP/ySBmN8Cj/8MqNQSIKY8ah7av/gzeFKjet1uIaj2+/sppAL1nIEOngFjT9LetT\nlZx0SIMHFp7t+2W5gxocd/kS97V+yNmkGH/JXH1yYQ2BWq8xtRPyhw==\n-----END RSA PRIVATE KEY-----"
	TLS_TEST_PKCS8_PRIVKEY = "-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDAH7fMIpRSRS2l\n6rwwG3qSZuUTsIBSwES0kYaD96x+DzD6EEhensOK+51eAJxfEknd1WkkkpZEiXFZ\nIEbLI19JWKDflw54WXmfVvsrpEh9u1bkyHYyoYzCRX05/TSwhlIY/dCXU7F8bUAW\n5a4hhUBdnxHjLmwWkaofgoc6sf91zYW5p5DWfIgRnOt+gKGZ0vJyuo9q9FMqcAes\nHmxO2O8EIQvMl3Q+ZXJOlNpyckd/YalZTWIVYHRRgrEM1Ra1iPx0F9stHh73SiPM\n0IKRJjAwQaZDjVKB9tpQOi2qrgNwPupWpc/GaawCKVK874/8jpdOGA5D/2PcoX9J\nL/UCGyItAgMBAAECggEAZpMIoFZ99ga1PnY7Dzw8jtmcyIxdltDMXdOsi9y6HXui\n3H/BA7OkFxoLhNqyohezlFqSjlItJF51Q+51nZM/FnvZOEcAFMVAqA/afgxyH57b\n374fTRzvd/F90G4dqXmtb2Zj7cdjZVk29IKgG4l3OgTpme95CfuvZq9b37yU9aFc\n4teRfYU2exYPIM8jYHZXlOs0Q+8G3OAILQFFL76Y/3Q59THaXMqJB0vMle9nP39o\nMqXFC8GKeI6hTVj3Am+NyBHMbFHgh7SDimvQBMJngSzEZDuOLRL3sdB8oGl5pUj7\nhLCjFErNP0IPJLMpHVQ+3a0JDHxxGcax1Sr5ORLvwQKBgQD3RQls/S3jeUcWayvx\ndjdY84EtCPCn+BczH0quXx7BMaIBVvOBKr4uFsHQxJn+dkBvRAP2kuDF7lLZ0PBe\n5/gt0Izg1e+kogfqmJ7ATUT0ZJyp2FRnz/8BlLXEiFb2hYNYVVyivPH8pPxLtpgu\nHVFkDAU9wlJPRJwMGbVCPIIjSQKBgQDG6D4PF3ZEuBWbjD1CeDOpr6reBXPlaA6p\n7pP01MK71S4CTHcadqMrrh/F1/U/3nS1ftGFfIJ5cgfz8Y6dwtXyAksDuCLPMQF2\n44Oceq0nV4vV4iwf8G1QDo6Ih4oWzZgjeLtbYwGWYqsSLxsgQVO7tYbRUJoAmpkg\nZPTDCwsjxQKBgQDc6GqRWmeC0Di4SKsLn+e2qmV/jHskNTrUy2H3ArgBNYxc0gfe\nIAekOWSSVPKXKaJLRWKdCX+SlWajfk0hHPDWTXlu7PKaDMAx0IQTnXS7Op/UBZby\n4BocmXUCziOTPpindCu8oD04z1Uvd0Ph/wNV9H9CO7c5ZjxbeyQ6tsA1MQKBgEFK\n8Tww/nC2YNlropxUKtlTweoZrh6rFs0vdMAUq0vPQcgdY6d7NDrdcK6YInhxxKvj\nUlHYOlg1V2j7bU5aK7LCj+9LnZmXw5xMOOK/3mEgutVd5hAh8zApZx5oJ6sul/dk\n5VOtX1MHVpKkheoDp8HhZ70EEVmLZiKdIT95O5HlAoGBAJbmVY7eXn52yFoYWvxJ\nMQ0qguJMwDRs1zUlNahSvTNzzl/4vK5l85XA//JIGY3wKP/wyo1BIgpjxqHtq/+D\nN4UqN63W4hqPb7+ymkAvWcgQ6eAWNP0t61OVnHRIgwcWnu37ZbmDGhx3+RL3tX7I\n2aQYf8lcfXJhDYFarzG1E/KH\n-----END PRIVATE KEY-----"
	TLS_TEST_ECDSA_KEY     = "-----BEGIN PRIVATE KEY-----\nMIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIArv14RWE1eJPwJMLY\nXbvXU5XufK9B/h1fp+23g2E/AqHqv8VNcQZy/a1IVFLrR8pxqE6pg4/vYiS+7BFJ\nwobGh/ahgYkDgYYABAFAIh6CwZTseH+WNsmHtl1d5BH29BjInF32fePbAPON1bXk\nTljhH2xwV5h/iJJf8zXHflxe9jIXRYF0ulL6vjV7TwHuO7MZkNi25oIwYj+rNj/j\nSWMHgdkSvSkWxfgogAWG0SxdiyUXOMeBlMZit1G3LijAubn9shTxPxNzLnHvbvVV\ngQ==\n-----END PRIVATE KEY-----"
	TLS_TEST_ECDSA_CERT    = "-----BEGIN CERTIFICATE-----\nMIIB7DCCAU6gAwIBAgIJAPrNxR4/otK9MAoGCCqGSM49BAMCMA8xDTALBgNVBAMM\nBHRlc3QwHhcNMTcwODA2MDQ0NzU2WhcNMTcwOTA1MDQ0NzU2WjAPMQ0wCwYDVQQD\nDAR0ZXN0MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQBQCIegsGU7Hh/ljbJh7Zd\nXeQR9vQYyJxd9n3j2wDzjdW15E5Y4R9scFeYf4iSX/M1x35cXvYyF0WBdLpS+r41\ne08B7juzGZDYtuaCMGI/qzY/40ljB4HZEr0pFsX4KIAFhtEsXYslFzjHgZTGYrdR\nty4owLm5/bIU8T8Tcy5x7271VYGjUDBOMB0GA1UdDgQWBBTnOngkY/2UtUdDgWUu\n/UQRBt2KbTAfBgNVHSMEGDAWgBTnOngkY/2UtUdDgWUu/UQRBt2KbTAMBgNVHRME\nBTADAQH/MAoGCCqGSM49BAMCA4GLADCBhwJCAOVYNUxX44Ex9+5nVTY8R5g0OQAW\nbDUTXqhgSNSr3bKybkmVhzbzKlZqLolloJMYvSw4nSGn3nF1WsDn/xyYLR0oAkEP\nraLAbWX42jMY9EC/l2++o0pLVkZs7tLok/w1cTaoSth2IGtPtMlL8N4prIWd84O7\nqrF9anZocHGTr9iD23E8fw==\n-----END CERTIFICATE-----"

	TLS_CLIENT_CSR = "-----BEGIN CERTIFICATE REQUEST-----\nMIIBVTCBvwIBADAWMRQwEgYDVQQDDAtjbGllbnQtdGVzdDCBnzANBgkqhkiG9w0B\nAQEFAAOBjQAwgYkCgYEAtKukT2LT/PJ/i1pbqfe4Vm9iN2yMFoiKj0em7FFOrAeU\n/5onq8fZEXhUruN+OhjMr+K1c2qy7noqbzD3Fz/vi2frB9DUFMA9rkj3teRIEXKB\nBDzb1cbDSTL0HxH47/tURxzxzGCVfTCc1xUY+dqMsd8SvowxuEptU4SO9H8CR2MC\nAwEAAaAAMA0GCSqGSIb3DQEBCwUAA4GBALCOKX+QHmNLGrrSCWB8p2iMuS+aPOcW\nYI9c1VaaTSQ43HOjF1smvGIa1iicM2L5zTBOEG36kI+sKFDOF2cXclhQF1WfLcxC\nIi/JSV+W7hbS6zWvJOnmoi15hzvVa1MRk8HZH+TpiMxO5uqQdDiEkV1sJ50v0ZtR\nTMuSBjdmmJ1t\n-----END CERTIFICATE REQUEST-----"
)

func TestLoadSinglePEMBlock_CompletelyWrong(t *testing.T) {
	_, err := LoadSinglePEMBlock([]byte("this isn't a pem block"), []string{"TEST"})
	testutil.CheckError(t, err, "Missing expected PEM header")
}

func TestLoadSinglePEMBlock_Malformed(t *testing.T) {
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN PEM BLOCK-----\nTHIS PEM BLOCK IS MALFORMED\n-----END PEM BLOCK-----\n"), []string{"TEST"})
	testutil.CheckError(t, err, "Could not parse PEM data")
}

func TestLoadSinglePEMBlock_MismatchEnds(t *testing.T) {
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST-----\nAAAA\n-----END PEM BLOCK-----\n"), []string{"TEST"})
	testutil.CheckError(t, err, "Could not parse PEM data")
}

func TestLoadSinglePEMBlock_TrailingData(t *testing.T) {
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST-----\nAAAA\n-----END TEST-----\nTrailing data\n"), []string{"TEST"})
	testutil.CheckError(t, err, "Trailing data")
}

func TestLoadSinglePEMBlock_TrailingNewlines(t *testing.T) {
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST-----\nAAAA\n-----END TEST-----\n\n"), []string{"TEST"})
	testutil.CheckError(t, err, "Trailing data")
}

func TestLoadSinglePEMBlock_EmptyTypes(t *testing.T) { // not really a supported configuration, but make sure it fails
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST-----\nAAAA\n-----END TEST-----\n"), nil)
	testutil.CheckError(t, err, "instead of types []")
}

func TestLoadSinglePEMBlock_EmptyTypes2(t *testing.T) { // not really a supported configuration, but make sure it fails
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN -----\nAAAA\n-----END -----\n"), nil)
	testutil.CheckError(t, err, "instead of types []")
}

func TestLoadSinglePEMBlock_FirstType(t *testing.T) { // not really a supported configuration, but make sure it fails
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST1-----\nAAAA\n-----END TEST1-----\n"), []string{"TEST1", "TEST2"})
	if err != nil {
		t.Error(err)
	}
}

func TestLoadSinglePEMBlock_SecondType(t *testing.T) { // not really a supported configuration, but make sure it fails
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST2-----\nAAAA\n-----END TEST2-----\n"), []string{"TEST1", "TEST2"})
	if err != nil {
		t.Error(err)
	}
}

func TestLoadSinglePEMBlock_NeitherType(t *testing.T) { // not really a supported configuration, but make sure it fails
	_, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST3-----\nAAAA\n-----END TEST3-----\n"), []string{"TEST1", "TEST2"})
	testutil.CheckError(t, err, "of type \"TEST3\" instead of types [TEST1 TEST2]")
}

func TestLoadSinglePEMBlock_CorrectResult(t *testing.T) { // not really a supported configuration, but make sure it fails
	out, err := LoadSinglePEMBlock([]byte("-----BEGIN TEST-----\nAAAA\n-----END TEST-----\n"), []string{"TEST"})
	if err != nil {
		t.Error(err)
	} else {
		if string(out) != "\x00\x00\x00" {
			t.Error("Wrong result.")
		}
	}
}

func TestLoadX509CertFromPEM(t *testing.T) {
	cert, err := LoadX509CertFromPEM([]byte(TLS_TEST_CERT))
	if err != nil {
		t.Fatal(err)
	}
	if cert.Subject.CommonName != "test" {
		t.Error("Wrong CN")
	}
	if cert.NotBefore.String() != "2017-08-06 03:38:38 +0000 UTC" {
		t.Error("Wrong start date")
	}
	if cert.NotAfter.String() != "2017-09-05 03:38:38 +0000 UTC" {
		t.Error("Wrong end date")
	}
}

func TestLoadX509CertFromPEM_Invalid(t *testing.T) {
	_, err := LoadX509CertFromPEM([]byte("invalid"))
	testutil.CheckError(t, err, "PEM header")
}

func TestLoadX509CertFromPEM_Empty(t *testing.T) {
	_, err := LoadX509CertFromPEM([]byte(""))
	testutil.CheckError(t, err, "PEM header")
}

func TestLoadX509CertFromPEM_Malformed(t *testing.T) {
	_, err := LoadX509CertFromPEM([]byte(strings.Replace(TLS_TEST_CERT, "AA", "ZZ", 1)))
	testutil.CheckError(t, err, "asn1: syntax error")
}

func TestLoadX509CertFromPEM_RawIsInput(t *testing.T) {
	cert, err := LoadX509CertFromPEM([]byte(TLS_TEST_CERT))
	if err != nil {
		t.Fatal(err)
	}
	data, err := LoadSinglePEMBlock([]byte(TLS_TEST_CERT), []string{"CERTIFICATE"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cert.Raw, data) {
		t.Error("Mismatch between raw bytes and expected raw bytes")
	}
}

func TestLoadX509CSRFromPEM(t *testing.T) {
	cert, err := LoadX509CSRFromPEM([]byte(TLS_CLIENT_CSR))
	if err != nil {
		t.Fatal(err)
	}
	if cert.Subject.CommonName != "client-test" {
		t.Error("Wrong CN")
	}
	if cert.SignatureAlgorithm != x509.SHA256WithRSA {
		t.Error("Wrong signature algorithm")
	}
}

func TestLoadX509CSRFromPEM_Invalid(t *testing.T) {
	_, err := LoadX509CSRFromPEM([]byte("invalid"))
	testutil.CheckError(t, err, "PEM header")
}

func TestLoadX509CSRFromPEM_Empty(t *testing.T) {
	_, err := LoadX509CSRFromPEM([]byte(""))
	testutil.CheckError(t, err, "PEM header")
}

func TestLoadX509CSRFromPEM_Malformed(t *testing.T) {
	_, err := LoadX509CSRFromPEM([]byte(strings.Replace(TLS_CLIENT_CSR, "AA", "ZZ", 1)))
	testutil.CheckError(t, err, "asn1: syntax error")
}

func TestLoadRSAKeyFromPEM_PKCS1(t *testing.T) {
	key, err := LoadRSAKeyFromPEM([]byte(TLS_TEST_PRIVKEY))
	if err != nil {
		t.Fatal(err)
	}
	err = key.Validate()
	if err != nil {
		t.Error(err)
	}
	cert, err := LoadX509CertFromPEM([]byte(TLS_TEST_CERT))
	if err != nil {
		t.Fatal(err)
	}
	if cert.PublicKey.(*rsa.PublicKey).N.Cmp(key.PublicKey.N) != 0 {
		t.Error("Key mismatch.")
	}
}

func TestLoadRSAKeyFromPEM_PKCS8(t *testing.T) {
	key, err := LoadRSAKeyFromPEM([]byte(TLS_TEST_PKCS8_PRIVKEY))
	if err != nil {
		t.Fatal(err)
	}
	err = key.Validate()
	if err != nil {
		t.Error(err)
	}
	cert, err := LoadX509CertFromPEM([]byte(TLS_TEST_CERT))
	if err != nil {
		t.Fatal(err)
	}
	if cert.PublicKey.(*rsa.PublicKey).N.Cmp(key.PublicKey.N) != 0 {
		t.Error("Key mismatch.")
	}
}

func TestLoadRSAKeyFromPEM_PKCS1_MATCHES_PKCS8(t *testing.T) {
	key1, err := LoadRSAKeyFromPEM([]byte(TLS_TEST_PRIVKEY))
	if err != nil {
		t.Fatal(err)
	}
	key8, err := LoadRSAKeyFromPEM([]byte(TLS_TEST_PKCS8_PRIVKEY))
	if err != nil {
		t.Fatal(err)
	}
	if key1.D.Cmp(key8.D) != 0 {
		t.Error("mismatched denom")
	}
	if key1.N.Cmp(key8.N) != 0 {
		t.Error("mismatched numer")
	}
	if len(key1.Primes) != len(key8.Primes) {
		t.Error("mismatch prime count")
	} else {
		for i, prime1 := range key1.Primes {
			prime8 := key8.Primes[i]
			if prime1.Cmp(prime8) != 0 {
				t.Error("mismatched prime")
			}
		}
	}
}

func TestLoadRSAKeyFromPEM_Fails_ECDSA(t *testing.T) {
	_, err := LoadRSAKeyFromPEM([]byte(TLS_TEST_ECDSA_KEY))
	testutil.CheckError(t, err, "non-RSA private key found")
}

func TestLoadRSAKeyFromPEM_Invalid(t *testing.T) {
	_, err := LoadRSAKeyFromPEM([]byte("invalid"))
	testutil.CheckError(t, err, "PEM header")
}

func TestLoadRSAKeyFromPEM_Empty(t *testing.T) {
	_, err := LoadRSAKeyFromPEM([]byte(""))
	testutil.CheckError(t, err, "PEM header")
}

func TestLoadRSAKeyFromPEM_Malformed(t *testing.T) {
	_, err := LoadRSAKeyFromPEM([]byte(strings.Replace(TLS_TEST_PRIVKEY, "AA", "ZZ", 1)))
	testutil.CheckError(t, err, "could not load PEM private key as PKCS#1 or PKCS#8")
}
