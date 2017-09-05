package keycommon

import (
	"testing"
	"crypto/tls"
	"net/http"
	"crypto/rsa"
	"crypto/x509"
	"util/testutil"
	"net"
	"strings"
	"context"
	"io/ioutil"
	"encoding/pem"
	"os"
	"keycommon/reqtarget"
	"util/testkeyutil"
	"time"
)

func launchTestServer(t *testing.T, f http.HandlerFunc) (stop func(), clientcakey *rsa.PrivateKey, clientcacert *x509.Certificate, servercert *x509.Certificate, hostname string) {
	time.Sleep(time.Millisecond * 5)
	clientcakey, clientcacert = testkeyutil.GenerateTLSRootForTests(t, "test-ca", nil, nil)
	serverkey, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-ca-2", []string {"localhost" }, nil)
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
	hostname = strings.Replace(ln.Addr().String(), "127.0.0.1", "localhost", 1)

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

func TestLoadKeyserver(t *testing.T) {
	stop, _, _, servercert, hostname := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("test contents\n"))
	})
	defer stop()
	err := ioutil.WriteFile("testdir/cert.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	template, err := ioutil.ReadFile("testdir/test.yaml.in")
	if err != nil {
		t.Fatal(err)
	}
	templated := []byte(strings.Replace(string(template), "{{HOST}}", hostname, 1))
	err = ioutil.WriteFile("testdir/test.yaml", templated, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("testdir/test.yaml")
	ks, _, err := LoadKeyserver("testdir/test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// check validity of hostname
	contents, err := ks.GetStatic("test")
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "test contents\n" {
		t.Error("wrong contents")
	}
	os.Remove("testdir/cert.pem")
}

func TestLoadKeyserver_NoConfiguration(t *testing.T) {
	_, _, err := LoadKeyserver("testdir/nonexistent.yaml")
	testutil.CheckError(t, err, "testdir/nonexistent.yaml: no such file or directory")
}

func TestLoadKeyserver_InvalidTest(t *testing.T) {
	_, _, err := LoadKeyserver("testdir/invalid.yaml")
	testutil.CheckError(t, err, "yaml: unmarshal errors")
}

func TestLoadKeyserver_NoCert(t *testing.T) {
	_, _, err := LoadKeyserver("testdir/nocert.yaml")
	testutil.CheckError(t, err, "open testdir/nonexistent.pem: no such file or directory")
}

func TestLoadKeyserver_InvalidCert(t *testing.T) {
	/*stop, _, _, _, _ := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("test contents\n"))
	})
	defer stop()*/
	err := ioutil.WriteFile("testdir/cert.pem", []byte("this is not a valid authority"), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = LoadKeyserver("testdir/testbase.yaml")
	testutil.CheckError(t, err, "authority certificate: Missing expected PEM header")
	os.Remove("testdir/cert.pem")
}

func TestLoadKeyserverWithCert(t *testing.T) {
	stop, cakey, cacert, servercert, hostname := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/apirequest" {
			http.Error(writer, "Wrong path", 404)
		} else if len(request.TLS.VerifiedChains) != 1 {
			http.Error(writer, "No client cert", 403)
		} else if len(request.TLS.VerifiedChains[0]) != 2 {
			http.Error(writer, "Wrong chain length", 403)
		} else if request.TLS.VerifiedChains[0][0].Subject.CommonName != "test-client" {
			http.Error(writer, "Wrong client cert", 403)
		} else if request.TLS.VerifiedChains[0][1].Subject.CommonName != "test-ca" {
			http.Error(writer, "Wrong CA cert", 403)
		} else {
			data, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				http.Error(writer, "Cannot read", 400)
			} else {
				if string(data) != "[{\"api\":\"apireq\",\"body\":\"apibody\"}]" {
					t.Error("Wrong data")
					http.Error(writer, "Wrong request", 400)
				} else {
					writer.Write([]byte("[\"apiresponse\"]"))
				}
			}
		}
	})
	defer stop()
	testkey, testcert := testkeyutil.GenerateTLSKeypairForTests(t, "test-client", nil, nil, cacert, cakey)
	err := ioutil.WriteFile("testdir/cert.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/grant.key", pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(testkey)}), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("testdir/grant.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: testcert.Raw}), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	template, err := ioutil.ReadFile("testdir/test.yaml.in")
	if err != nil {
		t.Fatal(err)
	}
	templated := []byte(strings.Replace(string(template), "{{HOST}}", hostname, 1))
	err = ioutil.WriteFile("testdir/test.yaml", templated, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("testdir/test.yaml")
	_, rt, err := LoadKeyserverWithCert("testdir/test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	result, err := reqtarget.SendRequest(rt, "apireq", "apibody")
	if err != nil {
		t.Fatal(err)
	}
	if result != "apiresponse" {
		t.Error("Wrong response")
	}
	os.Remove("testdir/cert.pem")
	os.Remove("testdir/grant.key")
	os.Remove("testdir/grant.pem")
}

func TestLoadKeyserverWithCert_NoConfig(t *testing.T) {
	_, _, err := LoadKeyserverWithCert("testdir/nocert.yaml")
	testutil.CheckError(t, err, "open testdir/nonexistent.pem: no such file or directory")
}

func TestLoadKeyserverWithCert_IncompleteCert(t *testing.T) {
	_, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-cert", nil, nil)
	err := ioutil.WriteFile("testdir/cert.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = LoadKeyserverWithCert("testdir/nocertpath.yaml")
	testutil.CheckError(t, err, "expected non-empty path")
	os.Remove("testdir/cert.pem")
}

func TestLoadKeyserverWithCert_IncompleteKey(t *testing.T) {
	_, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-cert", nil, nil)
	err := ioutil.WriteFile("testdir/cert.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = LoadKeyserverWithCert("testdir/nokeypath.yaml")
	testutil.CheckError(t, err, "expected non-empty path")
	os.Remove("testdir/cert.pem")
}

func TestLoadKeyserverWithCert_NoKeyPair(t *testing.T) {
	_, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-cert", nil, nil)
	err := ioutil.WriteFile("testdir/cert.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = LoadKeyserverWithCert("testdir/nokeypair.yaml")
	testutil.CheckError(t, err, "testdir/nonexistent.pem: no such file or directory")
	os.Remove("testdir/cert.pem")
}
