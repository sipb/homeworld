package config

import (
	"errors"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
	"io/ioutil"
	"path"
)

type ConfigAuthority struct {
	IsTLS bool
	Name  string
}

func (t ConfigAuthority) Filenames() (key string, cert string) {
	if t.IsTLS {
		return t.Name + ".key", t.Name + ".pem"
	} else {
		return t.Name, t.Name + ".pub"
	}
}

func TLSAuthority(name string) ConfigAuthority {
	return ConfigAuthority{IsTLS: true, Name: name}
}

func SSHAuthority(name string) ConfigAuthority {
	return ConfigAuthority{IsTLS: false, Name: name}
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
	if a.IsTLS {
		return authorities.LoadTLSAuthority(keydata, certdata)
	} else {
		return authorities.LoadSSHAuthority(keydata, certdata)
	}
}
