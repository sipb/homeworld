package config

import (
	"errors"
	"fmt"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/admit"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
)

type StaticFile struct {
	Filepath string
}

type Context struct {
	Authorities             map[string]authorities.Authority
	Accounts                map[string]*account.Account
	AdmitChecker            *admit.AdmitChecker
	AuthenticationAuthority *authorities.TLSAuthority
	ClusterCA               *authorities.TLSAuthority
	StaticFiles             map[string]StaticFile
	KeyserverDNS            string
}

func (ctx *Context) GetAccount(principal string) (*account.Account, error) {
	ac, found := ctx.Accounts[principal]
	if !found {
		return nil, fmt.Errorf("cannot find account for principal %s", principal)
	}
	if ac.Principal != principal {
		return nil, errors.New("mismatched principal during lookup")
	}
	return ac, nil
}
