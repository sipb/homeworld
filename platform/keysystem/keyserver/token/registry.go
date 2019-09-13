package token

import (
	"errors"
	"sync"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/token/scoped"
)

type TokenRegistry struct {
	mutex   sync.Mutex
	byToken map[string]scoped.ScopedToken
}

func (r *TokenRegistry) LookupToken(token string) (scoped.ScopedToken, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	tokdata, present := r.byToken[token]
	if !present {
		return scoped.ScopedToken{}, errors.New("unrecognized token")
	}
	return tokdata, nil
}

func (r *TokenRegistry) addToken(token scoped.ScopedToken) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	_, exists := r.byToken[token.Token]
	if exists {
		// It's better to crash than allow cross-contamination of tokens
		panic("Token collision (is that even possible?)")
	}
	r.byToken[token.Token] = token
}

func (r *TokenRegistry) expireOldEntries() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for k, v := range r.byToken {
		if v.HasExpired() {
			delete(r.byToken, k)
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
	return &TokenRegistry{byToken: make(map[string]scoped.ScopedToken)}
}
