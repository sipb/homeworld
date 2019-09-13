package config

import (
	"fmt"
	"github.com/pkg/errors"
	"net"
	"os"
	"path"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
)

var setupSteps = []SetupStep{CompileStaticFiles, CompileAuthorities, CompileGlobalAuthorities, CompileGroups, CompileAccounts, CompileGrants}

func CompileStaticFiles(context *Context, config *Config) error {
	context.StaticFiles = make(map[string]StaticFile)
	for _, file := range config.StaticFiles {
		fullpath := path.Join(config.StaticDir, string(file))
		// check for existence
		openfile, err := os.Open(fullpath)
		if err != nil {
			return err
		}
		openfile.Close()
		context.StaticFiles[string(file)] = StaticFile{string(file), fullpath}
	}
	return nil
}

func CompileAuthorities(context *Context, config *Config) error {
	context.Authorities = make(map[string]authorities.Authority)
	for name, authority := range config.Authorities {
		if name == "" {
			return errors.New("an authority name is required")
		}
		loaded, err := authority.Load(config.AuthorityDir)
		if err != nil {
			return err
		}
		context.Authorities[name] = loaded
	}
	return nil
}

func CompileGlobalAuthorities(context *Context, config *Config) error {
	if config.AuthenticationAuthority == "" || config.ServerTLS == "" {
		return errors.New("expected both authentication-authority and server-tls to be populated fields")
	}

	var err error
	context.AuthenticationAuthority, err = context.GetTLSAuthority(config.AuthenticationAuthority)
	if err != nil {
		return err
	}
	context.ServerTLS, err = context.GetTLSAuthority(config.ServerTLS)
	return err
}

func CompileAccounts(context *Context, config *Config) error {
	context.Accounts = make(map[string]*account.Account)
	for _, ac := range config.Accounts {
		if ac.Principal == "" {
			return errors.New("an account name is required")
		}
		_, found := context.Accounts[ac.Principal]
		if found {
			return fmt.Errorf("duplicate account %s", ac.Principal)
		}
		group := context.Groups[ac.Group]
		if group == nil {
			return fmt.Errorf("no such group %s (in account %s)", ac.Group, ac.Principal)
		}
		var limitIP net.IP
		if ac.LimitIP {
			limitIP = net.ParseIP(ac.Metadata["ip"])
			if limitIP == nil {
				return fmt.Errorf("invalid IP address: %s", ac.Metadata["ip"])
			}
		}
		metadata := map[string]string{"principal": ac.Principal}
		for k, v := range ac.Metadata {
			metadata[k] = v
		}
		context.Accounts[ac.Principal] = &account.Account{ac.Principal, group, ac.DisableDirectAuth, metadata, limitIP}
		for curgroup := group; curgroup != nil; curgroup = curgroup.SubgroupOf {
			for _, existing := range curgroup.AllMembers {
				if existing == ac.Principal {
					return fmt.Errorf("subgroupof cycle detected in config, involving group '%s' and principal '%s'", curgroup.Name, ac.Principal)
				}
			}
			curgroup.AllMembers = append(curgroup.AllMembers, ac.Principal)
		}
	}
	return nil
}

func CompileGroups(context *Context, config *Config) error {
	context.Groups = make(map[string]*account.Group)
	for name, _ := range config.Groups {
		if name == "" {
			return errors.New("a group name is required")
		}
		context.Groups[name] = &account.Group{Name: name}
	}
	for name, group := range config.Groups {
		if group.SubgroupOf != "" {
			subgroupof := context.Groups[group.SubgroupOf]
			if subgroupof == nil {
				return fmt.Errorf("cannot find group %s to be a subgroup of in %s", group.SubgroupOf, name)
			}
			context.Groups[name].SubgroupOf = subgroupof
		}
	}
	return nil
}

func CompileGrants(context *Context, config *Config) error {
	context.Grants = make(map[string]Grant)
	for api, grant := range config.Grants {
		if api == "" {
			return errors.New("an API name is required")
		}
		group, found := context.Groups[grant.Group]
		if !found {
			return fmt.Errorf("could not find group %s for grant %s", grant.Group, api)
		}
		privileges := make(map[string]account.Privilege)
		for _, accountname := range group.AllMembers {
			_, found := privileges[accountname]
			if found {
				return fmt.Errorf("duplicate account %s", accountname)
			}
			ac, found := context.Accounts[accountname]
			if !found {
				return fmt.Errorf("no such account %s", accountname)
			}
			cgrant, err := grant.CompileGrant(ac.Metadata, context)
			if err != nil {
				return err
			}
			priv, err := cgrant.CompileToPrivilege(context)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("in grant %s for account %s", api, accountname))
			}
			privileges[accountname] = priv
		}
		context.Grants[api] = Grant{api, group, privileges}
	}
	return nil
}
