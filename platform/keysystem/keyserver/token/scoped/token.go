package scoped

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sync"
	"time"
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
		return errors.New("cannot claim expired token")
	}
	has_claimed := false
	t.claimed.Do(func() { has_claimed = true })
	if !has_claimed {
		return errors.New("token already claimed")
	}
	return nil
}

func generateTokenID() string {
	out := make([]byte, 15)
	_, err := rand.Read(out)
	if err != nil {
		panic(err)
	}

	hash := base64.RawStdEncoding.EncodeToString(out)
	hashSha256 := sha256.Sum256([]byte(hash))
	return hash + base64.RawStdEncoding.EncodeToString(hashSha256[:])[0:2]
}

func GenerateToken(subject string, duration time.Duration) ScopedToken {
	return ScopedToken{generateTokenID(), subject, time.Now().Add(duration), &sync.Once{}}
}
