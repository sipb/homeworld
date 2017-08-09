package config

import (
	"account"
	"authorities"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
)

var setup_steps = []SetupStep{CompileStaticFiles, CompileAuthorities, CompileGlobalAuthorities, CompileGroups, CompileAccounts, CompileGrants}

func CompileStaticFiles(context *Context, config *Config) error {
	context.StaticFiles = make(map[string]StaticFile)
	for _, file := range config.StaticFiles {
		fullpath := path.Join(config.StaticDir, string(file))
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
			return fmt.Errorf("An authority name is required.")
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
		return errors.New("Expected both authentication-authority and server-tls to be populated fields.")
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
			return fmt.Errorf("An account name is required.")
		}
		_, found := context.Accounts[ac.Principal]
		if found {
			return fmt.Errorf("Duplicate account %s", ac.Principal)
		}
		group := context.Groups[ac.Group]
		if group == nil {
			return fmt.Errorf("No such group %s (in account %s)", ac.Group, ac.Principal)
		}
		var limitIP net.IP
		if ac.LimitIP {
			limitIP = net.ParseIP(ac.Metadata["ip"])
			if limitIP == nil {
				return fmt.Errorf("Invalid IP address: %s", ac.Metadata["ip"])
			}
		}
		metadata := map[string]string{"principal": ac.Principal}
		for k, v := range ac.Metadata {
			metadata[k] = v
		}
		context.Accounts[ac.Principal] = &account.Account{ac.Principal, group, ac.DisableDirectAuth, metadata, limitIP}
		group.Members = append(group.Members, ac.Principal)
	}
	return nil
}

func CompileGroups(context *Context, config *Config) error {
	context.Groups = make(map[string]*account.Group)
	for name, group := range config.Groups {
		if name == "" {
			return fmt.Errorf("A group name is required.")
		}
		var inherit *account.Group
		if group.Inherit != "" {
			inherit = context.Groups[group.Inherit]
			if inherit == nil {
				return fmt.Errorf("Cannot find group %s to inherit in %s (out of order?)", group.Inherit, name)
			}
		}
		context.Groups[name] = &account.Group{Inherit: inherit, Members: make([]string, 0)}
	}
	return nil
}

func CompileGrants(context *Context, config *Config) error {
	context.Grants = make(map[string]Grant)
	for api, grant := range config.Grants {
		if api == "" {
			return fmt.Errorf("An API name is required.")
		}
		group, found := context.Groups[grant.Group]
		if !found {
			return fmt.Errorf("Could not find group %s for grant %s", grant.Group, api)
		}
		privileges := make(map[string]account.Privilege)
		for _, accountname := range group.AllMembers() {
			_, found := privileges[accountname]
			if found {
				return fmt.Errorf("Duplicate account %s", accountname)
			}
			ac, found := context.Accounts[accountname]
			if !found {
				return fmt.Errorf("No such account %s", accountname)
			}
			cgrant, err := grant.CompileGrant(ac.Metadata, context)
			if err != nil {
				return err
			}
			priv, err := cgrant.CompileToPrivilege(context)
			if err != nil {
				return fmt.Errorf("%s (in grant %s for account %s)", err, api, accountname)
			}
			privileges[accountname] = priv
		}
		context.Grants[api] = Grant{api, group, privileges}
	}
	return nil
}
