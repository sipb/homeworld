package auth

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"token"
	"time"
	"strings"
)

func getTestRequest(token string) *http.Request {
	req := httptest.NewRequest("GET", "/test", nil)
	if token != "" {
		req.Header.Set("X-Bootstrap-Token", token)
	}
	return req
}

func TestCheckHasTokenHeader(t *testing.T) {
	if HasTokenAuthHeader(getTestRequest("")) {
		t.Error("Should not have header.")
	}
}

func TestCheckHasNoTokenHeader(t *testing.T) {
	if !HasTokenAuthHeader(getTestRequest("header")) {
		t.Error("Should have header.")
	}
}

func TestSimpleAuthenticate(t *testing.T) {
	registry := token.NewTokenRegistry()
	tok := registry.GrantToken("my-local-subject", time.Minute)
	subject, err := Authenticate(registry, getTestRequest(tok))
	if err != nil {
		t.Errorf("Expected token to work: %s", err)
	} else if subject != "my-local-subject" {
		t.Errorf("Incorrect subject retrieved: %s", subject)
	}
}

func TestFailAuthenticate(t *testing.T) {
	registry := token.NewTokenRegistry()
	_ = registry.GrantToken("my-local-subject", time.Minute)
	subject, err := Authenticate(registry, getTestRequest(""))
	if err == nil {
		t.Errorf("Expected token to fail: %s", subject)
	} else if !strings.Contains(err.Error(), "header") {
		t.Errorf("Error lacking mention of missing header: %s", err)
	}
}

func TestLimitedAuthentication(t *testing.T) {
	registry := token.NewTokenRegistry()
	tok := registry.GrantToken("my-local-subject", time.Minute)
	subject, err := Authenticate(registry, getTestRequest(tok))
	if err != nil {
		t.Errorf("Expected token to work: %s", err)
	} else if subject != "my-local-subject" {
		t.Errorf("Incorrect subject retrieved: %s", subject)
	}
	_, err = Authenticate(registry, getTestRequest(tok))
	if err == nil {
		t.Error("Expected token to fail")
	} else if !strings.Contains(err.Error(), "already") {
		t.Errorf("Expected error to mention already being claimed: %s", err.Error())
	}
}

func TestIndependentAuthentication(t *testing.T) {
	registry := token.NewTokenRegistry()
	tok1 := registry.GrantToken("my-local-subject", time.Minute)
	tok2 := registry.GrantToken("my-local-subject-2", time.Minute)
	subject1, err := Authenticate(registry, getTestRequest(tok1))
	if err != nil {
		t.Errorf("Expected token to work: %s", err)
	} else if subject1 != "my-local-subject" {
		t.Errorf("Incorrect subject retrieved: %s", subject1)
	}
	subject2, err := Authenticate(registry, getTestRequest(tok2))
	if err != nil {
		t.Errorf("Expected token to work: %s", err)
	} else if subject2 != "my-local-subject-2" {
		t.Errorf("Incorrect subject retrieved: %s", subject2)
	}
}

func TestExpiredAuthentication(t *testing.T) {
	registry := token.NewTokenRegistry()
	tok := registry.GrantToken("my-local-subject", time.Nanosecond)
	time.Sleep(time.Nanosecond)
	_, err := Authenticate(registry, getTestRequest(tok))
	if err == nil {
		t.Error("Expected token to fail")
	} else if !strings.Contains(err.Error(), "expired") {
		t.Errorf("Expected error to mention already being expired: %s", err.Error())
	}
}

func TestInvalidToken(t *testing.T) {
	fake_token := token.NewTokenRegistry().GrantToken("my-local-subject", time.Minute)
	registry := token.NewTokenRegistry()
	_ = registry.GrantToken("my-local-subject", time.Minute)
	_, err := Authenticate(registry, getTestRequest(fake_token))
	if err == nil {
		t.Error("Expected token to fail")
	} else if !strings.Contains(err.Error(), "Unrecognized") {
		t.Errorf("Expected error to mention already being recognized: %s", err.Error())
	}
}
