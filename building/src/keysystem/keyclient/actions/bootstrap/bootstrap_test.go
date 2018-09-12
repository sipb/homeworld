package bootstrap

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"keysystem/keyclient/config"
	"keysystem/keyclient/state"
	"keysystem/api/reqtarget"
	"keysystem/api/server"
	"keysystem/keyserver/authorities"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
	"util/fileutil"
	"util/testkeyutil"
	"util/testutil"
	"util/wraputil"
)

func TestPrepareBootstrapAction(t *testing.T) {
	s := &state.ClientState{}
	act, err := PrepareBootstrapAction(s, "testdir/testpath.token", "testapi")
	if err != nil {
		t.Fatal(err)
	}
	if act.(*BootstrapAction).State != s {
		t.Error("wrong state")
	}
	if act.(*BootstrapAction).TokenFilePath != "testdir/testpath.token" {
		t.Error("wrong token path")
	}
	if act.(*BootstrapAction).TokenAPI != "testapi" {
		t.Error("wrong token API")
	}
}

func TestPrepareBootstrapAction_InvalidAPI(t *testing.T) {
	_, err := PrepareBootstrapAction(nil, "testdir/testpath.token", "")
	testutil.CheckError(t, err, "no bootstrap api provided")
}

func TestBootstrapAction_CheckBlocker_NoKey(t *testing.T) {
	err := (&BootstrapAction{
		State: &state.ClientState{
			Config: config.Config{
				KeyPath: "testdir/nonexistent.key",
			},
		},
	}).CheckBlocker()
	testutil.CheckError(t, err, "key does not yet exist: testdir/nonexistent.key")
}

func TestBootstrapAction_CheckBlocker_YesKey(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/existent.key", []byte("test"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = (&BootstrapAction{
		State: &state.ClientState{
			Config: config.Config{
				KeyPath: "testdir/existent.key",
			},
		},
	}).CheckBlocker()
	if err != nil {
		t.Error(err)
	}
	err = os.Remove("testdir/existent.key")
	if err != nil {
		t.Fatal(err)
	}
}

func TestBootstrapAction_Pending_NoKeygrant_NoFile(t *testing.T) {
	act := &BootstrapAction{State: &state.ClientState{Keygrant: nil}, TokenFilePath: "testdir/nonexistent.token"}
	ispending, err := act.Pending()
	if err != nil {
		t.Error(err)
	}
	if ispending {
		t.Error("should not be pending")
	}
}

func TestBootstrapAction_Pending_NoKeygrant_YesFile(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/existent.token", []byte("test"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	act := &BootstrapAction{State: &state.ClientState{Keygrant: nil}, TokenFilePath: "testdir/existent.token"}
	ispending, err := act.Pending()
	if err != nil {
		t.Error(err)
	}
	if !ispending {
		t.Error("should be pending")
	}
}

func TestBootstrapAction_Pending_YesKeygrant_NoFile(t *testing.T) {
	act := &BootstrapAction{State: &state.ClientState{Keygrant: &tls.Certificate{}}, TokenFilePath: "testdir/nonexistent.token"}
	ispending, err := act.Pending()
	if err != nil {
		t.Error(err)
	}
	if ispending {
		t.Error("should not be pending")
	}
}

func TestBootstrapAction_Pending_YesKeygrant_YesFile(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/existent.token", []byte("test"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	act := &BootstrapAction{State: &state.ClientState{Keygrant: &tls.Certificate{}}, TokenFilePath: "testdir/existent.token"}
	ispending, err := act.Pending()
	if err != nil {
		t.Error(err)
	}
	if ispending {
		t.Error("should not be pending")
	}
	os.Remove("testdir/existent.token")
}

func TestGetToken(t *testing.T) {
	err := ioutil.WriteFile("testdir/test.token", []byte("  this-is-a-sample-token. \n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	tok, err := (&BootstrapAction{TokenFilePath: "testdir/test.token"}).getToken()
	if err != nil {
		t.Fatal(err)
	}
	if tok != "this-is-a-sample-token." {
		t.Error("wrong token")
	}
	os.Remove("testdir/test.token")
}

func TestGetToken_Nonexistent(t *testing.T) {
	_, err := (&BootstrapAction{TokenFilePath: "testdir/nonexistent.token"}).getToken()
	testutil.CheckError(t, err, "testdir/nonexistent.token: no such file or directory")
}

func TestGetToken_Blank(t *testing.T) {
	err := ioutil.WriteFile("testdir/test.token", []byte("  \n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&BootstrapAction{TokenFilePath: "testdir/test.token"}).getToken()
	testutil.CheckError(t, err, "blank token found")
}

func TestGetToken_InvalidCharacter(t *testing.T) {
	err := ioutil.WriteFile("testdir/test.token", []byte(" \x80 \n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&BootstrapAction{TokenFilePath: "testdir/test.token"}).getToken()
	testutil.CheckError(t, err, "invalid token found: bad character '")
}

func TestGetToken_InnerSpace(t *testing.T) {
	err := ioutil.WriteFile("testdir/test.token", []byte(" left-left- -right-right \n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&BootstrapAction{TokenFilePath: "testdir/test.token"}).getToken()
	testutil.CheckError(t, err, "invalid token found: bad character ' '")
}

func TestGetToken_NotPrintable(t *testing.T) {
	err := ioutil.WriteFile("testdir/test.token", []byte(" \x01 \n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&BootstrapAction{TokenFilePath: "testdir/test.token"}).getToken()
	testutil.CheckError(t, err, "invalid token found: bad character '\x01'")
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

func TestSendRequest(t *testing.T) {
	token := fmt.Sprintf("thisisanauthtoken%d", time.Now().Unix())
	stop, _, _, servercert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/apirequest" {
			if request.Header.Get("X-Bootstrap-Token") != token {
				http.Error(writer, "unauthed", 403)
				return
			}
			contents, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				return
			}
			if string(contents) != "[{\"api\":\"testapi\",\"body\":\"testparam\"}]" {
				t.Error("wrong contents")
				return
			}
			writer.Write([]byte("[\"testresponse\"]"))
		} else {
			http.Error(writer, "wrong path", 404)
		}
	})
	defer stop()
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/test.token", []byte(" "+token+"\n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	response, err := (&BootstrapAction{
		TokenFilePath: "testdir/test.token",
		State: &state.ClientState{
			Keyserver: ks,
		},
	}).sendRequest("testapi", "testparam")
	if err != nil {
		t.Error(err)
	}
	if response != "testresponse" {
		t.Error("wrong response")
	}
	if fileutil.Exists("testdir/test.token") {
		t.Error("token should have been deleted")
	}
}

func TestSendRequest_CannotRemove(t *testing.T) {
	token := fmt.Sprintf("thisisanauthtoken%d", time.Now().Unix())
	stop, _, _, servercert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/apirequest" {
			if request.Header.Get("X-Bootstrap-Token") != token {
				http.Error(writer, "unauthed", 403)
				return
			}
			writer.Write([]byte("[\"testresponse\"]"))
		} else {
			http.Error(writer, "wrong path", 404)
		}
	})
	defer stop()
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir/limited")
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chmod("testdir/limited", 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/limited/test.token", []byte(" "+token+"\n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chmod("testdir/limited", 0555)
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&BootstrapAction{
		TokenFilePath: "testdir/limited/test.token",
		State: &state.ClientState{
			Keyserver: ks,
		},
	}).sendRequest("testapi", "testparam")
	testutil.CheckError(t, err, "testdir/limited/test.token: permission denied")
	err = os.Chmod("testdir/limited", 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/limited/test.token")
	if err != nil {
		t.Error(err)
	}
}

func TestSendRequest_NoServer(t *testing.T) {
	_, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-server", nil, nil)
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost:55555")
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/test.token", []byte(" testtoken\n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	_, err = (&BootstrapAction{
		TokenFilePath: "testdir/test.token",
		State: &state.ClientState{
			Keyserver: ks,
		},
	}).sendRequest("testapi", "testparam")
	testutil.CheckError(t, err, "connection refused")
	err = os.Remove("testdir/test.token")
	if err != nil {
		t.Error(err)
	}
}

func TestSendRequest_NoToken(t *testing.T) {
	_, err := (&BootstrapAction{
		TokenFilePath: "testdir/nonexistent.token",
	}).sendRequest("testapi", "testparam")
	testutil.CheckError(t, err, "testdir/nonexistent.token: no such file or directory")
}

func TestBuildCSR_NoKey(t *testing.T) {
	act := &BootstrapAction{
		State: &state.ClientState{
			Config: config.Config{
				KeyPath: "testdir/nonexistent.key",
			},
		},
	}
	_, err := act.buildCSR()
	testutil.CheckError(t, err, "testdir/nonexistent.key: no such file or directory")
}

func TestBuildCSR_InvalidKey(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/invalid.key", []byte("this is not a key"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	act := &BootstrapAction{
		State: &state.ClientState{
			Config: config.Config{
				KeyPath: "testdir/invalid.key",
			},
		},
	}
	_, err = act.buildCSR()
	testutil.CheckError(t, err, "Missing expected PEM header")
	err = os.Remove("testdir/invalid.key")
	if err != nil {
		t.Error(err)
	}
}

func TestBuildCSR(t *testing.T) {
	err := fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	keypem, _, _ := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	err = ioutil.WriteFile("testdir/test.key", keypem, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	act := &BootstrapAction{
		State: &state.ClientState{
			Config: config.Config{
				KeyPath: "testdir/test.key",
			},
		},
	}
	csrdata, err := act.buildCSR()
	if err != nil {
		t.Fatal(err)
	}
	csr, err := wraputil.LoadX509CSRFromPEM(csrdata)
	if err != nil {
		t.Fatal(err)
	}
	if csr.Subject.CommonName != "invalid-cn-temporary-request" {
		t.Error("wrong common name")
	}
	err = os.Remove("testdir/test.key")
	if err != nil {
		t.Error(err)
	}
}

func TestPerform_NoKey(t *testing.T) {
	act := &BootstrapAction{
		State: &state.ClientState{
			Config: config.Config{
				KeyPath: "testdir/nonexistent.key",
			},
		},
	}
	err := act.Perform(nil)
	testutil.CheckError(t, err, "testdir/nonexistent.key: no such file or directory")
}

func TestPerform_NoServer(t *testing.T) {
	_, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-server", nil, nil)
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost:55555")
	if err != nil {
		t.Fatal(err)
	}
	keypem, _, _ := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	err = ioutil.WriteFile("testdir/test.key", keypem, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/test.token", []byte("testtoken\n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	act := &BootstrapAction{
		State: &state.ClientState{
			Keyserver: ks,
			Config: config.Config{
				KeyPath: "testdir/test.key",
			},
		},
		TokenFilePath: "testdir/test.token",
	}
	err = act.Perform(nil)
	testutil.CheckError(t, err, "connection refused")
	err = os.Remove("testdir/test.key")
	if err != nil {
		t.Error(err)
	}
}

func TestPerform(t *testing.T) {
	authoritykey, _, authoritycert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-authority", nil, nil)
	authority, err := authorities.LoadTLSAuthority(authoritykey, authoritycert)
	if err != nil {
		t.Fatal(err)
	}
	tlsauthority := authority.(*authorities.TLSAuthority)

	token := fmt.Sprintf("thisisanauthtoken%d", time.Now().Unix())
	stop, _, _, servercert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/apirequest" {
			if request.Header.Get("X-Bootstrap-Token") != token {
				http.Error(writer, "unauthed", 403)
				return
			}
			contents, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				return
			}
			requests := []reqtarget.Request{}
			err = json.Unmarshal(contents, &requests)
			if err != nil {
				t.Error(err)
				return
			}
			if len(requests) != 1 || requests[0].API != "testapi" {
				t.Error("invalid contents")
				return
			}

			signed, err := tlsauthority.Sign(requests[0].Body, false, time.Hour, "test-signed", []string{"test.example.com", "127.0.0.1"})
			if err != nil {
				t.Error(err)
				return
			}

			certdata, err := json.Marshal([]string{signed})
			if err != nil {
				t.Error(err)
				return
			}

			writer.Write(certdata)
		} else {
			http.Error(writer, "wrong path", 404)
		}
	})
	defer stop()
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/test.token", []byte(" "+token+"\n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	keypem, _, _ := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	err = ioutil.WriteFile("testdir/test.key", keypem, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/test.pem")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = (&BootstrapAction{
		TokenFilePath: "testdir/test.token",
		TokenAPI:      "testapi",
		State: &state.ClientState{
			Config: config.Config{
				KeyPath:  "testdir/test.key",
				CertPath: "testdir/test.pem",
			},
			Keyserver: ks,
		},
	}).Perform(nil)
	if err != nil {
		t.Error(err)
	}
	if fileutil.Exists("testdir/test.token") {
		t.Error("should not exist")
	}
	certdata, err := ioutil.ReadFile("testdir/test.pem")
	if err != nil {
		t.Fatal(err)
	}
	cert, err := wraputil.LoadX509CertFromPEM(certdata)
	if err != nil {
		t.Fatal(err)
	}
	name, err := tlsauthority.Verify(&http.Request{TLS: &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}})
	if err != nil {
		t.Fatal(err)
	}
	if name != "test-signed" {
		t.Error("wrong principal in cert")
	}
	err = os.Remove("testdir/test.key")
	if err != nil {
		t.Error(err)
	}
	err = os.Remove("testdir/test.pem")
	if err != nil {
		t.Fatal(err)
	}
}

func TestPerform_NoResponse(t *testing.T) {
	stop, _, _, servercert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/apirequest" {
			writer.Write([]byte("[\"\"]"))
		} else {
			http.Error(writer, "wrong path", 404)
		}
	})
	defer stop()
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/test.token", []byte("test\n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	keypem, _, _ := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	err = ioutil.WriteFile("testdir/test.key", keypem, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/test.pem")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = (&BootstrapAction{
		TokenFilePath: "testdir/test.token",
		TokenAPI:      "testapi",
		State: &state.ClientState{
			Config: config.Config{
				KeyPath:  "testdir/test.key",
				CertPath: "testdir/test.pem",
			},
			Keyserver: ks,
		},
	}).Perform(nil)
	testutil.CheckError(t, err, "received empty response")
	if fileutil.Exists("testdir/test.token") {
		t.Error("should not exist")
	}
	err = os.Remove("testdir/test.key")
	if err != nil {
		t.Error(err)
	}
}

func TestPerform_InvalidResponse(t *testing.T) {
	stop, _, _, servercert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/apirequest" {
			writer.Write([]byte("[\"this is not a cert\"]"))
		} else {
			http.Error(writer, "wrong path", 404)
		}
	})
	defer stop()
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	err = fileutil.EnsureIsFolder("testdir")
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/test.token", []byte("test\n"), os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	keypem, _, _ := testkeyutil.GenerateTLSRootPEMsForTests(t, "test", nil, nil)
	err = ioutil.WriteFile("testdir/test.key", keypem, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	err = os.Remove("testdir/test.pem")
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	err = (&BootstrapAction{
		TokenFilePath: "testdir/test.token",
		TokenAPI:      "testapi",
		State: &state.ClientState{
			Config: config.Config{
				KeyPath:  "testdir/test.key",
				CertPath: "testdir/test.pem",
			},
			Keyserver: ks,
		},
	}).Perform(nil)
	testutil.CheckError(t, err, "failed to reload keygranting certificate: tls: failed to find any PEM data in certificate input")
	if fileutil.Exists("testdir/test.token") {
		t.Error("should not exist")
	}
	err = os.Remove("testdir/test.key")
	if err != nil {
		t.Error(err)
	}
	err = os.Remove("testdir/test.pem")
	if err != nil {
		t.Fatal(err)
	}
}

func TestBootstrapAction_Info(t *testing.T) {
	if (&BootstrapAction{
		TokenFilePath: "testdir/test.token",
		TokenAPI:      "testapi",
	}).Info() != "bootstrap with token API testapi from path testdir/test.token" {
		t.Error("wrong info")
	}
}
