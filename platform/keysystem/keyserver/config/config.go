package config

import (
	"errors"
	"io/ioutil"
	"path"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
)

type AuthorityType int

const (
	InvalidAuthorityType AuthorityType = iota
	TLSAuthorityType
	SSHAuthorityType
)

type ConfigAuthority struct {
	Type AuthorityType
	Name string
}

func (t ConfigAuthority) Filenames() (key string, cert string) {
	switch t.Type {
	case TLSAuthorityType:
		return t.Name + ".key", t.Name + ".pem"
	case SSHAuthorityType:
		return t.Name, t.Name + ".pub"
	default:
		panic("invalid authority type in Filenames")
	}
}

func TLSAuthority(name string) ConfigAuthority {
	return ConfigAuthority{Type: TLSAuthorityType, Name: name}
}

func SSHAuthority(name string) ConfigAuthority {
	return ConfigAuthority{Type: SSHAuthorityType, Name: name}
}

func (a *ConfigAuthority) Load(dir string) (authorities.Authority, error) {
	if dir == "" {
		return nil, errors.New("empty directory path")
	}
	keyfile, certfile := a.Filenames()
	keydata, err := ioutil.ReadFile(path.Join(dir, keyfile))
	if err != nil {
		return nil, err
	}
	certdata, err := ioutil.ReadFile(path.Join(dir, certfile))
	if err != nil {
		return nil, err
	}
	switch a.Type {
	case TLSAuthorityType:
		return authorities.LoadTLSAuthority(keydata, certdata)
	case SSHAuthorityType:
		return authorities.LoadSSHAuthority(keydata, certdata)
	default:
		panic("invalid authority type in ConfigAuthority.Load")
	}
}
