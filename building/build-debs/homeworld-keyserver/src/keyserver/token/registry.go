package token

import (
	"fmt"
	"keyserver/token/scoped"
	"sync"
	"time"
)

type TokenRegistry struct {
	mutex    sync.Mutex
	by_token map[string]scoped.ScopedToken
}

func (r *TokenRegistry) LookupToken(token string) (scoped.ScopedToken, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	tokdata, present := r.by_token[token]
	if !present {
		return scoped.ScopedToken{}, fmt.Errorf("Unrecognized token")
	}
	return tokdata, nil
}

func (r *TokenRegistry) addToken(token scoped.ScopedToken) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, exists := r.by_token[token.Token]
	if exists {
		// It's better to crash than allow cross-contamination of tokens
		panic("Token collision (is that even possible?)")
	}
	r.by_token[token.Token] = token
}

func (r *TokenRegistry) expireOldEntries() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for k, v := range r.by_token {
		if v.HasExpired() {
			delete(r.by_token, k)
		}
	}
}

func (r *TokenRegistry) GrantToken(subject string, lifespan time.Duration) string {
	r.expireOldEntries()
	token := scoped.GenerateToken(subject, lifespan)
	r.addToken(token)
	return token.Token
}

func NewTokenRegistry() *TokenRegistry {
	return &TokenRegistry{by_token: make(map[string]scoped.ScopedToken)}
}
