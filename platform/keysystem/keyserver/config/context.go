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

type Grant struct {
	API                string
	Group              *account.Group
	PrivilegeByAccount map[string]account.Privilege
}

type Context struct {
	Authorities             map[string]authorities.Authority
	Groups                  map[string]*account.Group
	Grants                  map[string]Grant
	Accounts                map[string]*account.Account
	TokenVerifier           verifier.TokenVerifier
	AuthenticationAuthority *authorities.TLSAuthority
	ServerTLS               *authorities.TLSAuthority
	StaticFiles             map[string]StaticFile
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

func (c *Context) GetAuthority(name string) (authorities.Authority, error) {
	authority, found := c.Authorities[name]
	if found {
		return authority, nil
	} else {
		return nil, fmt.Errorf("no such authority: '%s'", name)
	}
}

func (c *Context) GetTLSAuthority(name string) (*authorities.TLSAuthority, error) {
	authorityAny, err := c.GetAuthority(name)
	if err != nil {
		return nil, err
	}
	authority, ok := authorityAny.(*authorities.TLSAuthority)
	if !ok {
		return nil, fmt.Errorf("authority is not a TLS authority: %s", name)
	}
	return authority, nil
}
