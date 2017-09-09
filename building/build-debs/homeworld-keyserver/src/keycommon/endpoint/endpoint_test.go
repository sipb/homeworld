package endpoint

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
	"util/testkeyutil"
	"util/testutil"
)

func createBaseEndpoint(t *testing.T, rootcert *x509.Certificate) ServerEndpoint {
	pool := x509.NewCertPool()
	if rootcert == nil {
		_, rootcert = testkeyutil.GenerateTLSRootForTests(t, "test-root", nil, nil)
	}
	pool.AddCert(rootcert)
	endpoint, err := NewServerEndpoint("https://localhost:50001/test/", pool)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.rootCAs != pool {
		t.Fatal("pool mismatch")
	}
	return endpoint
}

func launchTestServer(t *testing.T, f http.HandlerFunc) (stop func(), clientcakey *rsa.PrivateKey, clientcacert *x509.Certificate, servercert *x509.Certificate) {
	clientcakey, clientcacert = testkeyutil.GenerateTLSRootForTests(t, "test-ca", nil, nil)
	serverkey, servercert := testkeyutil.GenerateTLSRootForTests(t, "test-ca-2", []string{"localhost"}, nil)
	pool := x509.NewCertPool()
	pool.AddCert(clientcacert)
	srv := &http.Server{
		Addr:    "localhost:50001",
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

func TestNewServerEndpoint(t *testing.T) {
	endpoint := createBaseEndpoint(t, nil)
	if len(endpoint.extraHeaders) != 0 {
		t.Error("Extraneous headers.")
	}
	if len(endpoint.certificates) != 0 {
		t.Error("Extraneous headers.")
	}
	if endpoint.baseURL != "https://localhost:50001/test/" {
		t.Error("Wrong base URL")
	}
	if endpoint.timeout != time.Second*30 {
		t.Error("Wrong default timeout")
	}
	if len(endpoint.rootCAs.Subjects()) != 1 {
		t.Error("Wrong CAs")
	}
}

func TestNewServerEndpoint_NoURL(t *testing.T) {
	pool := x509.NewCertPool()
	_, rootcert := testkeyutil.GenerateTLSRootForTests(t, "test-root", nil, nil)
	pool.AddCert(rootcert)
	_, err := NewServerEndpoint("", pool)
	testutil.CheckError(t, err, "empty base URL")
}

func TestNewServerEndpoint_NoFinalSlash(t *testing.T) {
	pool := x509.NewCertPool()
	_, rootcert := testkeyutil.GenerateTLSRootForTests(t, "test-root", nil, nil)
	pool.AddCert(rootcert)
	_, err := NewServerEndpoint("https://localhost:50001", pool)
	testutil.CheckError(t, err, "must end in a slash")
}

func TestServerEndpoint_WithHeader(t *testing.T) {
	endpoint := createBaseEndpoint(t, nil)
	if len(endpoint.extraHeaders) != 0 {
		t.Error("Extraneous headers.")
	}
	ep2 := endpoint.WithHeader("X-Test-Header", "test-value")
	if len(endpoint.extraHeaders) != 0 {
		t.Error("Should not have affected previous copy.")
	}
	if len(ep2.extraHeaders) != 1 {
		t.Error("Should have affected new copy.")
	}
	if ep2.extraHeaders["X-Test-Header"] != "test-value" {
		t.Error("Wrong header.")
	}

	ep3 := ep2.WithHeader("X-Other-Header", "test-value-2")
	if len(ep2.extraHeaders) != 1 {
		t.Error("Should not have affected previous copy.")
	}
	if len(ep3.extraHeaders) != 2 {
		t.Error("Should have affected new copy.")
	}
	if ep3.extraHeaders["X-Test-Header"] != "test-value" {
		t.Error("Wrong header.")
	}
	if ep3.extraHeaders["X-Other-Header"] != "test-value-2" {
		t.Error("Wrong header.")
	}
}

func TestServerEndpoint_WithCertificate(t *testing.T) {
	endpoint := createBaseEndpoint(t, nil)
	cakey, cacert := testkeyutil.GenerateTLSRootForTests(t, "ca-root", nil, nil)
	ourkey, ourcert := testkeyutil.GenerateTLSKeypairForTests(t, "cert-test-1", nil, nil, cacert, cakey)
	ourkey2, ourcert2 := testkeyutil.GenerateTLSKeypairForTests(t, "cert-test-2", nil, nil, cacert, cakey)
	if len(endpoint.certificates) != 0 {
		t.Error("Extraneous certs.")
	}

	ep2 := endpoint.WithCertificate(tls.Certificate{PrivateKey: ourkey, Certificate: [][]byte{ourcert.Raw}})
	if len(endpoint.certificates) != 0 {
		t.Error("Should not have affected previous copy.")
	}
	if len(ep2.certificates) != 1 {
		t.Error("Should have affected new copy.")
	}
	if ep2.certificates[0].PrivateKey != ourkey {
		t.Error("Wrong key")
	}
	if len(ep2.certificates[0].Certificate) != 1 {
		t.Error("Wrong cert count")
	}
	if !bytes.Equal(ep2.certificates[0].Certificate[0], ourcert.Raw) {
		t.Error("Wrong cert")
	}

	ep3 := ep2.WithCertificate(tls.Certificate{PrivateKey: ourkey2, Certificate: [][]byte{ourcert2.Raw}})
	if len(ep2.certificates) != 1 {
		t.Error("Should not have affected previous copy.")
	}
	if len(ep3.certificates) != 2 {
		t.Error("Should have affected new copy.")
	}
	if ep3.certificates[0].PrivateKey != ourkey {
		t.Error("Wrong key")
	}
	if len(ep3.certificates[0].Certificate) != 1 {
		t.Error("Wrong cert count")
	}
	if !bytes.Equal(ep3.certificates[0].Certificate[0], ourcert.Raw) {
		t.Error("Wrong cert")
	}
	if ep3.certificates[1].PrivateKey != ourkey2 {
		t.Error("Wrong key")
	}
	if len(ep3.certificates[1].Certificate) != 1 {
		t.Error("Wrong cert count")
	}
	if !bytes.Equal(ep3.certificates[1].Certificate[0], ourcert2.Raw) {
		t.Error("Wrong cert")
	}
}

func TestServerEndpoint_WithBoth(t *testing.T) {
	endpoint := createBaseEndpoint(t, nil)
	cakey, cacert := testkeyutil.GenerateTLSRootForTests(t, "ca-root", nil, nil)
	ourkey, ourcert := testkeyutil.GenerateTLSKeypairForTests(t, "cert-test-1", nil, nil, cacert, cakey)
	ourkey2, ourcert2 := testkeyutil.GenerateTLSKeypairForTests(t, "cert-test-2", nil, nil, cacert, cakey)

	endpoint = endpoint.WithCertificate(tls.Certificate{PrivateKey: ourkey, Certificate: [][]byte{ourcert.Raw}})
	intermediate := endpoint.WithHeader("X-Sample-Header", "ABC")

	endpoint = intermediate.WithCertificate(tls.Certificate{PrivateKey: ourkey2, Certificate: [][]byte{ourcert2.Raw}})
	endpoint = endpoint.WithHeader("X-Sample-Header-2", "DEF")

	if len(intermediate.certificates) != 1 {
		t.Error("Wrong intermediate value")
	}
	if intermediate.certificates[0].PrivateKey != ourkey {
		t.Error("Wrong key")
	}
	if len(intermediate.certificates[0].Certificate) != 1 {
		t.Error("Wrong cert count")
	}
	if !bytes.Equal(intermediate.certificates[0].Certificate[0], ourcert.Raw) {
		t.Error("Wrong cert")
	}

	if len(intermediate.extraHeaders) != 1 {
		t.Error("Wrong intermediate value")
	}
	if intermediate.extraHeaders["X-Sample-Header"] != "ABC" {
		t.Error("Wrong header.")
	}

	if len(endpoint.certificates) != 2 {
		t.Error("Wrong final value")
	}
	if endpoint.certificates[0].PrivateKey != ourkey {
		t.Error("Wrong key")
	}
	if endpoint.certificates[1].PrivateKey != ourkey2 {
		t.Error("Wrong key")
	}
	if len(endpoint.certificates[0].Certificate) != 1 {
		t.Error("Wrong cert count")
	}
	if len(endpoint.certificates[1].Certificate) != 1 {
		t.Error("Wrong cert count")
	}
	if !bytes.Equal(endpoint.certificates[0].Certificate[0], ourcert.Raw) {
		t.Error("Wrong cert")
	}
	if !bytes.Equal(endpoint.certificates[1].Certificate[0], ourcert2.Raw) {
		t.Error("Wrong cert")
	}

	if len(endpoint.extraHeaders) != 2 {
		t.Error("Wrong intermediate value")
	}
	if endpoint.extraHeaders["X-Sample-Header"] != "ABC" {
		t.Error("Wrong header.")
	}
	if endpoint.extraHeaders["X-Sample-Header-2"] != "DEF" {
		t.Error("Wrong header.")
	}

	if len(endpoint.rootCAs.Subjects()) != 1 {
		t.Error("Wrong subjects")
	}
	if endpoint.baseURL != "https://localhost:50001/test/" {
		t.Error("Wrong URL")
	}
}

func TestServerEndpoint_Request_HeaderAuth(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "GET" {
			http.Error(writer, "Wrong method", 400)
		} else if request.URL.Path != "/test/testabc" {
			http.Error(writer, "Wrong path", 404)
		} else if request.Header.Get("X-Token") == "hello world" && request.Header.Get("X-Other-Header") == "other value" {
			data, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				http.Error(writer, "Internal failure", 400)
			} else if string(data) != "this == a body!\n" {
				http.Error(writer, "Wrong request data", 400)
			} else {
				writer.Write([]byte("this == a response!\n"))
			}
		} else {
			http.Error(writer, "Bad authentication header", 403)
		}
	})
	defer stop()
	endpoint := createBaseEndpoint(t, servercert).WithHeader("X-Token", "hello world").WithHeader("X-Other-Header", "other value")
	result, err := endpoint.Request("/testabc", "GET", []byte("this == a body!\n"))
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "this == a response!\n" {
		t.Error("Wrong result.")
	}
}

func TestServerEndpoint_Request_CertAuth(t *testing.T) {
	stop, clientcakey, clientcacert, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "GET" {
			http.Error(writer, "Wrong method", 400)
		} else if request.URL.Path != "/test/testabc" {
			http.Error(writer, "Wrong path", 404)
		} else if len(request.TLS.VerifiedChains) != 1 {
			t.Error("Wrong cert chain count")
			http.Error(writer, "Bad authentication cert", 403)
		} else if len(request.TLS.VerifiedChains[0]) != 2 {
			t.Error("Wrong cert chain length")
			http.Error(writer, "Bad authentication cert", 403)
		} else if request.TLS.VerifiedChains[0][0].Subject.CommonName != "test-temp-cert" {
			t.Error("Wrong cert name")
			http.Error(writer, "Bad authentication cert", 403)
		} else if request.TLS.VerifiedChains[0][1].Subject.CommonName != "test-ca" {
			t.Error("Wrong CA name")
			http.Error(writer, "Bad authentication cert", 403)
		} else {
			data, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				http.Error(writer, "Internal failure", 400)
			} else if string(data) != "this == a body!\n" {
				http.Error(writer, "Wrong request data", 400)
			} else {
				writer.Write([]byte("this == a response!\n"))
			}
		}
	})
	defer stop()
	clientkey, clientcert := testkeyutil.GenerateTLSKeypairForTests(t, "test-temp-cert", nil, nil, clientcacert, clientcakey)
	endpoint := createBaseEndpoint(t, servercert).WithCertificate(tls.Certificate{PrivateKey: clientkey, Certificate: [][]byte{clientcert.Raw}})
	result, err := endpoint.Request("/testabc", "GET", []byte("this == a body!\n"))
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "this == a response!\n" {
		t.Error("Wrong result.")
	}
}

func TestServerEndpoint_Request_BadPath(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		t.Error("should not have gotten here")
	})
	defer stop()
	endpoint := createBaseEndpoint(t, servercert)
	_, err := endpoint.Request("testabc", "GET", []byte("this == a body!\n"))
	testutil.CheckError(t, err, "path must be absolute")
}

func TestServerEndpoint_Request_BadMethod(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		t.Error("should not have gotten here")
	})
	defer stop()
	endpoint := createBaseEndpoint(t, servercert)
	_, err := endpoint.Request("/testabc", " ", []byte("this == a body!\n"))
	testutil.CheckError(t, err, "invalid method")
}

func TestServerEndpoint_Request_RequestTimeout(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(time.Millisecond)
		writer.Write([]byte("done"))
	})
	defer stop()
	endpoint := createBaseEndpoint(t, servercert)
	endpoint.timeout = time.Nanosecond
	_, err := endpoint.Request("/testabc", "GET", nil)
	testutil.CheckError(t, err, "Timeout exceeded")
}

func TestServerEndpoint_Request_RequestFail(t *testing.T) {
	var ncode int
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "Failed on purpose", ncode)
	})
	defer stop()
	endpoint := createBaseEndpoint(t, servercert)
	for _, code := range []int{500, 501, 400, 401, 403, 404} {
		ncode = code
		_, err := endpoint.Request("/testabc", "GET", nil)
		testutil.CheckError(t, err, fmt.Sprintf("status code: %d", code))
	}
}

// TODO: why does this take so long to run
func TestServerEndpoint_Request_RequestGrindToHalt(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		data := [10000]byte{}
		writer.Write(data[:])
		time.Sleep(time.Millisecond * 51)
		writer.Write([]byte("Second line\n"))
	})
	defer stop()
	endpoint := createBaseEndpoint(t, servercert)
	endpoint.timeout = time.Millisecond * 50
	_, err := endpoint.Request("/testabc", "GET", nil)
	testutil.CheckError(t, err, "Timeout exceeded while reading body")
}

func TestServerEndpoint_Get(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "GET" {
			http.Error(writer, "Wrong method", 400)
		} else if request.URL.Path != "/test/testdef" {
			http.Error(writer, "Wrong path", 404)
		} else if request.Header.Get("X-Token") == "hello world" {
			data, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				http.Error(writer, "Internal failure", 400)
			} else if string(data) != "" {
				http.Error(writer, "Wrong request data", 400)
			} else {
				writer.Write([]byte("this == another response!\n"))
			}
		} else {
			http.Error(writer, "Bad authentication header", 403)
		}
	})
	defer stop()
	endpoint := createBaseEndpoint(t, servercert).WithHeader("X-Token", "hello world")
	result, err := endpoint.Get("/testdef")
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "this == another response!\n" {
		t.Error("Wrong result.")
	}
}

type testRequest struct {
	Part1 string
	Part2 int
}

type testResponse struct {
	Out1 string
	Out2 int
}

func TestServerEndpoint_PostJSON(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != "POST" {
			http.Error(writer, "Wrong method", 400)
		} else if request.URL.Path != "/test/testdef" {
			http.Error(writer, "Wrong path", 404)
		} else if request.Header.Get("X-Token") == "hello world" {
			tr := &testRequest{}
			data, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				http.Error(writer, "Internal failure", 400)
			} else if json.Unmarshal(data, tr) != nil {
				t.Error("Cannot unmarshal")
				http.Error(writer, "Cannot unmarshal", 400)
			} else {
				if tr.Part1 != "demonic" || tr.Part2 != -666 {
					t.Errorf("Wrong data: %v from %s", tr, string(data))
					http.Error(writer, "Wrong request data", 400)
				} else {
					data, err := json.Marshal(testResponse{"hello underworld", 999})
					if err != nil {
						t.Error(err)
					} else {
						writer.Write(data)
					}
				}
			}
		} else {
			http.Error(writer, "Bad authentication header", 403)
		}
	})
	defer stop()
	endpoint := createBaseEndpoint(t, servercert).WithHeader("X-Token", "hello world")
	testdata := testRequest{"demonic", -666}
	testout := &testResponse{}
	err := endpoint.PostJSON("/testdef", testdata, testout)
	if err != nil {
		t.Fatal(err)
	}
	if testout.Out1 != "hello underworld" || testout.Out2 != 999 {
		t.Error("Wrong result.")
	}
}

func TestServerEndpoint_PostJSON_BadRequest(t *testing.T) {
	_, servercert := testkeyutil.GenerateTLSRootForTests(t, "root", nil, nil)
	endpoint := createBaseEndpoint(t, servercert)
	testdata := new(chan error)
	err := endpoint.PostJSON("/testdef", testdata, nil)
	testutil.CheckError(t, err, "json: unsupported type: chan error")
}

func TestServerEndpoint_PostJSON_NoServer(t *testing.T) {
	_, servercert := testkeyutil.GenerateTLSRootForTests(t, "root", nil, nil)
	endpoint := createBaseEndpoint(t, servercert)
	err := endpoint.PostJSON("/testdef", 10, nil)
	testutil.CheckError(t, err, "connection refused")
}

func TestServerEndpoint_PostJSON_BadResponse(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("invalid json\n"))
	})
	defer stop()
	endpoint := createBaseEndpoint(t, servercert)
	testout := &testResponse{}
	err := endpoint.PostJSON("/testdef", 10, testout)
	testutil.CheckError(t, err, "invalid character")
}

func TestServerEndpoint_PostJSON_BadUnmarshal(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("[]"))
	})
	defer stop()
	endpoint := createBaseEndpoint(t, servercert)
	testout := testResponse{}
	err := endpoint.PostJSON("/testdef", 10, testout)
	testutil.CheckError(t, err, "json: Unmarshal(non-pointer ")
}
