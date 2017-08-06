package config

import (
	"io/ioutil"
	"path"
	"fmt"
	"strings"
	"authorities"
	"time"
	"strconv"
	"token"
	"account"
	"os"
)

type StaticFile struct {
	Filename string
	Filepath string
}

type Context struct {
	Authorities   map[string]authorities.Authority
	Groups        map[string]*account.Group
	GroupGrants   map[string][]ConfigGrant
	Accounts      map[string]*account.Account
	TokenRegistry *token.TokenRegistry
	Authenticator *authorities.TLSAuthority
	ServerTLS     *authorities.TLSAuthority
	StaticFiles   map[string]StaticFile
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

func (config *Config) Compile() (*Context, error) {
	staticfiles, err := CompileStaticFiles(config.StaticDir, config.StaticFiles)
	if err != nil {
		return nil, err
	}
	authority_map, err := CompileAuthorities(config.AuthorityDir, config.Authorities)
	if err != nil {
		return nil, err
	}
	if config.Authenticator == "" {
		return nil, fmt.Errorf("No authenticator specified.")
	}
	authenticator_i, found := authority_map[config.Authenticator]
	if !found {
		return nil, fmt.Errorf("Authenticator not found: %s", config.Authenticator)
	}
	authenticator, ok := authenticator_i.(*authorities.TLSAuthority)
	if !ok {
		return nil, fmt.Errorf("Authenticator is not a TLS authority.")
	}
	servertls_i, found := authority_map[config.ServerTLS]
	if !found {
		return nil, fmt.Errorf("ServerTLS not found: %s", config.ServerTLS)
	}
	servertls, ok := servertls_i.(*authorities.TLSAuthority)
	if !ok {
		return nil, fmt.Errorf("ServerTLS is not a TLS authority.")
	}
	groups, grants, err := CompileGroups(config.Groups)
	if err != nil {
		return nil, err
	}
	registry := token.NewTokenRegistry()
	ctx := &Context{authority_map, groups, grants, nil, registry, authenticator, servertls, staticfiles}
	ctx.Accounts, err = CompileAccounts(config.Accounts, ctx)
	if err != nil {
		return nil, err
	}
	return ctx, nil
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

func CompileGroups(groups []ConfigGroup) (map[string]*account.Group, map[string][]ConfigGrant, error) {
	groupout := make(map[string]*account.Group)
	grantout := make(map[string][]ConfigGrant)
	for _, group := range groups {
		if group.Name == "" {
			return nil, nil, fmt.Errorf("A group name is required.")
		}
		_, found := groupout[group.Name]
		if found {
			return nil, nil, fmt.Errorf("Duplicate group: %s", group.Name)
		}
		grants := append([]ConfigGrant{}, group.Grants...)
		var inherit *account.Group
		if group.Inherit != "" {
			inherit = groupout[group.Inherit]
			if inherit == nil {
				return nil, nil, fmt.Errorf("Cannot find group %s to inherit in %s (out of order?)", group.Inherit, group.Name)
			}
			grants = append(grants, grantout[group.Inherit]...)
		}
		groupout[group.Name] = &account.Group{Inherit: inherit, Members: make([]string, 0)}
		grantout[group.Name] = grants
	}
	return groupout, grantout, nil
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

func CompileAccounts(accounts []ConfigAccount, ctx *Context) (map[string]*account.Account, error) {
	out := make(map[string]*account.Account)
	for _, ac := range accounts {
		if ac.Principal == "" {
			return nil, fmt.Errorf("An account name is required.")
		}
		_, found := out[ac.Principal]
		if found {
			return nil, fmt.Errorf("Duplicate account %s", ac.Principal)
		}
		group := ctx.Groups[ac.Group]
		if group == nil {
			return nil, fmt.Errorf("No such group %s (in account %s)", ac.Group, ac.Principal)
		}
		authority, found := ctx.Authorities[ac.Realm]
		if !found {
			return nil, fmt.Errorf("No such authority %s (in account %s)", ac.Realm, ac.Principal)
		}
		grants, err := CompileGrants(group, ac.Principal, ac.Metadata, ctx)
		if err != nil {
			return nil, err
		}
		out[ac.Principal] = &account.Account{ac.Principal, group, authority, grants, ac.Metadata}
		group.Members = append(group.Members, ac.Principal)
	}
	return out, nil
}

func CompileGrants(group *account.Group, principal string, metadata map[string]string, ctx *Context) (map[string]*account.Grant, error) {
	metadata["principal"] = principal // TODO: break encapsulation less?
	grants := make(map[string]*account.Grant)
	for _, grant := range ctx.GroupGrants[group.Name] {
		if grant.API == "" {
			return nil, fmt.Errorf("An API name is required.")
		}
		_, found := grants[grant.API]
		if found {
			return nil, fmt.Errorf("Duplicate grant %s (in account %s)", grant.API, principal)
		}
		cgrant, err := CompileGrant(grant, metadata, ctx)
		if err != nil {
			return nil, err
		}
		grants[grant.API] = cgrant
	}
	return grants, nil
}

func SubstituteVars(within string, vars map[string]string) (string, error) {
	parts := strings.Split(within, "(")
	snippets := []string{parts[0] }
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

func CompileGrant(grant ConfigGrant, vars map[string]string, ctx *Context) (*account.Grant, error) {
	var privilege account.Privilege
	switch grant.Privilege {
	case "bootstrap-account":
		if grant.CommonName != "" || len(grant.AllowedNames) != 0 || grant.IsHost != "" || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameters provided to bootstrap-account in %s", grant.API)
		}
		scope := ctx.Groups[grant.Scope]
		if scope == nil {
			return nil, fmt.Errorf("No such group %s in grant %s", grant.Scope, grant.API)
		}
		lifespan, err := time.ParseDuration(grant.Lifespan)
		if err != nil {
			return nil, err
		}
		privilege, err = account.NewBootstrapPrivilege(scope.AllMembers(), lifespan, ctx.TokenRegistry)
		if err != nil {
			return nil, err
		}
	case "sign-ssh":
		if grant.Scope != "" || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameters provided to Sign-ssh in %s", grant.API)
		}
		authority := ctx.Authorities[grant.Authority]
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
		privilege, err = account.NewSSHGrantPrivilege(authority, ishost, lifespan, keyid, principals)
		if err != nil {
			return nil, err
		}
	case "sign-tls":
		if grant.Scope != "" || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameters provided to Sign-tls in %s", grant.API)
		}
		authority := ctx.Authorities[grant.Authority]
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
		privilege, err = account.NewTLSGrantPrivilege(authority, ishost, lifespan, commonname, altnames)
		if err != nil {
			return nil, err
		}
	case "impersonate":
		if grant.Scope != "" || grant.CommonName != "" || len(grant.AllowedNames) != 0 || grant.Lifespan != "" || grant.IsHost != "" || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameters provided to impersonate in %s", grant.API)
		}
		group := ctx.Groups[grant.Scope]
		if group == nil {
			return nil, fmt.Errorf("No such group %s in grant %s", grant.Scope, grant.API)
		}
		var err error
		privilege, err = account.NewImpersonatePrivilege(ctx.GetAccount, group)
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
		privilege, err = account.NewConfigurationPrivilege(contents)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("No such privilege %s in grant %s", grant.Privilege, grant.API)
	}
	if privilege == nil {
		return nil, fmt.Errorf("Internal error: privilege is nil.")
	}
	return &account.Grant{API: grant.API, Privilege: privilege}, nil
}
