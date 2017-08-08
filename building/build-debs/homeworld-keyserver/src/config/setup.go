package config

import (
	"path"
	"os"
	"authorities"
	"fmt"
	"account"
	"errors"
	"net"
)

var setup_steps = []SetupStep{CompileStaticFiles, CompileAuthorities, CompileGlobalAuthorities, CompileGroups, CompileAccounts, CompileGrants }

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
	for _, authority := range config.Authorities {
		if authority.Name == "" {
			return fmt.Errorf("An authority name is required.")
		}
		loaded, err := authority.Load(config.AuthorityDir)
		if err != nil {
			return err
		}
		context.Authorities[authority.Name] = loaded
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
		metadata := map[string]string { "principal": ac.Principal }
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
	for _, group := range config.Groups {
		if group.Name == "" {
			return fmt.Errorf("A group name is required.")
		}
		_, found := context.Groups[group.Name]
		if found {
			return fmt.Errorf("Duplicate group: %s", group.Name)
		}
		var inherit *account.Group
		if group.Inherit != "" {
			inherit = context.Groups[group.Inherit]
			if inherit == nil {
				return fmt.Errorf("Cannot find group %s to inherit in %s (out of order?)", group.Inherit, group.Name)
			}
		}
		context.Groups[group.Name] = &account.Group{Inherit: inherit, Members: make([]string, 0)}
	}
	return nil
}

func CompileGrants(context *Context, config *Config) error {
	context.Grants = make(map[string]Grant)
	for _, grant := range config.Grants {
		if grant.API == "" {
			return fmt.Errorf("An API name is required.")
		}
		_, found := context.Grants[grant.API]
		if found {
			return fmt.Errorf("Duplicate grant %s", grant.API)
		}
		group, found := context.Groups[grant.Group]
		if !found {
			return fmt.Errorf("Could not find group %s for grant %s", grant.Group, grant.API)
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
				return fmt.Errorf("%s (in grant %s for account %s)", err, grant.API, accountname)
			}
			privileges[accountname] = priv
		}
		context.Grants[grant.API] = Grant{grant.API, group, privileges}
	}
	return nil
}
