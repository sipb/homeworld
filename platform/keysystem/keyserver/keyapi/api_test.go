package keyapi

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/config"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/verifier"
	"github.com/sipb/homeworld/platform/util/testkeyutil"
	"github.com/sipb/homeworld/platform/util/wraputil"
)

func TestVerifyAccountIP_NoLimit(t *testing.T) {
	acnt := &account.Account{
		LimitIP: nil,
	}

	request := httptest.NewRequest("GET", "/test", nil)
	request.RemoteAddr = "192.168.0.1:1234"

	err := verifyAccountIP(acnt, request)
	if err != nil {
		t.Error(err)
	}
}

func TestVerifyAccountIP_Valid(t *testing.T) {
	acnt := &account.Account{
		LimitIP: net.IPv4(192, 168, 0, 3),
	}

	request := httptest.NewRequest("GET", "/test", nil)
	request.RemoteAddr = "192.168.0.3:1234"

	err := verifyAccountIP(acnt, request)
	if err != nil {
		t.Error(err)
	}
}

func TestVerifyAccountIP_Invalid(t *testing.T) {
	acnt := &account.Account{
		LimitIP: net.IPv4(192, 168, 0, 3),
	}

	request := httptest.NewRequest("GET", "/test", nil)
	request.RemoteAddr = "172.16.0.1:1234"

	err := verifyAccountIP(acnt, request)
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "wrong IP address") {
		t.Error("Wrong error.")
	}
}

func TestVerifyAccountIP_BadRequest(t *testing.T) {
	acnt := &account.Account{
		LimitIP: net.IPv4(192, 168, 0, 3),
	}

	request := httptest.NewRequest("GET", "/test", nil)
	request.RemoteAddr = "invalid"

	err := verifyAccountIP(acnt, request)
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "invalid request address") {
		t.Errorf("Wrong error: %s", err)
	}
}

func TestAttemptAuthentication_NoAuthentication(t *testing.T) {
	request := httptest.NewRequest("GET", "/test", nil)
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		TokenVerifier:           verifier.NewTokenVerifier(),
		AuthenticationAuthority: authority.(*authorities.TLSAuthority),
	}
	_, err = attemptAuthentication(&gctx, request)
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "No authentication method") {
		t.Error("Wrong error.")
	}
}

func TestAttemptAuthentication_TokenAuth(t *testing.T) {
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		TokenVerifier:           verifier.NewTokenVerifier(),
		AuthenticationAuthority: authority.(*authorities.TLSAuthority),
		Accounts: map[string]*account.Account{
			"test-user": {Principal: "test-user"},
		},
	}
	request := httptest.NewRequest("GET", "/test", nil)
	request.Header.Set(verifier.TokenHeader, gctx.TokenVerifier.Registry.GrantToken("test-user", time.Minute))
	acnt, err := attemptAuthentication(&gctx, request)
	if err != nil {
		t.Error(err)
	} else if acnt.Principal != "test-user" {
		t.Error("Wrong account.")
	}
}

func TestAttemptAuthentication_TokenAuth_Fail(t *testing.T) {
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		TokenVerifier:           verifier.NewTokenVerifier(),
		AuthenticationAuthority: authority.(*authorities.TLSAuthority),
		Accounts: map[string]*account.Account{
			"test-user": {Principal: "test-user"},
		},
	}
	request := httptest.NewRequest("GET", "/test", nil)
	request.Header.Set(verifier.TokenHeader, "this-is-not-a-valid-token")
	_, err = attemptAuthentication(&gctx, request)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "Unrecognized token") {
		t.Errorf("Wrong error: %s", err.Error())
	}
}

func TestAttemptAuthentication_MissingAccount_Token(t *testing.T) {
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		TokenVerifier:           verifier.NewTokenVerifier(),
		AuthenticationAuthority: authority.(*authorities.TLSAuthority),
		Accounts:                map[string]*account.Account{},
	}
	request := httptest.NewRequest("GET", "/test", nil)
	request.Header.Set(verifier.TokenHeader, gctx.TokenVerifier.Registry.GrantToken("test-user", time.Minute))
	_, err = attemptAuthentication(&gctx, request)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "Cannot find account") {
		t.Errorf("Wrong error: %s", err)
	}
}

func TestAttemptAuthentication_NoDirectAuth_Token(t *testing.T) {
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		TokenVerifier:           verifier.NewTokenVerifier(),
		AuthenticationAuthority: authority.(*authorities.TLSAuthority),
		Accounts: map[string]*account.Account{
			"test-user": {Principal: "test-user", DisableDirectAuth: true},
		},
	}
	request := httptest.NewRequest("GET", "/test", nil)
	request.Header.Set(verifier.TokenHeader, gctx.TokenVerifier.Registry.GrantToken("test-user", time.Minute))
	_, err = attemptAuthentication(&gctx, request)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "disabled direct authentication") {
		t.Errorf("Wrong error: %s", err)
	}
}

func TestAttemptAuthentication_InvalidIP_Token(t *testing.T) {
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		TokenVerifier:           verifier.NewTokenVerifier(),
		AuthenticationAuthority: authority.(*authorities.TLSAuthority),
		Accounts: map[string]*account.Account{
			"test-user": {Principal: "test-user", LimitIP: net.IPv4(192, 168, 0, 16)},
		},
	}
	request := httptest.NewRequest("GET", "/test", nil)
	request.Header.Set(verifier.TokenHeader, gctx.TokenVerifier.Registry.GrantToken("test-user", time.Minute))
	request.RemoteAddr = "192.168.0.17:50000"
	_, err = attemptAuthentication(&gctx, request)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "from wrong IP address") {
		t.Errorf("Wrong error: %s", err)
	}
}

const (
	TLS_CLIENT_CSR = "-----BEGIN CERTIFICATE REQUEST-----\nMIIBVTCBvwIBADAWMRQwEgYDVQQDDAtjbGllbnQtdGVzdDCBnzANBgkqhkiG9w0B\nAQEFAAOBjQAwgYkCgYEAtKukT2LT/PJ/i1pbqfe4Vm9iN2yMFoiKj0em7FFOrAeU\n/5onq8fZEXhUruN+OhjMr+K1c2qy7noqbzD3Fz/vi2frB9DUFMA9rkj3teRIEXKB\nBDzb1cbDSTL0HxH47/tURxzxzGCVfTCc1xUY+dqMsd8SvowxuEptU4SO9H8CR2MC\nAwEAAaAAMA0GCSqGSIb3DQEBCwUAA4GBALCOKX+QHmNLGrrSCWB8p2iMuS+aPOcW\nYI9c1VaaTSQ43HOjF1smvGIa1iicM2L5zTBOEG36kI+sKFDOF2cXclhQF1WfLcxC\nIi/JSV+W7hbS6zWvJOnmoi15hzvVa1MRk8HZH+TpiMxO5uqQdDiEkV1sJ50v0ZtR\nTMuSBjdmmJ1t\n-----END CERTIFICATE REQUEST-----"
)

func prepCertAuth(t *testing.T, gctx *config.Context) *http.Request {
	certstr, err := gctx.AuthenticationAuthority.Sign(TLS_CLIENT_CSR, false, time.Minute, "test-user", []string{})
	if err != nil {
		t.Fatal(err)
	}
	cert, err := wraputil.LoadX509CertFromPEM([]byte(certstr))
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/test", nil)
	request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}
	return request
}

func TestAttemptAuthentication_CertAuth(t *testing.T) {
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		TokenVerifier:           verifier.NewTokenVerifier(),
		AuthenticationAuthority: authority.(*authorities.TLSAuthority),
		Accounts: map[string]*account.Account{
			"test-user": {Principal: "test-user"},
		},
	}
	acnt, err := attemptAuthentication(&gctx, prepCertAuth(t, &gctx))
	if err != nil {
		t.Error(err)
	} else if acnt.Principal != "test-user" {
		t.Error("Wrong account.")
	}
}

func TestAttemptAuthentication_MissingAccount_Cert(t *testing.T) {
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		TokenVerifier:           verifier.NewTokenVerifier(),
		AuthenticationAuthority: authority.(*authorities.TLSAuthority),
		Accounts:                map[string]*account.Account{},
	}
	_, err = attemptAuthentication(&gctx, prepCertAuth(t, &gctx))
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "Cannot find account") {
		t.Errorf("Wrong error: %s", err)
	}
}

func TestAttemptAuthentication_NoDirectAuth_Cert(t *testing.T) {
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		TokenVerifier:           verifier.NewTokenVerifier(),
		AuthenticationAuthority: authority.(*authorities.TLSAuthority),
		Accounts: map[string]*account.Account{
			"test-user": {Principal: "test-user", DisableDirectAuth: true},
		},
	}
	_, err = attemptAuthentication(&gctx, prepCertAuth(t, &gctx))
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "disabled direct authentication") {
		t.Errorf("Wrong error: %s", err)
	}
}

func TestAttemptAuthentication_InvalidIP_Cert(t *testing.T) {
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	gctx := config.Context{
		TokenVerifier:           verifier.NewTokenVerifier(),
		AuthenticationAuthority: authority.(*authorities.TLSAuthority),
		Accounts: map[string]*account.Account{
			"test-user": {Principal: "test-user", LimitIP: net.IPv4(192, 168, 0, 16)},
		},
	}
	request := prepCertAuth(t, &gctx)
	request.RemoteAddr = "192.168.0.17:50000"
	_, err = attemptAuthentication(&gctx, request)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "from wrong IP address") {
		t.Errorf("Wrong error: %s", err)
	}
}

func TestConfiguredKeyserver_GetClientCAs(t *testing.T) {
	keydata, _, certdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, certdata)
	if err != nil {
		t.Fatal(err)
	}
	ks := &ConfiguredKeyserver{Context: &config.Context{AuthenticationAuthority: authority.(*authorities.TLSAuthority)}}
	subjects := ks.GetClientCAs().Subjects()
	cert, err := x509.ParseCertificate(ks.Context.AuthenticationAuthority.ToHTTPSCert().Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(subjects) != 1 {
		t.Error("Wrong number of subjects.")
	} else if !bytes.Equal(subjects[0], cert.RawSubject) {
		t.Error("Mismatched raw subject bytes.")
	}
}

func TestConfiguredKeyserver_GetServerCert(t *testing.T) {
	keydata, _, cdata := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-ca", nil, nil)
	authority, err := authorities.LoadTLSAuthority(keydata, cdata)
	if err != nil {
		t.Fatal(err)
	}
	ks := &ConfiguredKeyserver{ServerCert: authority.(*authorities.TLSAuthority).ToHTTPSCert()}
	if len(ks.GetServerCert().Certificate) != 1 {
		t.Fatal("Wrong number of certs")
	}
	certdata := ks.GetServerCert().Certificate[0]
	refdata, err := wraputil.LoadSinglePEMBlock(cdata, []string{"CERTIFICATE"})
	if err != nil {
		t.Error(err)
	} else if !bytes.Equal(certdata, refdata) {
		t.Error("Cert mismatch")
	}
}

func TestConfiguredKeyserver_HandleStaticRequest(t *testing.T) {
	ks := &ConfiguredKeyserver{Context: &config.Context{StaticFiles: map[string]config.StaticFile{
		"testa.txt": {Filename: "testa.txt", Filepath: "../config/testdir/testa.txt"},
	}}}
	recorder := httptest.NewRecorder()
	err := ks.HandleStaticRequest(recorder, "testa.txt")
	if err != nil {
		t.Error(err)
	}
	result := recorder.Result()
	if result.Body == nil {
		t.Fatal("Nil body.")
	}
	response, err := ioutil.ReadAll(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := ioutil.ReadFile("../config/testdir/testa.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(response, ref) {
		t.Error("Mismatched file data.")
	}
}

func TestConfiguredKeyserver_HandleStaticRequest_NonexistentEntry(t *testing.T) {
	ks := &ConfiguredKeyserver{Context: &config.Context{}}
	err := ks.HandleStaticRequest(nil, "testa.txt")
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "No such static file") {
		t.Error("Wrong error.")
	}
}

func TestConfiguredKeyserver_HandleStaticRequest_NonexistentFile(t *testing.T) {
	ks := &ConfiguredKeyserver{Context: &config.Context{StaticFiles: map[string]config.StaticFile{
		"testa.txt": {Filename: "testa.txt", Filepath: "../config/testdir/nonexistent.txt"},
	}}}
	err := ks.HandleStaticRequest(nil, "testa.txt")
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Wrong error: %s", err)
	}
}

func TestConfiguredKeyserver_HandlePubRequest_NoAuthority(t *testing.T) {
	ks := &ConfiguredKeyserver{Context: &config.Context{}}
	err := ks.HandlePubRequest(nil, "grant")
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "No such authority") {
		t.Errorf("Wrong error: %s", err)
	}
}

type BrokenConnection struct {
}

func (BrokenConnection) Read(p []byte) (n int, err error) {
	return 0, errors.New("connection cut by a squirrel with scissors")
}

func TestConfiguredKeyserver_HandleAPIRequest_ConnError(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks := &ConfiguredKeyserver{
		Context: &config.Context{
			TokenVerifier: verifier.NewTokenVerifier(),
		},
		Logger: logger,
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api", BrokenConnection{})
	request.Header.Set(verifier.TokenHeader, ks.Context.TokenVerifier.Registry.GrantToken("test-account", time.Minute))
	err := ks.HandleAPIRequest(recorder, request)
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "connection cut by a squirrel with scissors") {
		t.Errorf("Wrong error: %s", err)
	}
	if logrecord.String() != "" {
		t.Error("Unexpected logging.")
	}
}

func TestConfiguredKeyserver_HandleAPIRequest_Unauthed(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks := &ConfiguredKeyserver{
		Context: &config.Context{
			TokenVerifier: verifier.NewTokenVerifier(),
		},
		Logger: logger,
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api", nil)
	err := ks.HandleAPIRequest(recorder, request)
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "No authentication method found in request") {
		t.Errorf("Wrong error: %s", err)
	}
	if logrecord.String() != "" {
		t.Error("Unexpected logging.")
	}
}

func TestConfiguredKeyserver_HandleAPIRequest_NoSuchGrant(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks := &ConfiguredKeyserver{
		Context: &config.Context{
			TokenVerifier: verifier.NewTokenVerifier(),
			Accounts: map[string]*account.Account{
				"test-account": {
					Principal: "test-account",
				},
			},
		},
		Logger: logger,
	}
	request_data := []byte("[{\"api\": \"test-api\", \"body\": \"\"}]")
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api", bytes.NewReader(request_data))
	request.Header.Set(verifier.TokenHeader, ks.Context.TokenVerifier.Registry.GrantToken("test-account", time.Minute))
	err := ks.HandleAPIRequest(recorder, request)
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "could not find API request") {
		t.Errorf("Wrong error: %s", err)
	}
	if logrecord.String() != "" {
		t.Error("Unexpected logging.")
	}
}
