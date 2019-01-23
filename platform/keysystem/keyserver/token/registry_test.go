package token

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/token/scoped"
)

func TestNonexistentTokens(t *testing.T) {
	registry := NewTokenRegistry()
	for i := 0; i < 1000; i++ {
		randtoken := scoped.GenerateToken("", time.Second).Token
		_, err := registry.LookupToken(randtoken)
		if err == nil {
			t.Errorf("Expected error when looking up token %s", randtoken)
		} else if !strings.Contains(err.Error(), "Unrecognized") {
			t.Errorf("Expected unrecognized token error")
		}
	}
}

func TestTokenExpiration(t *testing.T) {
	registry := NewTokenRegistry()
	for i := 0; i < 1000; i++ {
		registry.GrantToken("subject", time.Nanosecond)
		if i > 900 {
			time.Sleep(time.Nanosecond)
		}
	}
	if len(registry.by_token) != 1 {
		t.Errorf("Expiration process did not work properly (%d)", len(registry.by_token))
	}
}

func TestTokenNoExpiration(t *testing.T) {
	registry := NewTokenRegistry()
	for i := 0; i < 1000; i++ {
		registry.GrantToken("subject", time.Minute)
	}
	if len(registry.by_token) != 1000 {
		t.Errorf("Non-expiration process did not work properly (%d)", len(registry.by_token))
	}
}

func TestGrantedTokensAreRetrievable(t *testing.T) {
	registry := NewTokenRegistry()
	tokens := [1000]string{}
	for i := 0; i < 1000; i++ {
		tokens[i] = registry.GrantToken(fmt.Sprintf("subject-%d", i), time.Minute)
		if tokens[i] == "" {
			t.Error("Unexpected empty token.")
		}
	}
	for i := 0; i < 1000; i++ {
		sct, err := registry.LookupToken(tokens[i])
		if err != nil {
			t.Error(err)
		} else if sct.Token != tokens[i] {
			t.Error("Retrieved token mismatch.")
		} else if sct.Subject != fmt.Sprintf("subject-%d", i) {
			t.Error("Retrieved subject mismatch.")
		}
	}
}

func TestIndependentClaimsConflict(t *testing.T) {
	registry := NewTokenRegistry()
	tok := registry.GrantToken("test", time.Minute)
	st1, err := registry.LookupToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	st3, err := registry.LookupToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if st1.Claim() != nil {
		t.Error("Couldn't claim first value")
	}
	st2, err := registry.LookupToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if st2.Claim() == nil {
		t.Error("Shouldn't be able to claim second value")
	}
	if st3.Claim() == nil {
		t.Error("Shouldn't be able to claim third value")
	}
}
