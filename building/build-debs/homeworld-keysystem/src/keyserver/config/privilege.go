package config

import (
	"errors"
	"fmt"
	"keyserver/account"
	"keyserver/authorities"
	"strconv"
	"time"
	"util/strutil"
)

type CompiledGrant struct {
	Privilege    string
	Scope        *account.Group
	Authority    authorities.Authority
	IsHost       *bool
	Lifespan     time.Duration
	CommonName   string
	AllowedNames []string
	Contents     string
}

func (grant *ConfigGrant) CompileGrant(vars map[string]string, ctx *Context) (*CompiledGrant, error) {
	g := &CompiledGrant{Privilege: grant.Privilege}
	if grant.Privilege == "" {
		return nil, errors.New("Expected privilege to be specified")
	}
	if grant.Scope != "" {
		scope, found := ctx.Groups[grant.Scope]
		if !found {
			return nil, fmt.Errorf("No such group %s", grant.Scope)
		}
		g.Scope = scope
	}
	if grant.Authority != "" {
		authority, err := ctx.GetAuthority(grant.Authority)
		if err != nil {
			return nil, err
		}
		g.Authority = authority
	}
	if grant.IsHost != "" {
		ishost, err := strconv.ParseBool(grant.IsHost)
		if err != nil {
			return nil, err
		}
		g.IsHost = &ishost
	}
	if grant.Lifespan != "" {
		lifespan, err := time.ParseDuration(grant.Lifespan)
		if err != nil {
			return nil, err
		}
		if lifespan <= 0 {
			return nil, errors.New("Nonpositive lifespans are not supported.")
		}
		g.Lifespan = lifespan
	}
	if grant.CommonName != "" {
		commonname, err := strutil.SubstituteVars(grant.CommonName, vars)
		if err != nil {
			return nil, err
		}
		g.CommonName = commonname
	}
	if grant.AllowedNames != nil {
		allowednames, err := strutil.SubstituteAllVars(grant.AllowedNames, vars)
		if err != nil {
			return nil, err
		}
		g.AllowedNames = allowednames
	}
	if grant.Contents != "" {
		contents, err := strutil.SubstituteVars(grant.Contents, vars)
		if err != nil {
			return nil, err
		}
		g.Contents = contents
	}
	return g, nil
}

func (grant *CompiledGrant) CompileToPrivilege(context *Context) (account.Privilege, error) {
	switch grant.Privilege {
	case "bootstrap-account":
		if grant.CommonName != "" || grant.AllowedNames != nil || grant.IsHost != nil || grant.Contents != "" || grant.Authority != nil {
			return nil, errors.New("Extraneous parameter(s) provided to bootstrap-account")
		}
		if grant.Scope == nil || grant.Lifespan == 0 {
			return nil, errors.New("Missing parameter(s) to bootstrap-account")
		}
		return account.NewBootstrapPrivilege(grant.Scope.AllMembers, grant.Lifespan, context.TokenVerifier.Registry)
	case "sign-ssh":
		if grant.Scope != nil || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameter(s) provided to sign-ssh")
		}
		if grant.Authority == nil || grant.IsHost == nil || grant.Lifespan == 0 || grant.CommonName == "" || grant.AllowedNames == nil {
			return nil, errors.New("Missing parameter(s) to sign-ssh")
		}
		return account.NewSSHGrantPrivilege(grant.Authority, *grant.IsHost, grant.Lifespan, grant.CommonName, grant.AllowedNames)
	case "sign-tls":
		if grant.Scope != nil || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameter(s) provided to sign-tls")
		}
		if grant.Authority == nil || grant.IsHost == nil || grant.Lifespan == 0 || grant.CommonName == "" {
			return nil, errors.New("Missing parameter(s) to sign-tls")
		}
		return account.NewTLSGrantPrivilege(grant.Authority, *grant.IsHost, grant.Lifespan, grant.CommonName, grant.AllowedNames)
	case "impersonate":
		if grant.Authority != nil || grant.CommonName != "" || grant.AllowedNames != nil || grant.Lifespan != 0 || grant.IsHost != nil || grant.Contents != "" {
			return nil, fmt.Errorf("Extraneous parameter(s) provided to impersonate")
		}
		if grant.Scope == nil {
			return nil, errors.New("Missing parameter(s) to impersonate")
		}
		return account.NewImpersonatePrivilege(context.GetAccount, grant.Scope)
	case "construct-configuration":
		if grant.Scope != nil || grant.CommonName != "" || grant.AllowedNames != nil || grant.Lifespan != 0 || grant.IsHost != nil || grant.Authority != nil {
			return nil, fmt.Errorf("Extraneous parameter(s) provided to construct-configuration")
		}
		if grant.Contents == "" {
			return nil, errors.New("Missing parameter(s) to construct-configuration")
		}
		return account.NewConfigurationPrivilege(grant.Contents)
	default:
		return nil, fmt.Errorf("No such privilege kind: %s", grant.Privilege)
	}
}
