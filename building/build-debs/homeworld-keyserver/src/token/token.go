package token

import (
	"fmt"
	"time"
	"sync"
	"net/http"
	"encoding/base64"
	"crypto/rand"
	"net"
	"strings"
	"errors"
)

type tokenData struct {
	principal     string
	expires       time.Time
	claimed       bool
	sourceIP      net.IP
}

type TokenRegistry struct {
	mutex    sync.Mutex
	by_token map[string]tokenData
}

func (r *TokenRegistry) verifyToken(token string) (tokenData, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	tokdata, present := r.by_token[token]
	if !present {
		return tokenData{}, fmt.Errorf("Unrecognized token")
	}
	if tokdata.expires.Before(time.Now()) {
		return tokenData{}, fmt.Errorf("Token expired at %v", tokdata.expires)
	}
	if tokdata.claimed {
		return tokenData{}, fmt.Errorf("Token has already been claimed")
	}
	tokdata.claimed = true
	r.by_token[token] = tokdata
	return tokdata, nil
}

func generateToken() string {
	out := make([]byte, 16)
	_, err := rand.Read(out)
	if err != nil {
		panic(err)
	}
	return base64.RawStdEncoding.EncodeToString(out)
}

func (r *TokenRegistry) populateNewToken(newValue tokenData) string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for {
		token := generateToken()
		_, already_exists := r.by_token[token]
		if !already_exists {
			r.by_token[token] = newValue
			return token
		}
	}
}

func (r *TokenRegistry) expireOldEntries() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for k, v := range r.by_token {
		if v.expires.Before(time.Now()) {
			delete(r.by_token, k)
		}
	}
}

func (r *TokenRegistry) GrantToken(principal string, lifespan time.Duration) string {
	r.expireOldEntries()

	return r.populateNewToken(tokenData{
		claimed:   false,
		expires:   time.Now().Add(lifespan),
		principal: principal,
	})
}

// returns IP address
func parseRemoteAddr(remote_addr string) (net.IP, error) {
	parts := strings.Split(remote_addr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid request address (colon count mismatch of %d)", len(parts) - 1)
	}
	ip := net.ParseIP(parts[0])
	if ip == nil {
		return nil, fmt.Errorf("Invalid request address (invalid IP of '%s')", parts[0])
	}
	return ip, nil
}

type noAuthToken struct {
}

func (a noAuthToken) Error() string {
	return "No authentication token provided."
}

func IsNoTokenError(e error) bool {
	_, match := e.(noAuthToken)
	return match
}

func (r *TokenRegistry) Verify(request *http.Request) (string, error) {
	// TODO: Verify requester IP address matches hostname
	token := request.Header.Get("X-Bootstrap-Token")
	if token == "" {
		return "", noAuthToken{}
	}
	tdata, err := r.verifyToken(token)
	if err != nil {
		return "", err
	}
	ip, err := parseRemoteAddr(request.RemoteAddr)
	if err != nil {
		return "", err
	}
	if tdata.sourceIP.Equal(ip) {
		return "", fmt.Errorf("IP address mismatch on client: expected %s but got %s: rejecting request", tdata.sourceIP, ip)
	}
	return tdata.principal, nil
}

func NewTokenRegistry() *TokenRegistry {
	return &TokenRegistry{by_token: make(map[string]tokenData)}
}
