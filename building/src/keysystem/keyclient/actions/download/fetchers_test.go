package download

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"keysystem/api/server"
	"keysystem/keyclient/state"
	"net"
	"net/http"
	"strings"
	"testing"
	"util/testkeyutil"
	"util/testutil"
)

func TestAuthorityFetcher_PrereqsSatisfied(t *testing.T) {
	if (&AuthorityFetcher{}).PrereqsSatisfied() != nil {
		t.Error("should be satisfied")
	}
}

func TestStaticFetcher_PrereqsSatisfied(t *testing.T) {
	if (&StaticFetcher{}).PrereqsSatisfied() != nil {
		t.Error("should be satisfied")
	}
}

func TestAPIFetcher_PrereqsNotSatisfied(t *testing.T) {
	if (&APIFetcher{State: &state.ClientState{Keygrant: nil}}).PrereqsSatisfied() == nil {
		t.Error("should not be satisfied")
	}
}

func TestAPIFetcher_PrereqsSatisfied(t *testing.T) {
	key, cert := testkeyutil.GenerateTLSRootForTests(t, "test", nil, nil)
	cert2 := &tls.Certificate{PrivateKey: key, Certificate: [][]byte{cert.Raw}}
	if (&APIFetcher{State: &state.ClientState{Keygrant: cert2}}).PrereqsSatisfied() != nil {
		t.Error("should be satisfied")
	}
}

func TestStaticFetcher_Info(t *testing.T) {
	if (&StaticFetcher{StaticName: "test-static"}).Info() != "static file test-static" {
		t.Error("wrong info for static file")
	}
}

func TestAuthorityFetcher_Info(t *testing.T) {
	if (&AuthorityFetcher{AuthorityName: "test-authority"}).Info() != "pubkey for authority test-authority" {
		t.Error("wrong info for authority pubkey")
	}
}

func TestAPIFetcher_Info(t *testing.T) {
	if (&APIFetcher{API: "test-api"}).Info() != "result from api test-api" {
		t.Error("wrong info for api fetch")
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

func TestStaticFetcher_Fetch(t *testing.T) {
	stop, _, _, servercacert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/static/staticname" {
			http.Error(writer, "not found", 404)
		} else {
			writer.Write([]byte("*** test contents are here ***\n!!!!\n"))
		}
	})
	defer stop()
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercacert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	data, err := (&StaticFetcher{Keyserver: ks, StaticName: "staticname"}).Fetch()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "*** test contents are here ***\n!!!!\n" {
		t.Error("wrong result")
	}
}

func TestAuthorityFetcher_Fetch(t *testing.T) {
	stop, _, _, servercacert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/pub/authorityname" {
			http.Error(writer, "not found", 404)
		} else {
			writer.Write([]byte("*** test pubkey is here ***\n!!!!\n"))
		}
	})
	defer stop()
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercacert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	data, err := (&AuthorityFetcher{Keyserver: ks, AuthorityName: "authorityname"}).Fetch()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "*** test pubkey is here ***\n!!!!\n" {
		t.Error("wrong result")
	}
}

func TestAPIFetcher_Fetch(t *testing.T) {
	stop, clientcakey, clientcacert, servercacert, addr := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/apirequest" {
			http.Error(writer, "not found", 404)
		} else {
			b, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				http.Error(writer, "error", 500)
			} else {
				if string(b) != "[{\"api\":\"testapi\",\"body\":\"\"}]" {
					t.Error("wrong request", string(b))
					http.Error(writer, "error", 400)
				} else {
					writer.Write([]byte("[\"*** test result ***\\n\"]"))
				}
			}
		}
	})
	defer stop()
	clikey, clicert := testkeyutil.GenerateTLSKeypairForTests(t, "subkey", nil, nil, clientcacert, clientcakey)
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercacert.Raw}), addr)
	if err != nil {
		t.Fatal(err)
	}
	keygrant := &tls.Certificate{PrivateKey: clikey, Certificate: [][]byte{clicert.Raw}}
	data, err := (&APIFetcher{State: &state.ClientState{Keyserver: ks, Keygrant: keygrant}, API: "testapi"}).Fetch()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "*** test result ***\n" {
		t.Error("wrong result")
	}
}

func TestAPIFetcher_Fetch_Fail(t *testing.T) {
	_, servercacert := testkeyutil.GenerateTLSRootForTests(t, "test", nil, nil)
	ks, err := server.NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercacert.Raw}), "localhost:55555")
	if err != nil {
		t.Fatal(err)
	}
	clikey, clicert := testkeyutil.GenerateTLSRootForTests(t, "test2", nil, nil)
	keygrant := &tls.Certificate{PrivateKey: clikey, Certificate: [][]byte{clicert.Raw}}
	_, err = (&APIFetcher{State: &state.ClientState{Keyserver: ks, Keygrant: keygrant}, API: "testapi"}).Fetch()
	testutil.CheckError(t, err, "connection refused")
}
