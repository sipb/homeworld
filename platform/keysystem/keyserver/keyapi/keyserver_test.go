package keyapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/config"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/verifier"
	"github.com/sipb/homeworld/platform/util/testkeyutil"
)

func TestLoadConfiguredKeyserver(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks, err := LoadConfiguredKeyserver("../config/testdir/smalltest.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	out, err := ks.(*ConfiguredKeyserver).Context.Grants["test-1"].PrivilegeByAccount["my-admin"](nil, "")
	if err != nil {
		t.Error(err)
	}
	if out != "this is a test!" {
		t.Error("Not loaded properly.")
	}
}

func TestLoadConfiguredKeyserver_Nonexistent(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	_, err := LoadConfiguredKeyserver("../config/testdir/nonexistent.yaml", logger)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "no such file or directory") {
		t.Error("Wrong error.")
	}
}

func TestAPIToHTTP_Static(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks, err := LoadConfiguredKeyserver("../config/testdir/smalltest.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/static/testa.txt", nil)
	handler := apiToHTTP(ks, logger)
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	if response.StatusCode != 200 {
		t.Error("Wrong status code.")
	} else {
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		ref, err := ioutil.ReadFile("../config/testdir/testa.txt")
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(ref) {
			t.Error("Static file content mismatch.")
		}
	}
	if logrecord.String() != "" {
		t.Error("Wrong logs.")
	}
}

func TestAPIToHTTP_Static_Fail(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks, err := LoadConfiguredKeyserver("../config/testdir/smalltest.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/static/nonexistent.txt", nil)
	handler := apiToHTTP(ks, logger)
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	if response.StatusCode != 404 {
		t.Errorf("Wrong status code %d.", response.StatusCode)
	} else {
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "Request processing failed: No such static file nonexistent.txt\n" {
			t.Errorf("Wrong error body: %s", string(data))
		}
	}
	if logrecord.String() != "Static request failed with error: No such static file nonexistent.txt\n" {
		t.Errorf("Wrong logs: %s", logrecord.String())
	}
}

func TestAPIToHTTP_Pub(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks, err := LoadConfiguredKeyserver("../config/testdir/smalltest.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/pub/granting", nil)
	handler := apiToHTTP(ks, logger)
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	if response.StatusCode != 200 {
		t.Error("Wrong status code.")
	} else {
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		ref, err := ioutil.ReadFile("../config/testdir/test1.pem")
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(ref) {
			t.Error("Public key content mismatch.")
		}
	}
	if logrecord.String() != "" {
		t.Error("Wrong logs.")
	}
}

func TestAPIToHTTP_Pub_Fail(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks, err := LoadConfiguredKeyserver("../config/testdir/smalltest.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/pub/nonexistent", nil)
	handler := apiToHTTP(ks, logger)
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	if response.StatusCode != 404 {
		t.Errorf("Wrong status code %d.", response.StatusCode)
	} else {
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "Request processing failed: No such authority nonexistent\n" {
			t.Errorf("Wrong error body: %s", string(data))
		}
	}
	if logrecord.String() != "Public key request failed with error: No such authority nonexistent\n" {
		t.Errorf("Wrong logs: %s", logrecord.String())
	}
}

func TestAPIToHTTP_API(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks, err := LoadConfiguredKeyserver("../config/testdir/smalltest2.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request_data := "[{\"api\": \"test-1\", \"body\": \"\"}]"
	request := httptest.NewRequest("GET", "/apirequest", bytes.NewReader([]byte(request_data)))
	request.Header.Set(verifier.TokenHeader, ks.(*ConfiguredKeyserver).Context.TokenVerifier.Registry.GrantToken("my-admin", time.Minute))
	handler := apiToHTTP(ks, logger)
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	if response.StatusCode != 200 {
		t.Error("Wrong status code.")
	} else {
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "[\"after this test, there will be cake\"]" {
			t.Errorf("Result mismatch: %s", string(data))
		}
	}
	if logrecord.String() != "attempting to perform API operation test-1 for my-admin\noperation test-1 for my-admin succeeded\n" {
		t.Errorf("Wrong logs: %s", logrecord.String())
	}
}

func TestAPIToHTTP_API_Fail(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks, err := LoadConfiguredKeyserver("../config/testdir/smalltest2.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request_data := "[{\"api\": \"nonexistent\", \"body\": \"\"}]"
	request := httptest.NewRequest("GET", "/apirequest", bytes.NewReader([]byte(request_data)))
	request.Header.Set(verifier.TokenHeader, ks.(*ConfiguredKeyserver).Context.TokenVerifier.Registry.GrantToken("my-admin", time.Minute))
	handler := apiToHTTP(ks, logger)
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	if response.StatusCode != 400 {
		t.Errorf("Wrong status code %d.", response.StatusCode)
	} else {
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "Request processing failed. See server logs for details.\n" {
			t.Errorf("Wrong error body: %s", string(data))
		}
	}
	if logrecord.String() != "API request failed with error: could not find API request 'nonexistent'\n" {
		t.Errorf("Wrong logs: %s", logrecord.String())
	}
}

func TestAPIToHTTP_Invalid(t *testing.T) {
	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	ks, err := LoadConfiguredKeyserver("../config/testdir/smalltest2.yaml", logger)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request_data := "[{\"api\": \"test-1\", \"body\": \"\"}]"
	request := httptest.NewRequest("GET", "/nonexistent", bytes.NewReader([]byte(request_data)))
	request.Header.Set(verifier.TokenHeader, ks.(*ConfiguredKeyserver).Context.TokenVerifier.Registry.GrantToken("my-admin", time.Minute))
	handler := apiToHTTP(ks, logger)
	handler.ServeHTTP(recorder, request)
	response := recorder.Result()
	if response.StatusCode != 404 {
		t.Error("Expected failing status code.")
	} else {
		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Error(err)
		} else if string(data) != "404 page not found\n" {
			t.Errorf("Found: %s", data)
		}
	}
}

func TestRun_Static(t *testing.T) {
	tlskey, _, tlscert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-selfsig", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)})
	err := ioutil.WriteFile("../config/testdir/selfsig.key", tlskey, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/selfsig.key")
	err = ioutil.WriteFile("../config/testdir/selfsig.pem", tlscert, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/selfsig.pem")

	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	halt, onend, err := Run("../config/testdir/smalltest3.yaml", "127.0.0.1:51234", logger)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		halt()
		err := <-onend
		if err != http.ErrServerClosed {
			t.Error(err)
		}
	}()

	serverCertAsCA, err := authorities.LoadTLSAuthority(tlskey, tlscert)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: serverCertAsCA.(*authorities.TLSAuthority).ToCertPool(),
		},
	}}

	response, err := client.Get("https://127.0.0.1:51234/static/testa.txt")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Errorf("Wrong status code: %d", response.StatusCode)
	} else {
		content, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		ref, err := ioutil.ReadFile("../config/testdir/testa.txt")
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != string(ref) {
			t.Error("Content mismatch.")
		}
	}
}

func TestRun_Pub(t *testing.T) {
	tlskey, _, tlscert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-selfsig", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)})
	err := ioutil.WriteFile("../config/testdir/selfsig.key", tlskey, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/selfsig.key")
	err = ioutil.WriteFile("../config/testdir/selfsig.pem", tlscert, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/selfsig.pem")

	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	halt, onend, err := Run("../config/testdir/smalltest3.yaml", "127.0.0.1:51234", logger)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		halt()
		err := <-onend
		if err != http.ErrServerClosed {
			t.Error(err)
		}
	}()

	serverCertAsCA, err := (&config.ConfigAuthority{Type: "TLS", Key: "selfsig.key", Cert: "selfsig.pem"}).Load("../config/testdir")
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: serverCertAsCA.(*authorities.TLSAuthority).ToCertPool(),
		},
	}}

	response, err := client.Get("https://127.0.0.1:51234/pub/granting")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Errorf("Wrong status code: %d", response.StatusCode)
	} else {
		content, err := ioutil.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		ref, err := ioutil.ReadFile("../config/testdir/selfsig.pem")
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != string(ref) {
			t.Error("Pubkey mismatch.")
		}
	}
}

func TestRun_API(t *testing.T) {
	tlskey, _, tlscert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-selfsig", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)})
	err := ioutil.WriteFile("../config/testdir/selfsig.key", tlskey, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/selfsig.key")
	err = ioutil.WriteFile("../config/testdir/selfsig.pem", tlscert, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/selfsig.pem")

	clikey, _, clicert := testkeyutil.GenerateTLSKeypairPEMsForTests(t, "my-admin", nil, nil, tlscert, tlskey)
	err = ioutil.WriteFile("../config/testdir/client-of-selfsig.key", clikey, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/client-of-selfsig.key")
	err = ioutil.WriteFile("../config/testdir/client-of-selfsig.pem", clicert, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/client-of-selfsig.pem")

	logrecord := bytes.NewBuffer(nil)
	logger := log.New(logrecord, "", 0)
	halt, onend, err := Run("../config/testdir/smalltest3.yaml", "127.0.0.1:51234", logger)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		halt()
		err := <-onend
		if err != http.ErrServerClosed {
			t.Error(err)
		}
	}()

	serverCertAsCA, err := (&config.ConfigAuthority{Type: "TLS", Key: "selfsig.key", Cert: "selfsig.pem"}).Load("../config/testdir")
	if err != nil {
		t.Fatal(err)
	}

	keypair, err := tls.LoadX509KeyPair("../config/testdir/client-of-selfsig.pem", "../config/testdir/client-of-selfsig.key")
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{keypair},
			RootCAs:      serverCertAsCA.(*authorities.TLSAuthority).ToCertPool(),
		},
	}}

	json_data := []byte("[{\"api\": \"test-2\", \"body\": \"my-server\"}]")

	response, err := client.Post("https://127.0.0.1:51234/apirequest", "application/json", bytes.NewReader(json_data))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("Wrong status code: %d", response.StatusCode)
	}
	tokenresponse, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	results := []string{}
	err = json.Unmarshal(tokenresponse, &results)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatal("Incorrect result count")
	}
	token := results[0]

	client = &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: serverCertAsCA.(*authorities.TLSAuthority).ToCertPool(),
		},
	}}

	json_data = []byte("[{\"api\": \"test-1\", \"body\": \"\"}]")

	request, err := http.NewRequest("POST", "https://127.0.0.1:51234/apirequest", bytes.NewReader(json_data))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Bootstrap-Token", string(token))
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("Wrong status code: %d", response.StatusCode)
	}
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != "[\"and now, there will be cake\"]" {
		t.Error("Mismatch result.")
	}
	if logrecord.String() != "attempting to perform API operation test-2 for my-admin\noperation test-2 for my-admin succeeded\nattempting to perform API operation test-1 for my-server\noperation test-1 for my-server succeeded\n" {
		t.Error("Wrong logs.")
	}
}

func TestRun_NoConfFile(t *testing.T) {
	_, _, err := Run("../config/testdir/nonexistent.yaml", ":12345", nil)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "no such file or directory") {
		t.Error("Wrong error.")
	}
}

func TestRun_BadAddress(t *testing.T) {
	tlskey, _, tlscert := testkeyutil.GenerateTLSRootPEMsForTests(t, "test-selfsig", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)})
	err := ioutil.WriteFile("../config/testdir/selfsig.key", tlskey, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/selfsig.key")
	err = ioutil.WriteFile("../config/testdir/selfsig.pem", tlscert, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/selfsig.pem")

	clikey, _, clicert := testkeyutil.GenerateTLSKeypairPEMsForTests(t, "my-admin", nil, nil, tlscert, tlskey)
	err = ioutil.WriteFile("../config/testdir/client-of-selfsig.key", clikey, os.FileMode(0600))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/client-of-selfsig.key")
	err = ioutil.WriteFile("../config/testdir/client-of-selfsig.pem", clicert, os.FileMode(0644))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("../config/testdir/client-of-selfsig.pem")

	_, _, err = Run("../config/testdir/smalltest3.yaml", "8.8.8.8:1", nil)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "cannot assign requested address") {
		t.Errorf("Wrong error: %s", err.Error())
	}
}
