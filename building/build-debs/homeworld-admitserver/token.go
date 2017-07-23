package main

import (
	"fmt"
	"time"
	"sync"
	"net/http"
	"encoding/base64"
	"crypto/rand"
)

type tokenData struct {
	hostname      string
	expires       time.Time
	configuration []byte
	admin         string
	claimed       bool
}

type tokenHandler struct {
	mutex    sync.Mutex
	by_token map[string]tokenData
}

const (
	expirationInterval = time.Minute * 20       // tokens last twenty minutes before expiration
)

// returns true -> claim this tokenData, false -> don't claim this tokenData
type authedHandlerFunc func(data tokenData, writer http.ResponseWriter, request *http.Request) bool

func (handler *tokenHandler) VerifyAndClaim(token string) (tokenData, error) {
	handler.mutex.Lock()
	defer handler.mutex.Unlock()
	tokdata, present := handler.by_token[token]
	if !present {
		return tokdata, fmt.Errorf("Unrecognized token")
	}
	if tokdata.expires.Before(time.Now()) {
		return tokdata, fmt.Errorf("Token expired at %v", tokdata.expires)
	}
	if tokdata.claimed {
		return tokdata, fmt.Errorf("Token has already been claimed")
	}
	tokdata.claimed = true
	handler.by_token[token] = tokdata
	return tokdata, nil
}

func (handler *tokenHandler) Unclaim(token string) {
	handler.mutex.Lock()
	defer handler.mutex.Unlock()
	tokdata, present := handler.by_token[token]
	if present {
		tokdata.claimed = false
		handler.by_token[token] = tokdata
	}
}

func generateToken() string {
	out := make([]byte, 16)
	_, err := rand.Read(out)
	if err != nil {
		panic(err)
	}
	return base64.RawStdEncoding.EncodeToString(out)
}

func (handler *tokenHandler) populateNewToken(newValue tokenData) string {
	handler.mutex.Lock()
	defer handler.mutex.Unlock()

	for {
		token := generateToken()
		_, already_exists := handler.by_token[token]
		if !already_exists {
			handler.by_token[token] = newValue
			return token
		}
	}
}

func (handler *tokenHandler) ExpireOldEntries() {
	handler.mutex.Lock()
	defer handler.mutex.Unlock()

	for k, v := range handler.by_token {
		if v.expires.Before(time.Now()) {
			delete(handler.by_token, k)
		}
	}
}

func (handler *tokenHandler) GrantToken(hostname string, configuration []byte, admin string) string {
	handler.ExpireOldEntries()

	return handler.populateNewToken(tokenData{
		claimed: false,
		expires: time.Now().Add(expirationInterval),
		hostname: hostname,
		configuration: configuration,
		admin: admin,
	})
}

func (handler *tokenHandler) wrapHandler(handlerFunc authedHandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		// TODO: verify requester IP address matches hostname
		token := request.Header.Get("X-Bootstrap-Token")
		if token == "" {
			http.Error(writer, "No authentication tokenData", http.StatusForbidden)
			return
		}
		tdata, err := handler.VerifyAndClaim(token)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusForbidden)
			return
		}
		should_claim := handlerFunc(tdata, writer, request)
		if !should_claim {
			handler.Unclaim(token)
		}
	}
}

func NewTokenHandler() *tokenHandler {
	return &tokenHandler{by_token: make(map[string]tokenData)}
}
