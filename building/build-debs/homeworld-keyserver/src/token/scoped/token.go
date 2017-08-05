package scoped

import (
	"time"
	"encoding/base64"
	"crypto/rand"
	"errors"
	"sync"
)

type ScopedToken struct {
	Token   string
	Subject string
	expires time.Time
	claimed *sync.Once
}

func (t ScopedToken) HasExpired() bool {
	return time.Now().After(t.expires)
}

func (t ScopedToken) Claim() error {
	if t.HasExpired() {
		return errors.New("Cannot claim expired token")
	}
	has_claimed := false
	t.claimed.Do(func() { has_claimed = true })
	if !has_claimed {
		return errors.New("Token already claimed")
	}
	return nil
}

func generateTokenID() string {
	out := make([]byte, 15)
	_, err := rand.Read(out)
	if err != nil {
		panic(err)
	}
	return base64.RawStdEncoding.EncodeToString(out)
}

func GenerateToken(subject string, duration time.Duration) ScopedToken {
	return ScopedToken{generateTokenID(), subject, time.Now().Add(duration), &sync.Once{}}
}
