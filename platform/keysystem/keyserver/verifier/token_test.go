package verifier

import (
	"github.com/sipb/homeworld/platform/keysystem/keyserver/token"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func getTestRequest(token string) *http.Request {
	req := httptest.NewRequest("GET", "/test", nil)
	if token != "" {
		req.Header.Set("X-Bootstrap-Token", token)
	}
	return req
}

func TestCheckHasTokenHeader(t *testing.T) {
	verifier := NewTokenVerifier()
	if verifier.HasAttempt(getTestRequest("")) {
		t.Error("Should not have header.")
	}
}

func TestCheckHasNoTokenHeader(t *testing.T) {
	verifier := NewTokenVerifier()
	if !verifier.HasAttempt(getTestRequest("header")) {
		t.Error("Should have header.")
	}
}

func TestSimpleAuthenticate(t *testing.T) {
	verifier := NewTokenVerifier()
	tok := verifier.Registry.GrantToken("my-local-subject", time.Minute)
	subject, err := verifier.Verify(getTestRequest(tok))
	if err != nil {
		t.Errorf("Expected token to work: %s", err)
	} else if subject != "my-local-subject" {
		t.Errorf("Incorrect subject retrieved: %s", subject)
	}
}

func TestFailAuthenticate(t *testing.T) {
	verifier := NewTokenVerifier()
	_ = verifier.Registry.GrantToken("my-local-subject", time.Minute)
	subject, err := verifier.Verify(getTestRequest(""))
	if err == nil {
		t.Errorf("Expected token to fail: %s", subject)
	} else if !strings.Contains(err.Error(), "header") {
		t.Errorf("Error lacking mention of missing header: %s", err)
	}
}

func TestLimitedAuthentication(t *testing.T) {
	verifier := NewTokenVerifier()
	tok := verifier.Registry.GrantToken("my-local-subject", time.Minute)
	subject, err := verifier.Verify(getTestRequest(tok))
	if err != nil {
		t.Errorf("Expected token to work: %s", err)
	} else if subject != "my-local-subject" {
		t.Errorf("Incorrect subject retrieved: %s", subject)
	}
	_, err = verifier.Verify(getTestRequest(tok))
	if err == nil {
		t.Error("Expected token to fail")
	} else if !strings.Contains(err.Error(), "already") {
		t.Errorf("Expected error to mention already being claimed: %s", err.Error())
	}
}

func TestIndependentAuthentication(t *testing.T) {
	verifier := NewTokenVerifier()
	tok1 := verifier.Registry.GrantToken("my-local-subject", time.Minute)
	tok2 := verifier.Registry.GrantToken("my-local-subject-2", time.Minute)
	subject1, err := verifier.Verify(getTestRequest(tok1))
	if err != nil {
		t.Errorf("Expected token to work: %s", err)
	} else if subject1 != "my-local-subject" {
		t.Errorf("Incorrect subject retrieved: %s", subject1)
	}
	subject2, err := verifier.Verify(getTestRequest(tok2))
	if err != nil {
		t.Errorf("Expected token to work: %s", err)
	} else if subject2 != "my-local-subject-2" {
		t.Errorf("Incorrect subject retrieved: %s", subject2)
	}
}

func TestExpiredAuthentication(t *testing.T) {
	verifier := NewTokenVerifier()
	tok := verifier.Registry.GrantToken("my-local-subject", time.Nanosecond)
	time.Sleep(time.Nanosecond)
	_, err := verifier.Verify(getTestRequest(tok))
	if err == nil {
		t.Error("Expected token to fail")
	} else if !strings.Contains(err.Error(), "expired") {
		t.Errorf("Expected error to mention already being expired: %s", err.Error())
	}
}

func TestInvalidToken(t *testing.T) {
	fake_token := token.NewTokenRegistry().GrantToken("my-local-subject", time.Minute)
	verifier := NewTokenVerifier()
	_ = verifier.Registry.GrantToken("my-local-subject", time.Minute)
	_, err := verifier.Verify(getTestRequest(fake_token))
	if err == nil {
		t.Error("Expected token to fail")
	} else if !strings.Contains(err.Error(), "Unrecognized") {
		t.Errorf("Expected error to mention already being recognized: %s", err.Error())
	}
}
