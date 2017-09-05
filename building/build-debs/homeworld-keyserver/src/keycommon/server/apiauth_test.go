package server

import (
	"testing"
	"net/http"
	"encoding/pem"
	"testutil"
	"crypto/tls"
	"keycommon/reqtarget"
	"io/ioutil"
)

func TestKeyserver_AuthenticateWithCert(t *testing.T) {
	stop, cakey, cacert, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
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
				if string(data) != "[{\"api\":\"testapi\",\"body\":\"testbody\"}]" {
					t.Error("Wrong data")
					http.Error(writer, "Wrong request", 400)
				} else {
					writer.Write([]byte("[\"testresponse\"]"))
				}
			}
		}
	})
	defer stop()
	ks, err := NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	clikey, clicert := testutil.GenerateTLSKeypairForTests(t, "test-client", nil, nil, cacert, cakey)
	rt, err := ks.AuthenticateWithCert(tls.Certificate{PrivateKey: clikey, Certificate: [][]byte{ clicert.Raw }})
	if err != nil {
		t.Fatal(err)
	}
	response, err := rt.SendRequests([]reqtarget.Request{ {API: "testapi", Body: "testbody"} })
	if err != nil {
		t.Fatal(err)
	}
	if len(response) != 1 {
		t.Error("Wrong response count.")
	}
	if response[0] != "testresponse" {
		t.Error("Wrong response.")
	}
}

func TestKeyserver_AuthenticateWithToken(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/apirequest" {
			http.Error(writer, "Wrong path", 404)
		} else if request.Header.Get("X-Bootstrap-Token") != "mytoken" {
			http.Error(writer, "Invalid token", 403)
		} else {
			data, err := ioutil.ReadAll(request.Body)
			if err != nil {
				t.Error(err)
				http.Error(writer, "Cannot read", 400)
			} else {
				if string(data) != "[{\"api\":\"testapi\",\"body\":\"testbody\"}]" {
					t.Error("Wrong data")
					http.Error(writer, "Wrong request", 400)
				} else {
					writer.Write([]byte("[\"testresponse\"]"))
				}
			}
		}
	})
	defer stop()
	ks, err := NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	rt, err := ks.AuthenticateWithToken("mytoken")
	if err != nil {
		t.Fatal(err)
	}
	response, err := rt.SendRequests([]reqtarget.Request{ {API: "testapi", Body: "testbody"} })
	if err != nil {
		t.Fatal(err)
	}
	if len(response) != 1 {
		t.Error("Wrong response count.")
	}
	if response[0] != "testresponse" {
		t.Error("Wrong response.")
	}
}

func TestKeyserver_AuthenticateWithToken_NoToken(t *testing.T) {
	_, servercert := testutil.GenerateTLSRootForTests(t, "test-serv", nil, nil)
	ks, err := NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	_, err = ks.AuthenticateWithToken("")
	testutil.CheckError(t, err, "Invalid token.")
}

func TestKeyserver_AuthenticateWithToken_NoServer(t *testing.T) {
	_, servercert := testutil.GenerateTLSRootForTests(t, "test-serv", nil, nil)
	ks, err := NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	rt, err := ks.AuthenticateWithToken("test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.SendRequests(nil)
	testutil.CheckError(t, err, "connection refused")
}

func TestKeyserver_AuthenticateWithToken_ResponseMismatch(t *testing.T) {
	stop, _, _, servercert := launchTestServer(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("[\"123\"]"))
	})
	defer stop()
	ks, err := NewKeyserver(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servercert.Raw}), "localhost")
	if err != nil {
		t.Fatal(err)
	}
	rt, err := ks.AuthenticateWithToken("test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.SendRequests(nil)
	testutil.CheckError(t, err, "wrong number of responses")
}
