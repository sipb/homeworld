package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/verifier"
)

type StaticFile struct {
	Filename string
	Filepath string
}

type Context struct {
	Authorities             map[string]authorities.Authority
	Accounts                map[string]*account.Account
	TokenVerifier           verifier.TokenVerifier
	AuthenticationAuthority *authorities.TLSAuthority
	ClusterCA               *authorities.TLSAuthority
	StaticFiles             map[string]StaticFile
	KeyserverDNS            string
}

func (a *ConfigAuthority) Load(dir string) (authorities.Authority, error) {
	if dir == "" {
		return nil, errors.New("empty directory path")
	}
	keydata, err := ioutil.ReadFile(path.Join(dir, a.Key))
	if err != nil {
		return nil, err
	}
	certdata, err := ioutil.ReadFile(path.Join(dir, a.Cert))
	if err != nil {
		return nil, err
	}
	return authorities.LoadAuthority(a.Type, keydata, certdata)
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
