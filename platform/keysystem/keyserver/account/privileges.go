package account

import (
	"errors"
	"fmt"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/token"
)

type OperationContext struct {
	Account *Account
}

type Privilege func(ctx *OperationContext, param string) (string, error)

type TLSSignPrivilege struct {
	authority  *authorities.TLSAuthority
	ishost     bool
	lifespan   time.Duration
	commonname string
	dnsnames   []string
}

func NewTLSGrantPrivilege(tauth *authorities.TLSAuthority, ishost bool, lifespan time.Duration, commonname string, dnsnames []string) Privilege {
	return func(_ *OperationContext, signingRequest string) (string, error) {
		return tauth.Sign(signingRequest, ishost, lifespan, commonname, dnsnames)
	}
}

func NewSSHGrantPrivilege(tauth *authorities.SSHAuthority, ishost bool, lifespan time.Duration, keyid string, principals []string) Privilege {
	return func(_ *OperationContext, signingRequest string) (string, error) {
		return tauth.Sign(signingRequest, ishost, lifespan, keyid, principals)
	}
}

func NewBootstrapPrivilege(allowed *Group, lifespan time.Duration, registry *token.TokenRegistry) Privilege {
	return func(_ *OperationContext, encodedPrincipal string) (string, error) {
		principal := string(encodedPrincipal)
		if !allowed.HasMember(principal) {
			return "", fmt.Errorf("principal not allowed to be bootstrapped: %s", encodedPrincipal)
		}
		generatedToken := registry.GrantToken(principal, lifespan)
		return generatedToken, nil
	}
}

func NewImpersonatePrivilege(getAccount func(string) (*Account, error), scope *Group) Privilege {
	return func(ctx *OperationContext, newPrincipal string) (string, error) {
		if !scope.HasMember(newPrincipal) {
			return "", errors.New("attempt to impersonate outside of allowed scope")
		}
		account, err := getAccount(newPrincipal)
		if err != nil {
			return "", err
		}
		if account.Principal != newPrincipal {
			return "", errors.New("wrong account returned")
		}
		ctx.Account = account
		return "", nil
	}
}

func NewConfigurationPrivilege(contents string) Privilege {
	return func(_ *OperationContext, request string) (string, error) {
		if len(request) != 0 {
			return "", errors.New("expected empty request to configuration endpoint")
		}
		return contents, nil
	}
}

func NewFetchKeyPrivilege(static *authorities.StaticAuthority) Privilege {
	return func(_ *OperationContext, request string) (string, error) {
		if len(request) != 0 {
			return "", errors.New("expected empty request to fetch-key endpoint")
		}
		return string(static.GetPrivateKey()), nil
	}
}
