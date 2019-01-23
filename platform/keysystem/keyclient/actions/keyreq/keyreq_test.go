package keyreq

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"keysystem/api/reqtarget"
	"keysystem/api/server"
	"keysystem/keyclient/config"
	"keysystem/keyclient/state"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
	"util/certutil"
	"util/csrutil"
	"util/fileutil"
	"util/testkeyutil"
	"util/testutil"
	"util/wraputil"
)

func TestPrepareRequestOrRenewKeys_TLS(t *testing.T) {
	s := &state.ClientState{}
	act, err := PrepareRequestOrRenewKeys(s, config.ConfigKey{
		Name:      "examplekey",
		Type:      "tls",
		Key:       "testdir/test.key",
		Cert:      "testdir/test.cert",
		API:       "testapi",
		InAdvance: "26h",
	})
	if err != nil {
		t.Fatal(err)
	}
	action := act.(*RequestOrRenewAction)
	if action.InAdvance != time.Hour*26 {
		t.Error("wrong inadvance duration")
	}
	if action.Name != "examplekey" {
		t.Error("wrong name")
	}
	if action.API != "testapi" {
		t.Error("wrong API")
	}
	if action.State != s {
		t.Error("wrong state")
	}
	if action.CertFile != "testdir/test.cert" {
		t.Error("wrong certfile")
	}
	if action.KeyFile != "testdir/test.key" {
		t.Error("wrong keyfile")
	}
	// verify function identity by checking error messages
	_, err = action.GenCSR([]byte{})
	testutil.CheckError(t, err, "Missing expected PEM header")
	_, err = action.CheckExpiration([]byte{})
	testutil.CheckError(t, err, "Missing expected PEM header")
}

func TestPrepareRequestOrRenewKeys_SSH(t *testing.T) {
	for _, ktype := range []string{"ssh", "ssh-pubkey"} {
		s := &state.ClientState{}
		act, err := PrepareRequestOrRenewKeys(s, config.ConfigKey{
			Name:      "examplekey",
			Type:      ktype,
			Key:       "testdir/test",
			Cert:      "testdir/test-cert.pub",
			API:       "testapi",
			InAdvance: "26h",
		})
		if err != nil {
			t.Fatal(err)
		}
		action := act.(*RequestOrRenewAction)
		if action.InAdvance != time.Hour*26 {
			t.Error("wrong inadvance duration")
		}
		if action.Name != "examplekey" {
			t.Error("wrong name")
		}
		if action.API != "testapi" {
			t.Error("wrong API")
		}
		if action.State != s {
			t.Error("wrong state")
		}
		if action.CertFile != "testdir/test-cert.pub" {
			t.Error("wrong certfile")
		}
		if action.KeyFile != "testdir/test" {
			t.Error("wrong keyfile")
		}
		// verify function identity by checking error messages
		_, err = action.GenCSR([]byte{})
		testutil.CheckError(t, err, "ssh: no key found")
		_, err = action.CheckExpiration([]byte{})
		testutil.CheckError(t, err, "ssh: no key found")
	}
}

func TestPrepareRequestOrRenewKeys_BadType(t *testing.T) {
	s := &state.ClientState{}
	_, err := PrepareRequestOrRenewKeys(s, config.ConfigKey{
		Name:      "examplekey",
		Type:      "pin-tumbler",
		Key:       "testdir/test",
		Cert:      "testdir/test-cert.pub",
		API:       "testapi",
		InAdvance: "26h",
	})
	testutil.CheckError(t, err, "unrecognized key type: pin-tumbler")
}

func TestPrepareRequestOrRenewKeys_NoAPI(t *testing.T) {
	s := &state.ClientState{}
	_, err := PrepareRequestOrRenewKeys(s, config.ConfigKey{
		Name:      "examplekey",
		Type:      "ssh",
		Key:       "testdir/test",
		Cert:      "testdir/test-cert.pub",
		InAdvance: "26h",
	})
	testutil.CheckError(t, err, "no renew api provided")
}

func TestPrepareRequestOrRenewKeys_NoInadvance(t *testing.T) {
	s := &state.ClientState{}
	_, err := PrepareRequestOrRenewKeys(s, config.ConfigKey{
		Name: "examplekey",
		Type: "ssh",
		Key:  "testdir/test",
		Cert: "testdir/test-cert.pub",
		API:  "testapi",
	})
	testutil.CheckError(t, err, "invalid in-advance interval for key renewal: time: invalid duration")
}

func TestPrepareRequestOrRenewKeys_ZeroAdvance(t *testing.T) {
	s := &state.ClientState{}
	_, err := PrepareRequestOrRenewKeys(s, config.ConfigKey{
		Name:      "examplekey",
		Type:      "ssh",
		Key:       "testdir/test",
		Cert:      "testdir/test-cert.pub",
		API:       "testapi",
		InAdvance: "0h",
	})
	testutil.CheckError(t, err, "invalid in-advance interval for key renewal: nonpositive duration")
}

func TestPrepareRequestOrRenewKeys_NegativeAdvance(t *testing.T) {
	s := &state.ClientState{}
	_, err := PrepareRequestOrRenewKeys(s, config.ConfigKey{
		Name:      "examplekey",
		Type:      "ssh",
		Key:       "testdir/test",
		Cert:      "testdir/test-cert.pub",
		API:       "testapi",
		InAdvance: "-1h",
	})
	testutil.CheckError(t, err, "invalid in-advance interval for key renewal: nonpositive duration")
}

func TestRequestOrRenewAction_Pending_Nonexistent(t *testing.T) {
	ispending, err := (&RequestOrRenewAction{
		CertFile: "testdir/nonexistent.pem",
	}).Pending()
	if err != nil {
		t.Error(err)
	}
	if !ispending {
		t.Error("should be pending")
	}
}

func TestRequestOrRenewAction_Pending_Inaccessible(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Error(err)
	}
	err = os.Mkdir("testdir/inaccessible", 0)
	if err != nil && !os.IsExist(err) {
		t.Error(err)
	}
	ispending, err := (&RequestOrRenewAction{
		CertFile: "testdir/inaccessible/nonexistent.pem",
	}).Pending()
	if !ispending {
		t.Error("should be pending")
	}
	testutil.CheckError(t, err, "permission denied")
}

func TestRequestOrRenewAction_Pending_MalformedTLS(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Error(err)
	}
	err = ioutil.WriteFile("testdir/malformed.pem", []byte("this is not valid PEM data"), os.FileMode(0644))
	if err != nil {
		t.Error(err)
	}
	ispending, err := (&RequestOrRenewAction{
		CertFile:        "testdir/malformed.pem",
		CheckExpiration: certutil.CheckTLSCertExpiration,
	}).Pending()
	if !ispending {
		t.Error("should be pending")
	}
	testutil.CheckError(t, err, "while trying to check expiration status of certificate: Missing expected PEM header")
	os.Remove("testdir/malformed.pem")
}

func TestRequestOrRenewAction_Pending_MalformedSSH(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Error(err)
	}
	err = ioutil.WriteFile("testdir/malformed.pem", []byte("this is not valid PEM data"), os.FileMode(0644))
	if err != nil {
		t.Error(err)
	}
	ispending, err := (&RequestOrRenewAction{
		CertFile:        "testdir/malformed.pem",
		CheckExpiration: certutil.CheckSSHCertExpiration,
	}).Pending()
	if !ispending {
		t.Error("should be pending")
	}
	testutil.CheckError(t, err, "while trying to check expiration status of certificate: ssh: no key found")
	os.Remove("testdir/malformed.pem")
}

func TestRequestOrRenewAction_Pending_NonRenewable(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Error(err)
	}
	_, cert := testkeyutil.GenerateTLSRootForTests(t, "test", nil, nil)
	err = ioutil.WriteFile("testdir/real.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), os.FileMode(0644))
	if err != nil {
		t.Error(err)
	}
	ispending, err := (&RequestOrRenewAction{
		CertFile:        "testdir/real.pem",
		CheckExpiration: certutil.CheckTLSCertExpiration,
		InAdvance:       time.Minute * 30, // less than the default time.Hour for temporary certs
	}).Pending()
	if err != nil {
		t.Error(err)
	}
	if ispending {
		t.Error("should not be pending")
	}
	os.Remove("testdir/real.pem")
}

func TestRequestOrRenewAction_Pending_Renewable(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Error(err)
	}
	_, cert := testkeyutil.GenerateTLSRootForTests(t, "test", nil, nil)
	err = ioutil.WriteFile("testdir/real.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}), os.FileMode(0644))
	if err != nil {
		t.Error(err)
	}
	ispending, err := (&RequestOrRenewAction{
		CertFile:        "testdir/real.pem",
		CheckExpiration: certutil.CheckTLSCertExpiration,
		InAdvance:       time.Minute * 90, // longer than the default time.Hour for temporary certs
	}).Pending()
	if err != nil {
		t.Error(err)
	}
	if !ispending {
		t.Error("should be pending")
	}
	os.Remove("testdir/real.pem")
}

func TestRequestOrRenewAction_CheckBlocker_NoGrant(t *testing.T) {
	blocked := (&RequestOrRenewAction{
		State: &state.ClientState{
			Keygrant: nil,
		},
	}).CheckBlocker()
	testutil.CheckError(t, blocked, "no keygranting certificate ready")
}

func TestRequestOrRenewAction_CheckBlocker_YesGrant_NoKey(t *testing.T) {
	blocked := (&RequestOrRenewAction{
		State: &state.ClientState{
			Keygrant: &tls.Certificate{}, // stub because it doesn't matter
		},
		KeyFile: "testdir/nonexistent.key",
	}).CheckBlocker()
	testutil.CheckError(t, blocked, "key does not yet exist: testdir/nonexistent.key")
}

func TestRequestOrRenewAction_CheckBlocker_YesGrant_YesKey(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/test.key", []byte("test"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	blocked := (&RequestOrRenewAction{
		State: &state.ClientState{
			Keygrant: &tls.Certificate{}, // stub because it doesn't matter
		},
		KeyFile: "testdir/test.key",
	}).CheckBlocker()
	if blocked != nil {
		t.Error(blocked)
	}
	err = os.Remove("testdir/test.key")
	if err != nil {
		t.Fatal(err)
	}
}

func launchTestServer(t *testing.T, f http.HandlerFunc) (stop func(), clientcakey *rsa.PrivateKey, clientcacert *x509.Certificate, servercert *x509.Certificate, addr string) {
	clientcakey, clientcacert = testkeyutil.GenerateTLSRootForTests(t, "test-ca", nil, nil)
	serverkey, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-ca-2", []string{"localhost"}, nil)
	pool := x509.NewCertPool()
	pool.AddCert(clientcacert)
	srv := &http.Server{
		Addr:    "localhost:0",
		Handler: f,
		TLSConfig: &tls.Config{
			ClientAuth:   tls.VerifyClientCertIfGiven,
			ClientCAs:    pool,
			Certificates: []tls.Certificate{{PrivateKey: serverkey, Certificate: [][]byte{servercert.Raw}}},
			MinVersion:   tls.VersionTLS12,
			NextProtos:   []string{"http/1.1", "h2"},
		},
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	addr = strings.Replace(ln.Addr().String(), "127.0.0.1", "localhost", 1)

	done := make(chan bool)

	go func() {
		tlsListener := tls.NewListener(ln, srv.TLSConfig)
		err := srv.Serve(tlsListener)
		if err == http.ErrServerClosed || strings.Contains(err.Error(), "use of closed network connection") {
			done <- false
		} else {
			t.Error(err)
			done <- true
		}
	}()

	stop = func() {
		err := srv.Shutdown(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		err = ln.Close()
		if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			t.Fatal(err)
		}
		if <-done {
			t.Fatal("error reported")
		}
	}
	return
}

func TestRequestOrRenewAction_Perform_TLS(t *testing.T) {
	// generate the key file
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/testkey.key",
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	rid, err := rand.Int(rand.Reader, big.NewInt(0xFFFFFFFFFFF))
	if err != nil {
		t.Fatal(err)
	}
	thisid := fmt.Sprint("testid-%x", rid)
	// launch the transient server
	stop, clientcakey, clientcacert, servercert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/apirequest" {
			http.Error(writer, "invalid path", 404)
		} else if len(request.TLS.VerifiedChains) != 1 {
			http.Error(writer, "invalid auth", 403)
		} else if len(request.TLS.VerifiedChains[0]) != 2 {
			http.Error(writer, "invalid auth", 403)
		} else if request.TLS.VerifiedChains[0][0].Subject.CommonName != "test-client-keyreq" {
			http.Error(writer, "invalid auth", 403)
		} else {
			contents, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				return
			}
			request := []reqtarget.Request{}
			err = json.Unmarshal(contents, &request)
			if err != nil {
				t.Error(err)
				return
			}
			if len(request) != 1 {
				t.Error("wrong count")
				return
			}
			if request[0].API != "testapi" {
				t.Error("wrong request")
			}
			csr, err := wraputil.LoadX509CSRFromPEM([]byte(request[0].Body))
			if err != nil {
				t.Error(err)
				return
			}
			if csr.Subject.CommonName != "invalid-cn-temporary-request" {
				t.Error("wrong common name for CSR")
			}
			err = csr.CheckSignature()
			if err != nil {
				t.Error(err)
			}
			if csr.SignatureAlgorithm != x509.SHA256WithRSA {
				t.Error("wrong signature algorithm")
			}
			if csr.PublicKey.(*rsa.PublicKey).N.Cmp(key.N) != 0 {
				t.Error("wrong public key")
			}
			writer.Write([]byte("[\"this is definitely not a certificate but you don't care.\\n" + thisid + "\\n\"]"))
		}
	})
	defer stop()
	// generate the client auth cert
	clikey, clicert := testkeyutil.GenerateTLSKeypairForTests(t, "test-client-keyreq", nil, nil, clientcacert, clientcakey)
	keyserver, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	// the actual action setup
	action := &RequestOrRenewAction{
		KeyFile:  "testdir/testkey.key",
		CertFile: "testdir/testcert.pem",
		State: &state.ClientState{
			Keyserver: keyserver,
			Keygrant:  &tls.Certificate{PrivateKey: clikey, Certificate: [][]byte{clicert.Raw}},
		},
		API:    "testapi",
		GenCSR: csrutil.BuildTLSCSR,
	}
	// the actual request
	err = action.Perform(nil)
	if err != nil {
		t.Error(err)
	}
	contents, err := ioutil.ReadFile("testdir/testcert.pem")
	if err != nil {
		t.Error(err)
	}
	if string(contents) != "this is definitely not a certificate but you don't care.\n"+thisid+"\n" {
		t.Error("wrong contents")
	}
	os.Remove("testdir/testcert.pem")
	os.Remove("testdir/testkey.key")
}

func TestRequestOrRenewAction_Perform_SSH(t *testing.T) {
	// generate the key file
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	pubkey, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/testkey.pub", ssh.MarshalAuthorizedKey(pubkey), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	rid, err := rand.Int(rand.Reader, big.NewInt(0xFFFFFFFFFFF))
	if err != nil {
		t.Fatal(err)
	}
	thisid := fmt.Sprint("testid-%x", rid)
	// launch the transient server
	stop, clientcakey, clientcacert, servercert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/apirequest" {
			http.Error(writer, "invalid path", 404)
		} else if len(request.TLS.VerifiedChains) != 1 {
			http.Error(writer, "invalid auth", 403)
		} else if len(request.TLS.VerifiedChains[0]) != 2 {
			http.Error(writer, "invalid auth", 403)
		} else if request.TLS.VerifiedChains[0][0].Subject.CommonName != "test-client-keyreq" {
			http.Error(writer, "invalid auth", 403)
		} else {
			contents, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				return
			}
			request := []reqtarget.Request{}
			err = json.Unmarshal(contents, &request)
			if err != nil {
				t.Error(err)
				return
			}
			if len(request) != 1 {
				t.Error("wrong count")
				return
			}
			if request[0].API != "testapi" {
				t.Error("wrong request")
			}
			pubkey2, err := wraputil.ParseSSHTextPubkey([]byte(request[0].Body))
			if err != nil {
				t.Error(err)
				return
			}
			if pubkey.Type() != pubkey2.Type() || !bytes.Equal(pubkey.Marshal(), pubkey2.Marshal()) {
				t.Error("wrong public key")
			}
			writer.Write([]byte("[\"this is definitely not a certificate but you don't care.\\n" + thisid + "\\n\"]"))
		}
	})
	defer stop()
	// generate the client auth cert
	clikey, clicert := testkeyutil.GenerateTLSKeypairForTests(t, "test-client-keyreq", nil, nil, clientcacert, clientcakey)
	keyserver, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	// the actual action setup
	action := &RequestOrRenewAction{
		KeyFile:  "testdir/testkey.pub",
		CertFile: "testdir/testkey-cert.pub",
		State: &state.ClientState{
			Keyserver: keyserver,
			Keygrant:  &tls.Certificate{PrivateKey: clikey, Certificate: [][]byte{clicert.Raw}},
		},
		API:    "testapi",
		GenCSR: csrutil.BuildSSHCSR,
	}
	// the actual request
	err = action.Perform(nil)
	if err != nil {
		t.Error(err)
	}
	contents, err := ioutil.ReadFile("testdir/testkey-cert.pub")
	if err != nil {
		t.Error(err)
	}
	if string(contents) != "this is definitely not a certificate but you don't care.\n"+thisid+"\n" {
		t.Error("wrong contents")
	}
	os.Remove("testdir/testkey-cert.pub")
	os.Remove("testdir/testkey.pub")
}

func TestRequestOrRenewAction_Perform_TLS_ResultFail(t *testing.T) {
	// generate the key file
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/testkey.key",
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	// launch the transient server
	stop, clientcakey, clientcacert, servercert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("[\"not a cert but you don't care\"]"))
	})
	defer stop()
	// generate the client auth cert
	clikey, clicert := testkeyutil.GenerateTLSKeypairForTests(t, "test-client-keyreq", nil, nil, clientcacert, clientcakey)
	keyserver, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	// the actual action setup
	action := &RequestOrRenewAction{
		KeyFile:  "testdir/testkey.key",
		CertFile: "testdir/nonexistent/testcert.pem",
		State: &state.ClientState{
			Keyserver: keyserver,
			Keygrant:  &tls.Certificate{PrivateKey: clikey, Certificate: [][]byte{clicert.Raw}},
		},
		API:    "testapi",
		GenCSR: csrutil.BuildTLSCSR,
	}
	// the actual request
	err = action.Perform(nil)
	testutil.CheckError(t, err, "nonexistent/testcert.pem: no such file or directory")
	os.Remove("testdir/testkey.key")
}

func TestRequestOrRenewAction_Perform_TLS_NoResult(t *testing.T) {
	// generate the key file
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/testkey.key",
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	// launch the transient server
	stop, clientcakey, clientcacert, servercert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("[\"\"]"))
	})
	defer stop()
	// generate the client auth cert
	clikey, clicert := testkeyutil.GenerateTLSKeypairForTests(t, "test-client-keyreq", nil, nil, clientcacert, clientcakey)
	keyserver, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	// the actual action setup
	action := &RequestOrRenewAction{
		KeyFile:  "testdir/testkey.key",
		CertFile: "testdir/testcert.pem",
		State: &state.ClientState{
			Keyserver: keyserver,
			Keygrant:  &tls.Certificate{PrivateKey: clikey, Certificate: [][]byte{clicert.Raw}},
		},
		API:    "testapi",
		GenCSR: csrutil.BuildTLSCSR,
	}
	// the actual request
	err = action.Perform(nil)
	testutil.CheckError(t, err, "received empty response")
	os.Remove("testdir/testkey.key")
}

func TestRequestOrRenewAction_Perform_TLS_NoServer(t *testing.T) {
	// generate the key file
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/testkey.key",
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	// generate the client auth cert
	_, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-server-keyreq", nil, nil)
	clikey, clicert := testkeyutil.GenerateTLSRootForTests(t, "test-client-keyreq", nil, nil)
	keyserver, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost:55555")
	if err != nil {
		t.Fatal(err)
	}
	// the actual action setup
	action := &RequestOrRenewAction{
		KeyFile:  "testdir/testkey.key",
		CertFile: "testdir/testcert.pem",
		State: &state.ClientState{
			Keyserver: keyserver,
			Keygrant:  &tls.Certificate{PrivateKey: clikey, Certificate: [][]byte{clicert.Raw}},
		},
		API:    "testapi",
		GenCSR: csrutil.BuildTLSCSR,
	}
	// the actual request
	err = action.Perform(nil)
	testutil.CheckError(t, err, "connection refused")
	os.Remove("testdir/testkey.pem")
}

func TestRequestOrRenewAction_Perform_TLS_InvalidKey(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/testkey.key", pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("JUNK")}), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	// the actual action setup
	action := &RequestOrRenewAction{
		KeyFile: "testdir/testkey.key",
		GenCSR:  csrutil.BuildTLSCSR,
	}
	// the actual request
	err = action.Perform(nil)
	testutil.CheckError(t, err, "could not load PEM private key as PKCS#1 or PKCS#8")
	os.Remove("testdir/testkey.key")
}

func TestRequestOrRenewAction_Perform_TLS_NoKey(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	// the actual action setup
	action := &RequestOrRenewAction{
		KeyFile: "testdir/nonexistent.key",
	}
	// the actual request
	err = action.Perform(nil)
	testutil.CheckError(t, err, "testdir/nonexistent.key: no such file or directory")
}

func TestRequestOrRenewAction_Info(t *testing.T) {
	act := &RequestOrRenewAction{
		KeyFile:   "testdir/testkey.key",
		CertFile:  "testdir/testcert.pem",
		API:       "testapi",
		InAdvance: time.Hour * 2,
		Name:      "testname",
	}
	if act.Info() != "req/renew testname from key testdir/testkey.key into cert testdir/testcert.pem with API testapi in advance by 2h0m0s" {

	}
}
