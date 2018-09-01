package config

import (
	"bytes"
	"io/ioutil"
	"keysystem/keyserver/account"
	"keysystem/keyserver/authorities"
	"strings"
	"testing"
	"time"
)

func TestConfigAuthority_Load_TLS(t *testing.T) {
	auth, err := (&ConfigAuthority{Type: "TLS", Key: "test1.key", Cert: "test1.pem"}).Load("testdir")
	if err != nil {
		t.Error(err)
	} else {
		pubkey := auth.(*authorities.TLSAuthority).GetPublicKey()
		pubkey_ref, err := ioutil.ReadFile("testdir/test1.pem")
		if err != nil {
			t.Error(err)
		} else if !bytes.Equal(pubkey, pubkey_ref) {
			t.Error("Pubkey mismatch.")
		}
	}
}

func TestConfigAuthority_Load_SSH(t *testing.T) {
	auth, err := (&ConfigAuthority{Type: "SSH", Key: "test2", Cert: "test2.pub"}).Load("testdir")
	if err != nil {
		t.Error(err)
	} else {
		pubkey := auth.(*authorities.SSHAuthority).GetPublicKey()
		pubkey_ref, err := ioutil.ReadFile("testdir/test2.pub")
		if err != nil {
			t.Error(err)
		} else if !bytes.Equal(pubkey, pubkey_ref) {
			t.Error("Pubkey mismatch.")
		}
	}
}

func TestConfigAuthority_Load_NoKey(t *testing.T) {
	_, err := (&ConfigAuthority{Type: "TLS", Key: "nokey.key", Cert: "test1.pem"}).Load("testdir")
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "nokey.key") {
		t.Error("Expected error to mention missing file.")
	}
}

func TestConfigAuthority_Load_NoCert(t *testing.T) {
	_, err := (&ConfigAuthority{Type: "TLS", Key: "test1.key", Cert: "nokey.pem"}).Load("testdir")
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "nokey.pem") {
		t.Error("Expected error to mention missing file.")
	}
}

func TestConfigAuthority_Load_EmptyDir(t *testing.T) {
	_, err := (&ConfigAuthority{Type: "TLS", Key: "test1.key", Cert: "test1.pem"}).Load("")
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "Empty") {
		t.Errorf("Expected error to mention empty field, not %s.", err)
	}
}

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

type FakeAuthority struct {
	pubkey []byte
}

func (f *FakeAuthority) GetPublicKey() []byte {
	return f.pubkey
}

func TestContext_GetAuthority(t *testing.T) {
	at := &FakeAuthority{}
	ctx := &Context{Authorities: map[string]authorities.Authority{
		"test-authority": at,
	}}
	authority_lookup, err := ctx.GetAuthority("test-authority")
	if err != nil {
		t.Error(err)
	} else if at != authority_lookup {
		t.Error("Authority mismatch.")
	}
	_, err = ctx.GetAuthority("missing-authority")
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "missing-authority") {
		t.Error("Expected error to talk about authority name.")
	}
}

func TestContext_GetTLSAuthority(t *testing.T) {
	tls, err := (&ConfigAuthority{Type: "TLS", Key: "test1.key", Cert: "test1.pem"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{Authorities: map[string]authorities.Authority{
		"fake-authority": &FakeAuthority{},
		"tls-authority":  tls,
	}}
	authority_lookup, err := ctx.GetTLSAuthority("tls-authority")
	if err != nil {
		t.Error(err)
	} else if authority_lookup != tls {
		t.Error("Authority mismatch.")
	}
	_, err = ctx.GetTLSAuthority("fake-authority")
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "not a TLS authority") {
		t.Error("Expected error to talk about wrong kind.")
	}
	_, err = ctx.GetTLSAuthority("missing-authority")
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "missing-authority") {
		t.Error("Expected error to talk about missing name.")
	}
}

func TestConfig_Compile(t *testing.T) {
	config := Config{
		AuthorityDir:            "testdir",
		StaticDir:               "testdir",
		AuthenticationAuthority: "granting",
		ServerTLS:               "granting", // don't do this in production
		StaticFiles:             []ConfigStatic{"testa.txt"},
		Authorities: map[string]ConfigAuthority{
			"granting": {Type: "TLS", Key: "test1.key", Cert: "test1.pem"},
		},
		Accounts: []ConfigAccount{
			{Principal: "my-admin", Group: "admins", DisableDirectAuth: true},
		},
		Groups: map[string]ConfigGroup{
			"admins": {},
		},
		Grants: map[string]ConfigGrant{
			"test-1": {
				Group:     "admins",
				Privilege: "construct-configuration",
				Contents:  "this is a test!",
			},
		},
	}
	ctx, err := config.Compile()
	if err != nil {
		t.Error(err)
	} else {
		expected_pubkey_bytes, err := ioutil.ReadFile("testdir/test1.pem")
		if err != nil {
			t.Error(err)
		}
		if len(ctx.Authorities) != 1 {
			t.Error("Wrong # of authorities")
		} else if !bytes.Equal(ctx.Authorities["granting"].(*authorities.TLSAuthority).GetPublicKey(), expected_pubkey_bytes) {
			t.Error("Wrong authority pubkey.")
		} else if ctx.ServerTLS != ctx.Authorities["granting"] {
			t.Error("Wrong granting authority.")
		} else if ctx.AuthenticationAuthority != ctx.Authorities["granting"] {
			t.Error("Wrong authentication authority.")
		}
		// check if the verifier is properly initialized
		teststr, err := ctx.TokenVerifier.Registry.LookupToken(ctx.TokenVerifier.Registry.GrantToken("test", time.Hour))
		if err != nil {
			t.Error(err)
		} else if teststr.Subject != "test" {
			t.Error("Wrong token back.")
		}
		if len(ctx.StaticFiles) != 1 {
			t.Error("Wrong number of static files.")
		} else if ctx.StaticFiles["testa.txt"].Filename != "testa.txt" {
			t.Errorf("Wrong filename %s.", ctx.StaticFiles["testa.txt"].Filename)
		} else if ctx.StaticFiles["testa.txt"].Filepath != "testdir/testa.txt" {
			t.Error("Wrong filename.")
		}
		if len(ctx.Groups) != 1 {
			t.Error("Wrong number of groups.")
		} else if ctx.Groups["admins"].Name != "admins" {
			t.Error("Wrong group name.")
		} else if ctx.Groups["admins"].SubgroupOf != nil {
			t.Error("Unexpected subgroupof.")
		} else if len(ctx.Groups["admins"].AllMembers) != 1 {
			t.Error("Wrong number of members.")
		} else if ctx.Groups["admins"].AllMembers[0] != "my-admin" {
			t.Error("Wrong number of members.")
		}
		if len(ctx.Accounts) != 1 {
			t.Error("Wrong number of accounts.")
		} else if ctx.Accounts["my-admin"].Principal != "my-admin" {
			t.Error("Wrong admin.")
		} else if ctx.Accounts["my-admin"].Group != ctx.Groups["admins"] {
			t.Error("Wrong group.")
		} else if ctx.Accounts["my-admin"].LimitIP != nil {
			t.Error("Wrong limitip.")
		} else if !ctx.Accounts["my-admin"].DisableDirectAuth {
			t.Error("Wrong disabledirectauth.")
		} else if len(ctx.Accounts["my-admin"].Metadata) != 1 {
			t.Error("Wrong amount of metadata.")
		} else if ctx.Accounts["my-admin"].Metadata["principal"] != "my-admin" {
			t.Error("Wrong metadata value.")
		}
		if len(ctx.Grants) != 1 {
			t.Error("Wrong number of grants.")
		} else if ctx.Grants["test-1"].API != "test-1" {
			t.Error("Wrong grant API")
		} else if ctx.Grants["test-1"].Group != ctx.Groups["admins"] {
			t.Error("Wrong grant group.")
		} else if len(ctx.Grants["test-1"].PrivilegeByAccount) != 1 {
			t.Error("Wrong number of grant instances.")
		} else {
			res, err := ctx.Grants["test-1"].PrivilegeByAccount["my-admin"](nil, "")
			if err != nil {
				t.Error(err)
			} else if res != "this is a test!" {
				t.Error("Wrong result from privilege!")
			}
		}
	}
}

func TestConfig_Compile_Fail(t *testing.T) {
	config := Config{
		StaticDir:               "testdir",
		AuthenticationAuthority: "granting",
		ServerTLS:               "granting", // don't do this in production
		StaticFiles:             []ConfigStatic{"testa.txt"},
		Authorities: map[string]ConfigAuthority{
			"granting": {Type: "TLS", Key: "test1.key", Cert: "test1.pem"},
		},
		Accounts: []ConfigAccount{
			{Principal: "my-admin", Group: "admins", DisableDirectAuth: true},
		},
		Groups: map[string]ConfigGroup{
			"admins": {},
		},
		Grants: map[string]ConfigGrant{
			"test-1": {
				Group:     "admins",
				Privilege: "construct-configuration",
				Contents:  "this is a test!",
			},
		},
	}
	_, err := config.Compile()
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "Empty directory path.") {
		t.Errorf("Wrong error: %s", err)
	}
}
