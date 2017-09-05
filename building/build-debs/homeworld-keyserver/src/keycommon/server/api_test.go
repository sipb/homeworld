package server

import (
	"testing"
	"util/testutil"
	"crypto/tls"
	"net/http"
	"crypto/rsa"
	"crypto/x509"
	"net"
	"strings"
	"context"
	"encoding/pem"
	"util/testkeyutil"
)

func launchTestServer(t *testing.T, f http.HandlerFunc) (stop func(), clientcakey *rsa.PrivateKey, clientcacert *x509.Certificate, servercert *x509.Certificate) {
	clientcakey, clientcacert = testkeyutil.GenerateTLSRootForTests(t, "test-ca", nil, nil)
	serverkey, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-ca-2", []string {"localhost" }, nil)
	pool := x509.NewCertPool()
	pool.AddCert(clientcacert)
	srv := &http.Server{
		Addr:    "localhost:20557",
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

func TestNewKeyserver_NoAuthority(t *testing.T) {
	_, err := NewKeyserver([]byte{}, "localhost")
	testutil.CheckError(t, err, "Missing expected PEM header")
}

func TestNewKeyserver_InvalidAuthority(t *testing.T) {
	_, err := NewKeyserver([]byte("-----BEGIN CERTIFICATE-----\ninvalid"), "localhost")
	testutil.CheckError(t, err, "Could not parse PEM data")
}

func TestKeyserver_GetStatic(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/static/local.conf" {
			http.Error(writer, "No such file", 404)
		} else {
			writer.Write([]byte("Example contents.\n"))
		}
	})
	defer stop()
	ks, err := NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	data, err := ks.GetStatic("local.conf")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Example contents.\n" {
		t.Error("Wrong contents.")
	}
}

func TestKeyserver_GetStatic_NoStatic(t *testing.T) {
	_, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-serv", nil, nil)
	ks, err := NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ks.GetStatic("")
	testutil.CheckError(t, err, "Static filename is empty.")
}

func TestKeyserver_GetPubkey(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/pub/testauthority" {
			http.Error(writer, "No such file", 404)
		} else {
			writer.Write([]byte("Example key.\n"))
		}
	})
	defer stop()
	ks, err := NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	data, err := ks.GetPubkey("testauthority")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Example key.\n" {
		t.Error("Wrong pubkey.")
	}
}

func TestKeyserver_GetPublic_NoStatic(t *testing.T) {
	_, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-serv", nil, nil)
	ks, err := NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ks.GetPubkey("")
	testutil.CheckError(t, err, "Authority name is empty.")
}
