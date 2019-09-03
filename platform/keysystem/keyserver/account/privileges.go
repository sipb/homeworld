package account

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/admit"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/authorities"
)

type OperationContext struct {
	Account *Account
}

type Privilege func(ctx *OperationContext, param string) (string, error)

type TLSSignPrivilege struct {
	authority  *authorities.TLSAuthority
	ishost     bool
	lifespan   time.Duration
	commonname string
	dnsnames   []string
}

func NewTLSGrantPrivilege(tauth *authorities.TLSAuthority, ishost bool, lifespan time.Duration, commonname string, dnsnames []string) Privilege {
	return func(_ *OperationContext, signingRequest string) (string, error) {
		return tauth.Sign(signingRequest, ishost, lifespan, commonname, dnsnames)
	}
}

func NewSSHGrantPrivilege(tauth *authorities.SSHAuthority, ishost bool, lifespan time.Duration, keyid string, principals []string) Privilege {
	return func(_ *OperationContext, signingRequest string) (string, error) {
		return tauth.Sign(signingRequest, ishost, lifespan, keyid, principals)
	}
}

func NewListAdmitRequestsPrivilege(checker *admit.AdmitChecker) Privilege {
	if checker == nil {
		panic("expected checker to exist")
	}
	return func(_ *OperationContext, request string) (string, error) {
		if len(request) != 0 {
			return "", errors.New("expected empty request to request list endpoint")
		}
		encoded, err := json.Marshal(checker.ListRequests())
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	}
}

func NewApproveAdmissionPrivilege(checker *admit.AdmitChecker) Privilege {
	if checker == nil {
		panic("expected checker to exist")
	}
	return func(_ *OperationContext, selection string) (string, error) {
		var approval admit.AdmitApproval
		err := json.Unmarshal([]byte(selection), &approval)
		if err != nil {
			return "", err
		}
		if approval.Principal == "" || approval.Fingerprint == "" {
			return "", errors.New("invalid principal or fingerprint")
		}
		err = approval.Normalize()
		if err != nil {
			return "", err
		}
		checker.Approve(approval)
		return "", nil
	}
}

func NewImpersonatePrivilege(getAccount func(string) (*Account, error), scope *Group) Privilege {
	return func(ctx *OperationContext, newPrincipal string) (string, error) {
		if !scope.HasMember(newPrincipal) {
			return "", errors.New("attempt to impersonate outside of allowed scope")
		}
		account, err := getAccount(newPrincipal)
		if err != nil {
			return "", err
		}
		if account.Principal != newPrincipal {
			return "", errors.New("wrong account returned")
		}
		ctx.Account = account
		return "", nil
	}
}

func NewConfigurationPrivilege(contents string) Privilege {
	return func(_ *OperationContext, request string) (string, error) {
		if len(request) != 0 {
			return "", errors.New("expected empty request to configuration endpoint")
		}
		return contents, nil
	}
}

func NewFetchKeyPrivilege(static *authorities.TLSAuthority) Privilege {
	return func(_ *OperationContext, request string) (string, error) {
		if len(request) != 0 {
			return "", errors.New("expected empty request to fetch-key endpoint")
		}
		return string(static.GetPrivateKey()), nil
	}
}
