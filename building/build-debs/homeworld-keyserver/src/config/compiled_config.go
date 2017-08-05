package config

import (
	"io/ioutil"
	"path"
	"fmt"
	"strings"
	"authorities"
	"privileges"
	"time"
	"strconv"
	"token"
	"account"
	"os"
)

type Group struct {
	Members []string
	Grants []ConfigGrant
	Inherit *Group
}

type StaticFile struct {
	Filename string
	Filepath string
}

type Context struct {
	Authorities map[string]authorities.Authority
	Groups map[string]*Group
	Accounts map[string]account.Account
	TokenRegistry *token.TokenRegistry
	Authenticator authorities.Authority
	ServerTLS authorities.Authority
	StaticFiles map[string]StaticFile
}

func (g *Group) AllGrants() []ConfigGrant {
	grants := make([]ConfigGrant, 0, 10)
	for g != nil {
		for _, grant := range g.Grants {
			grants = append(grants, grant)
		}
		g = g.Inherit
	}
	return grants
}

func (g *Group) AllMembers() []string {
	members := make([]string, 0, 10)
	for g != nil {
		for _, member := range g.Members {
			members = append(members, member)
		}
		g = g.Inherit
	}
	return members
}

func (config *Config) Compile() (*Context, error) {
	staticfiles, err := CompileStaticFiles(config.StaticDir, config.StaticFiles)
	if err != nil {
		return nil, err
	}
	authorities, err := CompileAuthorities(config.AuthorityDir, config.Authorities)
	if err != nil {
		return nil, err
	}
	if config.Authenticator == "" {
		return nil, fmt.Errorf("No authenticator specified.")
	}
	authenticator, found := authorities[config.Authenticator]
	if !found {
		return nil, fmt.Errorf("Authenticator not found: %s", config.Authenticator)
	}
	servertls, found := authorities[config.ServerTLS]
	if !found {
		return nil, fmt.Errorf("ServerTLS not found: %s", config.ServerTLS)
	}
	groups, err := CompileGroups(config.Groups)
	if err != nil {
		return nil, err
	}
	registry := token.NewTokenRegistry()
	accounts, err := CompileAccounts(config.Accounts, authorities, groups, registry)
	if err != nil {
		return nil, err
	}
	return &Context{authorities, groups, accounts, registry, authenticator, servertls, staticfiles}, nil
}

func CompileStaticFiles(dir string, files []string) (map[string]StaticFile, error) {
	out := make(map[string]StaticFile)
	for _, file := range files {
		fullpath := path.Join(dir, file)
		openfile, err := os.Open(fullpath)
		if err != nil {
			return nil, err
		}
		openfile.Close()
		out[file] = StaticFile{file, fullpath}
	}
	return out, nil
}

func CompileGroups(groups []ConfigGroup) (map[string]*Group, error) {
	out := make(map[string]*Group)
	for _, group := range groups {
		if group.Name == "" {
			return nil, fmt.Errorf("A group name is required.")
		}
		_, found := out[group.Name]
		if found {
			return nil, fmt.Errorf("Duplicate group: %s", group.Name)
		}
		var inherit *Group
		if group.Inherit != "" {
			inherit = out[group.Inherit]
			if inherit == nil {
				return nil, fmt.Errorf("Cannot find group %s to inherit in %s (out of order?)", group.Inherit, group.Name)
			}
		}
		out[group.Name] = &Group{Inherit: inherit, Grants: group.Grants, Members: make([]string, 0)}
	}
	return out, nil
}

func CompileAuthorities(directory string, certauthorities []ConfigAuthority) (map[string]authorities.Authority, error) {
	out := make(map[string]authorities.Authority)
	for _, authority := range certauthorities {
		if authority.Name == "" {
			return nil, fmt.Errorf("An authority name is required.")
		}
		if authority.Type == "delegated" {
			if authority.Key != "" || authority.Cert != "" {
				return nil, fmt.Errorf("Extraneous key or cert in delegated authority %s", authority.Name)
			}
			out[authority.Name] = authorities.NewDelegatedAuthority(authority.Name)
			continue
		}
		if authority.Type != "SSH" && authority.Type != "TLS" {
			return nil, fmt.Errorf("Unknown authority type: %s", authority.Type)
		}
		keydata, err := ioutil.ReadFile(path.Join(directory, authority.Key))
		if err != nil {
			return nil, err
		}
		certdata, err := ioutil.ReadFile(path.Join(directory, authority.Cert))
		if err != nil {
			return nil, err
		}
		var loaded authorities.Authority
		if authority.Type == "SSH" {
			loaded, err = authorities.LoadSSHAuthority(keydata, certdata)
		} else {
			loaded, err = authorities.LoadTLSAuthority(keydata, certdata)
		}
		if err != nil {
			return nil, err
		}
		out[authority.Name] = loaded
	}
	return out, nil
}

func CompileAccounts(accounts []ConfigAccount, authorities map[string]authorities.Authority, groups map[string]*Group, registry *token.TokenRegistry) (map[string]account.Account, error) {
	out := make(map[string]account.Account)
	for _, account := range accounts {
		if account.Principal == "" {
			return nil, fmt.Errorf("An account name is required.")
		}
		_, found := out[account.Principal]
		if found {
			return nil, fmt.Errorf("Duplicate account %s", account.Principal)
		}
		group := groups[account.Group]
		if group == nil {
			return nil, fmt.Errorf("No such group %s (in account %s)", account.Group, account.Principal)
		}
		authority, found := authorities[account.Realm]
		if !found {
			return nil, fmt.Errorf("No such authority %s (in account %s)", account.Realm, account.Principal)
		}
		grants, err := CompileGrants(group, account.Principal, account.Metadata, groups, authorities, registry)
		if err != nil {
			return nil, err
		}
		out[account.Principal] = account.Account{account.Principal, group, authority, grants}
		group.Members = append(group.Members, account.Principal)
	}
	return out, nil
}

func CompileGrants(group *Group, principal string, metadata map[string]string, groups map[string]*Group, authorities map[string]authorities.Authority, registry *token.TokenRegistry) (map[string]account.Grant, error) {
	metadata["principal"] = principal // TODO: break encapsulation less?
	grants := make(map[string]account.Grant)
	for _, grant := range group.AllGrants() {
		if grant.API == "" {
			return nil, fmt.Errorf("An API name is required.")
		}
		_, found := grants[grant.API]
		if found {
			return nil, fmt.Errorf("Duplicate grant %s (in account %s)", grant.API, principal)
		}
		cgrant, err := CompileGrant(grant, metadata, groups, authorities, registry)
		if err != nil {
			return nil, err
		}
		grants[grant.API] = cgrant
	}
	return grants, nil
}

func SubstituteVars(within string, vars map[string]string) (string, error) {
	parts := strings.Split(within, "(")
	snippets := []string { parts[0] }
	for _, part := range parts[1:] {
		subparts := strings.Split(part, ")")
		if len(subparts) < 2 {
			return "", fmt.Errorf("Missing close parenthesis in substitution string '%s'", within)
		}
		if len(subparts) > 2 {
			return "", fmt.Errorf("Extraneous close parenthesis in substitution string '%s'", within)
		}
		varname, text := subparts[0], subparts[1]
		value := vars[varname]
		if value == "" {
			return "", fmt.Errorf("Undefined metadata variable %s in substitution string '%s'", varname, within)
		}
		snippets = append(snippets, value)
		snippets = append(snippets, text)
	}
	return strings.Join(snippets, ""), nil
}

func SubstituteAllVars(within []string, vars map[string]string) ([]string, error) {
	out := make([]string, len(within))
	for i, str := range within {
		value, err := SubstituteVars(str, vars)
		if err != nil {
			return nil, err
		}
		out[i] = value
	}
	return out, nil
}

func CompileGrant(grant ConfigGrant, vars map[string]string, groups map[string]*Group, authorities map[string]authorities.Authority, registry *token.TokenRegistry) (account.Grant, error) {
	var privilege privileges.Privilege
	switch grant.Privilege {
	case "bootstrap-account":
		if grant.CommonName != "" || len(grant.AllowedNames) != 0 || grant.IsHost != "" || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameters provided to bootstrap-account in %s", grant.API)
		}
		scope := groups[grant.Scope]
		if scope == nil {
			return nil, fmt.Errorf("No such group %s in grant %s", grant.Scope, grant.API)
		}
		lifespan, err := time.ParseDuration(grant.Lifespan)
		if err != nil {
			return nil, err
		}
		privilege, err = privileges.NewBootstrapPrivilege(scope.AllMembers(), lifespan, registry)
		if err != nil {
			return nil, err
		}
	case "Sign-ssh":
		if grant.Scope != "" || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameters provided to Sign-ssh in %s", grant.API)
		}
		authority := authorities[grant.Authority]
		if authority == nil {
			return nil, fmt.Errorf("No such authority %s in grant %s", grant.Authority, grant.API)
		}
		keyid, err := SubstituteVars(grant.CommonName, vars)
		if err != nil {
			return nil, err
		}
		principals, err := SubstituteAllVars(grant.AllowedNames, vars)
		if err != nil {
			return nil, err
		}
		ishost, err := strconv.ParseBool(grant.IsHost)
		if err != nil {
			return nil, err
		}
		lifespan, err := time.ParseDuration(grant.Lifespan)
		if err != nil {
			return nil, err
		}
		privilege, err = privileges.NewSSHGrantPrivilege(authority, ishost, lifespan, keyid, principals)
		if err != nil {
			return nil, err
		}
	case "Sign-tls":
		if grant.Scope != "" || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameters provided to Sign-tls in %s", grant.API)
		}
		authority := authorities[grant.Authority]
		if authority == nil {
			return nil, fmt.Errorf("No such authority %s in grant %s", grant.Authority, grant.API)
		}
		commonname, err := SubstituteVars(grant.CommonName, vars)
		if err != nil {
			return nil, err
		}
		altnames, err := SubstituteAllVars(grant.AllowedNames, vars)
		if err != nil {
			return nil, err
		}
		ishost, err := strconv.ParseBool(grant.IsHost)
		if err != nil {
			return nil, err
		}
		lifespan, err := time.ParseDuration(grant.Lifespan)
		if err != nil {
			return nil, err
		}
		privilege, err = privileges.NewTLSGrantPrivilege(authority, ishost, lifespan, commonname, altnames)
		if err != nil {
			return nil, err
		}
	case "delegate-authority":
		if grant.Scope != "" || grant.CommonName != "" || len(grant.AllowedNames) != 0 || grant.Lifespan != "" || grant.IsHost != "" || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameters provided to delegate-authority in %s", grant.API)
		}
		authority := authorities[grant.Authority]
		if authority == nil {
			return nil, fmt.Errorf("No such authority %s in grant %s", grant.Authority, grant.API)
		}
		var err error
		privilege, err = privileges.NewDelegateAuthorityPrivilege(authority)
		if err != nil {
			return nil, err
		}
	case "construct-configuration":
		if grant.Scope != "" || grant.CommonName != "" || len(grant.AllowedNames) != 0 || grant.Lifespan != "" || grant.IsHost != "" || grant.Authority != "" {
			return nil, fmt.Errorf("Extraneous parameters provided to construct-configuration in %s", grant.API)
		}
		contents, err := SubstituteVars(grant.Contents, vars)
		if err != nil {
			return nil, err
		}
		privilege, err = privileges.NewConfigurationPrivilege(contents)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("No such privilege %s in grant %s", grant.Privilege, grant.API)
	}
	if privilege == nil {
		return nil, fmt.Errorf("Internal error: privilege is nil.")
	}
	return account.Grant{API: grant.API, Privilege: privilege}, nil
}
