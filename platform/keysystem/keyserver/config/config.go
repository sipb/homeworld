package config

import (
	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
)

type ConfigAuthority struct {
	Type      string
	Key       string
	Cert      string
	PresentAs []string
}

type ConfigGrant struct {
	Group      *account.Group
	Specialize func(*account.Account, *Context) account.Privilege
}
