package scoped

import (
	"testing"
	"unicode"
	"time"
	"strings"
	"math"
)

func TestTokensAreDistinct(t *testing.T) {
	found := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		new_token := generateTokenID()
		if found[new_token] {
			t.Errorf("Duplicate token detected!")
		}
		found[new_token] = true
	}
}

func TestTokensArePrintable(t *testing.T) {
	for i := 0; i < 1000; i++ {
		token := generateTokenID()
		for _, c := range token {
			acceptable := unicode.IsPrint(c) && c != ' ' && c <= 127
			if !acceptable {
				t.Errorf("Unacceptable character found in token (must be printable non-space ASCII)")
			}
		}
	}
}

func TestTokensLength(t *testing.T) {
	for i := 0; i < 1000; i++ {
		token := generateTokenID()
		if len(token) != 20 {
			t.Errorf("Incorrect token length %d (expected %d)", len(token), 22)
		}
	}
}

func TestTokensCharSpread(t *testing.T) {
	found := make(map[int32]int)
	for i := 0; i < 1000; i++ {
		token := generateTokenID()
		for _, c := range token {
			found[c] += 1
		}
	}
	count := 0
	for k, v := range found {
		if v != 0 {
			if v < 50 {
				t.Errorf("Character not found sufficiently frequently: %s (%d)", string(k), v)
			} else if v >= 500 {
				t.Errorf("Character found too frequently: %s (%d)", string(k), v)
			}
			count++
		}
	}
	if count != 64 {
		t.Errorf("Expected exactly 64 different used characters; not %d", count)
	}
}

func TestExpiry(t *testing.T) {
	tok := ScopedToken{expires: time.Unix(0, 0)}
	if !tok.HasExpired() {
		t.Errorf("Unix epoch should have passed by now")
	}
	tok.expires = time.Now()
	if !tok.HasExpired() {
		t.Errorf("Token should immediately expire")
	}
	tok.expires = time.Now().Add(time.Millisecond * 100)
	if tok.HasExpired() {
		t.Errorf("Token shouldn't be expired yet")
	}
	time.Sleep(time.Millisecond * 100)
	if !tok.HasExpired() {
		t.Errorf("Token should have expired by now")
	}
	tok.expires = time.Now().Add(time.Millisecond * 200)
	if tok.HasExpired() {
		t.Errorf("Should not have expired yet")
	}
	time.Sleep(time.Millisecond * 100)
	if tok.HasExpired() {
		t.Errorf("Should not have expired yet")
	}
}

func TestCanClaimTrivialToken(t *testing.T) {
	tok := GenerateToken("", time.Millisecond*500)
	err := tok.Claim()
	if err != nil {
		t.Errorf("Should not have gotten error: %v", err)
	}
}

func TestCannotClaimExpiredToken(t *testing.T) {
	tok := GenerateToken("", time.Millisecond*100)
	time.Sleep(time.Millisecond * 100)
	if !tok.HasExpired() {
		t.Errorf("Token should have expired now, but has not")
	}
	err := tok.Claim()
	if err == nil {
		t.Errorf("Should have failed to claim expired token")
	} else if !strings.Contains(err.Error(), "expired") {
		t.Errorf("Error message '%s' should have contained the word 'expired'", err)
	}
}

func TestCannotClaimTwice(t *testing.T) {
	tok := GenerateToken("", time.Millisecond*500)
	err := tok.Claim()
	if err != nil {
		t.Errorf("Should have claimed token the first time")
	}
	err = tok.Claim()
	if err == nil {
		t.Errorf("Should have failed to claim token the second time")
	} else if !strings.Contains(err.Error(), "already claimed") {
		t.Errorf("Error message '%s' should have contained the words 'already claimed'", err)
	}
}

func TestGenerateToken(t *testing.T) {
	token := GenerateToken("my-subject", time.Minute*90)
	delta := token.expires.Sub(time.Now())
	if math.Abs(delta.Minutes()-90) >= 0.1 {
		t.Errorf("Inaccurately represented expiration time (delta is %s)", delta.Minutes())
	}
	if token.claimed == nil {
		t.Errorf("Inaccurately used nil claim")
	}
	if !token.claimed.Set() {
		t.Errorf("Already set claimed flag")
	}
	if token.Subject != "my-subject" {
		t.Errorf("Inaccurately represented subject name")
	}
	if len(token.Token) != 20 {
		t.Errorf("Used invalid token name in generated token")
	}
}
