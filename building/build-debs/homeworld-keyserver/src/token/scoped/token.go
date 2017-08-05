package scoped

import (
	"time"
	"encoding/base64"
	"crypto/rand"
	"net"
	"util"
	"errors"
)

type ScopedToken struct {
	Token    string
	Subject  string
	expires  time.Time
	claimed  *util.OnceFlag
	sourceIP net.IP
}

func (t ScopedToken) HasExpired() bool {
	return time.Now().After(t.expires)
}

func (t ScopedToken) Claim(fromip net.IP) error {
	if t.HasExpired() {
		return errors.New("Cannot claim expired token")
	}
	if !t.sourceIP.Equal(fromip) {
		return errors.New("Cannot claim token from incorrect source IP.")
	}
	if !t.claimed.Set() {
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

func GenerateToken(subject string, duration time.Duration, limitIP net.IP) ScopedToken {
	return ScopedToken{generateTokenID(), subject, time.Now().Add(duration), util.NewOnceFlag(), limitIP}
}
