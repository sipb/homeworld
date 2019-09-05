package config

import (
	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
	"strings"
	"testing"
)

func TestContext_GetAccount(t *testing.T) {
	ac := &account.Account{Principal: "test-account"}
	ctx := &Context{Accounts: map[string]*account.Account{
		"test-account": ac,
		"wrong-princ":  ac,
	}}
	account_lookup, err := ctx.GetAccount("test-account")
	if err != nil {
		t.Error(err)
	} else if ac != account_lookup {
		t.Error("Account mismatch.")
	}
	_, err = ctx.GetAccount("wrong-princ")
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "Mismatched") {
		t.Errorf("Expected error to talk about mismatched name, not %s.", err)
	}
	_, err = ctx.GetAccount("missing-account")
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "missing-account") {
		t.Error("Expected error to talk about account name.")
	}
}
