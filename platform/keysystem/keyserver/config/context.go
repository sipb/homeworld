package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"keysystem/keyserver/account"
	"keysystem/keyserver/authorities"
	"keysystem/keyserver/verifier"
	"path"
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

type SetupStep func(*Context, *Config) error

func (config *Config) Compile() (*Context, error) {
	context := &Context{TokenVerifier: verifier.NewTokenVerifier()}
	for _, step := range setup_steps {
		err := step(context, config)
		if err != nil {
			return nil, err
		}
	}
	return context, nil
}

func (a *ConfigAuthority) Load(dir string) (authorities.Authority, error) {
	if dir == "" {
		return nil, errors.New("Empty directory path.")
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
		return nil, fmt.Errorf("Cannot find account for principal %s.", principal)
	}
	if ac.Principal != principal {
		return nil, fmt.Errorf("Mismatched principal during lookup")
	}
	return ac, nil
}

func (c *Context) GetAuthority(name string) (authorities.Authority, error) {
	authority, found := c.Authorities[name]
	if found {
		return authority, nil
	} else {
		return nil, fmt.Errorf("No such authority: '%s'", name)
	}
}

func (c *Context) GetTLSAuthority(name string) (*authorities.TLSAuthority, error) {
	authority_any, err := c.GetAuthority(name)
	if err != nil {
		return nil, err
	}
	authority, ok := authority_any.(*authorities.TLSAuthority)
	if !ok {
		return nil, fmt.Errorf("Authority is not a TLS authority: %s", name)
	}
	return authority, nil
}
