package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"keysystem/keyserver/account"
	"keysystem/keyserver/authorities"
	"keysystem/keyserver/verifier"
	"net"
	"strings"
	"testing"
)

func TestCompileStaticFiles(t *testing.T) {
	var ctx Context
	if err := CompileStaticFiles(&ctx, &Config{
		StaticDir:   "testdir",
		StaticFiles: []ConfigStatic{"testa.txt", "testb.txt"},
	}); err != nil {
		t.Error(err)
	} else {
		if len(ctx.StaticFiles) != 2 {
			t.Error("Wrong number of static files")
		} else if _, found := ctx.StaticFiles["testa.txt"]; !found {
			t.Error("Expected to find testa.txt")
		} else if _, found := ctx.StaticFiles["testb.txt"]; !found {
			t.Error("Expected to find testb.txt")
		} else {
			if ctx.StaticFiles["testa.txt"].Filename != "testa.txt" {
				t.Error("Wrong filename")
			}
			if ctx.StaticFiles["testb.txt"].Filename != "testb.txt" {
				t.Error("Wrong filename")
			}
			if ctx.StaticFiles["testa.txt"].Filepath != "testdir/testa.txt" {
				t.Error("Wrong filename")
			}
			if ctx.StaticFiles["testb.txt"].Filepath != "testdir/testb.txt" {
				t.Error("Wrong filename")
			}
		}
	}
}

func TestCompileStaticFiles_Fail(t *testing.T) {
	var ctx Context
	if err := CompileStaticFiles(&ctx, &Config{
		StaticDir:   "testdir",
		StaticFiles: []ConfigStatic{"testa.txt", "testc.txt"},
	}); err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "testc.txt") {
		t.Error("Wrong error string.")
	}
}

func TestCompileAuthorities(t *testing.T) {
	var ctx Context
	if err := CompileAuthorities(&ctx, &Config{
		AuthorityDir: "testdir",
		Authorities: map[string]ConfigAuthority{
			"granting": {Type: "TLS", Key: "test1.key", Cert: "test1.pem"},
			"entry":    {Type: "SSH", Key: "test2", Cert: "test2.pub"},
		},
	}); err != nil {
		t.Error(err)
	} else {
		if len(ctx.Authorities) != 2 {
			t.Error("Wrong number of authorities.")
		} else if _, found := ctx.Authorities["granting"]; !found {
			t.Error("Authority not found.")
		} else if _, found := ctx.Authorities["entry"]; !found {
			t.Error("Authority not found.")
		} else {
			pubkey := ctx.Authorities["granting"].(*authorities.TLSAuthority).GetPublicKey()
			pubkey_ref, err := ioutil.ReadFile("testdir/test1.pem")
			if err != nil {
				t.Error(err)
			} else if !bytes.Equal(pubkey, pubkey_ref) {
				t.Error("Pubkey mismatch.")
			}
			pubkey = ctx.Authorities["entry"].(*authorities.SSHAuthority).GetPublicKey()
			pubkey_ref, err = ioutil.ReadFile("testdir/test2.pub")
			if err != nil {
				t.Error(err)
			} else if !bytes.Equal(pubkey, pubkey_ref) {
				t.Error("Pubkey mismatch.")
			}
		}
	}
}

func TestCompileAuthorities_Empty(t *testing.T) {
	var ctx Context
	if err := CompileAuthorities(&ctx, &Config{
		AuthorityDir: "testdir",
		Authorities: map[string]ConfigAuthority{
			"granting": {Type: "TLS", Key: "test1.key", Cert: "test1.pem"},
			"":         {Type: "SSH", Key: "test2", Cert: "test2.pub"},
		},
	}); err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "name is required") {
		t.Error("Expected name requirement.")
	}
}

func TestCompileAuthorities_Missing(t *testing.T) {
	var ctx Context
	if err := CompileAuthorities(&ctx, &Config{
		AuthorityDir: "testdir",
		Authorities: map[string]ConfigAuthority{
			"granting": {Type: "TLS", Key: "test1.key", Cert: "test1.pem"},
			"test":     {Type: "SSH", Key: "test2", Cert: "nokey.pub"},
		},
	}); err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "nokey.pub") {
		t.Error("Expected name requirement.")
	}
}

func TestCompileGlobalAuthorities(t *testing.T) {
	test1, err := (&ConfigAuthority{Type: "TLS", Key: "test1.key", Cert: "test1.pem"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	test3, err := (&ConfigAuthority{Type: "TLS", Key: "test1.key", Cert: "test1.pem"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	ctx := Context{
		Authorities: map[string]authorities.Authority{
			"servertls-test": test1,
			"authauth-test":  test3,
		},
	}
	if err := CompileGlobalAuthorities(&ctx, &Config{
		ServerTLS:               "servertls-test",
		AuthenticationAuthority: "authauth-test",
	}); err != nil {
		t.Error(err)
	} else {
		if ctx.AuthenticationAuthority != test3 {
			t.Error("Wrong authority.")
		}
		if ctx.ServerTLS != test1 {
			t.Error("Wrong authority.")
		}
	}
}

func TestCompileGlobalAuthorities_Unpop(t *testing.T) {
	ctx := Context{}
	if err := CompileGlobalAuthorities(&ctx, &Config{
		ServerTLS:               "",
		AuthenticationAuthority: "authauth-test",
	}); err == nil || !strings.Contains(err.Error(), "to be populated fields") {
		t.Error("Expected error.")
	}
	if err := CompileGlobalAuthorities(&ctx, &Config{
		ServerTLS:               "servertls-test",
		AuthenticationAuthority: "",
	}); err == nil || !strings.Contains(err.Error(), "to be populated fields") {
		t.Error("Expected error.")
	}
}

func TestCompileGlobalAuthorities_Missing(t *testing.T) {
	test1, err := (&ConfigAuthority{Type: "TLS", Key: "test1.key", Cert: "test1.pem"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	ctx := Context{
		Authorities: map[string]authorities.Authority{
			"populated-test": test1,
		},
	}
	if err := CompileGlobalAuthorities(&ctx, &Config{
		ServerTLS:               "populated-test",
		AuthenticationAuthority: "missing-test",
	}); err == nil || err.Error() != "No such authority: 'missing-test'" {
		t.Errorf("Expected error, not %s.", err)
	}
	if err := CompileGlobalAuthorities(&ctx, &Config{
		ServerTLS:               "missing-test",
		AuthenticationAuthority: "populated-test",
	}); err == nil || err.Error() != "No such authority: 'missing-test'" {
		t.Errorf("Expected error, not %s.", err)
	}
}

func TestCompileAccounts(t *testing.T) {
	test_group := &account.Group{
		Name:       "test-group",
		AllMembers: []string{},
	}
	ctx := Context{
		Groups: map[string]*account.Group{
			"test-group": test_group,
		},
	}
	if err := CompileAccounts(&ctx, &Config{
		Accounts: []ConfigAccount{
			{
				Principal:         "ruby-01.mit.edu",
				Metadata:          map[string]string{"abc": "def", "ip": "192.168.0.1"},
				Group:             "test-group",
				LimitIP:           false,
				DisableDirectAuth: false,
			},
			{
				Principal:         "ruby-02.mit.edu",
				Metadata:          map[string]string{"abc": "def", "ip": "192.168.0.2"},
				Group:             "test-group",
				LimitIP:           true,
				DisableDirectAuth: false,
			},
			{
				Principal:         "ruby-03.mit.edu",
				Metadata:          map[string]string{"abc": "def", "ip": "broken-ip-but-not-used"},
				Group:             "test-group",
				LimitIP:           false,
				DisableDirectAuth: true,
			},
			{
				Principal:         "ruby-04.mit.edu",
				Metadata:          map[string]string{"abc": "def", "ip": "192.168.0.4"},
				Group:             "test-group",
				LimitIP:           true,
				DisableDirectAuth: true,
			},
		},
	}); err != nil {
		t.Error(err)
	} else {
		names := []string{"ruby-01.mit.edu", "ruby-02.mit.edu", "ruby-03.mit.edu", "ruby-04.mit.edu"}
		if len(ctx.Accounts) != len(names) {
			t.Error("Expected four accounts.")
		}
		if len(test_group.AllMembers) != 4 {
			t.Error("Wrong number of group members.")
		}
		accounts := [4]*account.Account{}
		for i, name := range names {
			accounts[i], err = ctx.GetAccount(name)
			if err != nil {
				t.Error(err)
				continue
			}
			if test_group.AllMembers[i] != name {
				t.Error("Wrong member.")
			}
			if accounts[i].Principal != name {
				t.Error("Wrong name")
			}
			if accounts[i].Group != test_group {
				t.Error("Expected match for group")
			}
			if len(accounts[i].Metadata) != 3 {
				t.Error("Expected three metadata elements.")
			} else if accounts[i].Metadata["principal"] != name {
				t.Error("Expected principal registration.")
			} else if accounts[i].Metadata["abc"] != "def" {
				t.Error("Expected abc metadata.")
			} else if i != 2 && accounts[i].Metadata["ip"] != fmt.Sprintf("192.168.0.%d", i+1) {
				t.Error("Expected abc metadata.")
			}
			if i == 1 || i == 3 {
				if accounts[i].LimitIP == nil || !accounts[i].LimitIP.Equal(net.IPv4(192, 168, 0, byte(i+1))) {
					t.Error("Wrong limit IP")
				}
			} else {
				if accounts[i].LimitIP != nil {
					t.Error("Extraneous limit IP")
				}
			}
		}
		if accounts[0].DisableDirectAuth || accounts[1].DisableDirectAuth || !accounts[2].DisableDirectAuth || !accounts[3].DisableDirectAuth {
			t.Error("Wrong auth disablement.")
		}
	}
}

func TestCompileAccounts_RecursiveMembership(t *testing.T) {
	rg1 := &account.Group{Name: "root-group-1"}
	rg2 := &account.Group{Name: "root-group-2"}
	sg1 := &account.Group{Name: "sub-group-1", SubgroupOf: rg1}
	sg2 := &account.Group{Name: "sub-group-2", SubgroupOf: rg2}
	sg3 := &account.Group{Name: "sub-group-3", SubgroupOf: rg1}
	lg1 := &account.Group{Name: "leaf-group-1", SubgroupOf: sg2}
	lg2 := &account.Group{Name: "leaf-group-2", SubgroupOf: sg3}
	ctx := Context{Groups: map[string]*account.Group{}}
	cfg := Config{}
	groups := []*account.Group{rg1, rg2, sg1, sg2, sg3, lg1, lg2}
	for _, group := range groups {
		ctx.Groups[group.Name] = group
		cfg.Accounts = append(cfg.Accounts, ConfigAccount{
			Principal: "test-member-of-" + group.Name,
			Group:     group.Name,
		})
	}
	if err := CompileAccounts(&ctx, &cfg); err != nil {
		t.Error(err)
	} else {
		expected := [][]string{
			{"test-member-of-root-group-1", "test-member-of-sub-group-1", "test-member-of-sub-group-3", "test-member-of-leaf-group-2"},
			{"test-member-of-root-group-2", "test-member-of-sub-group-2", "test-member-of-leaf-group-1"},
			{"test-member-of-sub-group-1"},
			{"test-member-of-sub-group-2", "test-member-of-leaf-group-1"},
			{"test-member-of-sub-group-3", "test-member-of-leaf-group-2"},
			{"test-member-of-leaf-group-1"},
			{"test-member-of-leaf-group-2"},
		}
		if len(expected) != len(groups) || len(groups) != len(ctx.Groups) {
			t.Errorf("Wrong number of groups")
		} else {
			for i, expected_princs := range expected {
				actual := groups[i].AllMembers
				if len(actual) != len(expected_princs) {
					t.Errorf("Mismatch on number of principals for group %s: %d instead of %d", groups[i].Name, len(actual), len(expected_princs))
				} else {
					for j, val := range expected_princs {
						if actual[j] != val {
							t.Errorf("Mismatch between element %s and expected %s", actual[j], val)
						}
					}
				}
			}
		}
	}
}

func TestCompileAccounts_Fail(t *testing.T) {
	test_group := &account.Group{
		Name:       "test-group",
	}
	for _, test := range []struct {
		account ConfigAccount
		errbody string
	}{
		{
			account: ConfigAccount{
				Principal: "",
				Metadata:  map[string]string{"abc": "def", "ip": "192.168.0.1"},
				Group:     "test-group",
			},
			errbody: "account name is required",
		},
		{
			account: ConfigAccount{
				Principal: "duplicate-name",
				Metadata:  map[string]string{"abc": "def", "ip": "192.168.0.1"},
				Group:     "test-group",
			},
			errbody: "duplicate account",
		},
		{
			account: ConfigAccount{
				Principal: "real-name",
				Metadata:  map[string]string{"abc": "def", "ip": "192.168.0.1"},
				Group:     "missing-group",
			},
			errbody: "no such group",
		},
		{
			account: ConfigAccount{
				Principal: "real-name",
				Metadata:  map[string]string{"abc": "def", "ip": "broken-ip-that-is-used"},
				Group:     "test-group",
				LimitIP:   true,
			},
			errbody: "invalid IP address",
		},
	} {
		ctx := Context{
			Groups: map[string]*account.Group{
				"test-group": test_group,
			},
		}
		test_group.AllMembers = nil
		if err := CompileAccounts(&ctx, &Config{
			Accounts: []ConfigAccount{
				{
					Principal: "duplicate-name",
					Metadata:  map[string]string{},
					Group:     "test-group",
				},
				test.account,
			},
		}); err == nil {
			t.Error("Expected an error!")
		} else if !strings.Contains(err.Error(), test.errbody) {
			t.Errorf("Expected error that contains \"%s\", not \"%s\"", test.errbody, err)
		}
	}
}

func TestCompileGroups(t *testing.T) {
	ctx := Context{}
	if err := CompileGroups(&ctx, &Config{
		Groups: map[string]ConfigGroup{
			"test-group": {},
			"inheriting-group": {
				SubgroupOf: "test-group",
			},
			"other-group": {
				SubgroupOf: "test-group",
			},
			"extra-group": {
				SubgroupOf: "inheriting-group",
			},
		},
	}); err != nil {
		t.Error(err)
	} else {
		if len(ctx.Groups) != 4 {
			t.Error("Wrong number of groups.")
		} else {
			names := []string{"test-group", "inheriting-group", "other-group", "extra-group"}
			for _, name := range names {
				if ctx.Groups[name].Name != name {
					t.Error("Name mismatch.")
				}
				if len(ctx.Groups[name].AllMembers) != 0 {
					t.Error("Wrong number of members.")
				}
			}
			if ctx.Groups["test-group"].SubgroupOf != nil {
				t.Error("Wrong inherit.")
			}
			if ctx.Groups["inheriting-group"].SubgroupOf != ctx.Groups["test-group"] {
				t.Error("Wrong inherit.")
			}
			if ctx.Groups["other-group"].SubgroupOf != ctx.Groups["test-group"] {
				t.Error("Wrong inherit.")
			}
			if ctx.Groups["extra-group"].SubgroupOf != ctx.Groups["inheriting-group"] {
				t.Error("Wrong inherit.")
			}
		}
	}
}

func TestCompileGroups_Fail(t *testing.T) {
	ctx := Context{}
	if err := CompileGroups(&ctx, &Config{
		Groups: map[string]ConfigGroup{
			"": {},
		},
	}); err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Error("Missing or wrong error.")
	}
	if err := CompileGroups(&ctx, &Config{
		Groups: map[string]ConfigGroup{
			"group": {SubgroupOf: "missing"},
		},
	}); err == nil || !strings.Contains(err.Error(), "Cannot find group") {
		t.Error("Missing or wrong error.")
	}
}

func TestCompileGrants(t *testing.T) {
	admins := &account.Group{Name: "admins", AllMembers: []string{"my-admin"}}
	root_admins := &account.Group{Name: "root-admins", SubgroupOf: admins, AllMembers: []string{"my-admin"}}
	servers := &account.Group{Name: "servers", AllMembers: []string{"my-server"}}
	testservers := &account.Group{Name: "test-servers", SubgroupOf: servers, AllMembers: []string{"my-server"}}
	ctx := Context{
		TokenVerifier: verifier.NewTokenVerifier(),
		Groups: map[string]*account.Group{
			"admins":       admins,
			"root-admins":  root_admins,
			"servers":      servers,
			"test-servers": testservers,
		},
		Accounts: map[string]*account.Account{
			"my-admin":  {Group: root_admins},
			"my-server": {Group: testservers},
		},
	}
	err := CompileGrants(&ctx, &Config{
		Grants: map[string]ConfigGrant{
			"test-grant": {
				Privilege: "bootstrap-account",
				Group:     "admins",
				Scope:     "servers",
				Lifespan:  "4h",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	} else {
		if len(ctx.Grants) != 1 {
			t.Fatal("Wrong number of grants.")
		}
		grant := ctx.Grants["test-grant"]
		if grant.Group != admins {
			t.Error("Wrong admin group.")
		}
		if grant.API != "test-grant" {
			t.Error("Wrong API name.")
		}
		if len(grant.PrivilegeByAccount) != 1 {
			t.Fatalf("Wrong number of privileges: expected one, not %d.", len(grant.PrivilegeByAccount))
		}
		priv := grant.PrivilegeByAccount["my-admin"]
		tok, err := priv(nil, "my-server")
		if err != nil {
			t.Error(err)
		} else {
			princ, err := ctx.TokenVerifier.Registry.LookupToken(tok)
			if err != nil {
				t.Error(err)
			} else if princ.Subject != "my-server" {
				t.Error("Wrong principal.")
			} else if princ.Claim() != nil {
				t.Error("Cannot claim.")
			}
		}
	}
}

func TestCompileGrants_NoAPI(t *testing.T) {
	err := CompileGrants(&Context{}, &Config{
		Grants: map[string]ConfigGrant{
			"": {
				Privilege: "bootstrap-account",
				Group:     "admins",
				Scope:     "servers",
				Lifespan:  "4h",
			},
		},
	})
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "An API name is required.") {
		t.Error("Wrong error.")
	}
}

func TestCompileGrants_NoGroup(t *testing.T) {
	err := CompileGrants(&Context{}, &Config{
		Grants: map[string]ConfigGrant{
			"test-grant": {
				Privilege: "bootstrap-account",
				Group:     "admins",
				Scope:     "servers",
				Lifespan:  "4h",
			},
		},
	})
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "Could not find group admins") {
		t.Error("Wrong error.")
	}
}

func TestCompileGrants_DupAccount(t *testing.T) {
	admins := &account.Group{Name: "admins", AllMembers: []string{"my-admin", "my-admin"}}
	servers := &account.Group{Name: "servers", AllMembers: []string{"my-server"}}
	ctx := Context{
		TokenVerifier: verifier.NewTokenVerifier(),
		Groups: map[string]*account.Group{
			"admins":  admins,
			"servers": servers,
		},
		Accounts: map[string]*account.Account{
			"my-admin":  {Group: admins},
			"my-server": {Group: servers},
		},
	}
	err := CompileGrants(&ctx, &Config{
		Grants: map[string]ConfigGrant{
			"test-grant": {
				Privilege: "bootstrap-account",
				Group:     "admins",
				Scope:     "servers",
				Lifespan:  "4h",
			},
		},
	})
	if err == nil {
		t.Error("Expected error")
	} else if err.Error() != "Duplicate account my-admin" {
		t.Errorf("Wrong error: %s", err)
	}
}

func TestCompileGrants_NoAccount(t *testing.T) {
	admins := &account.Group{Name: "admins", AllMembers: []string{"my-admin"}}
	servers := &account.Group{Name: "servers", AllMembers: []string{"my-server"}}
	ctx := Context{
		TokenVerifier: verifier.NewTokenVerifier(),
		Groups: map[string]*account.Group{
			"admins":  admins,
			"servers": servers,
		},
		Accounts: map[string]*account.Account{
			"my-server": {Group: servers},
		},
	}
	err := CompileGrants(&ctx, &Config{
		Grants: map[string]ConfigGrant{
			"test-grant": {
				Privilege: "bootstrap-account",
				Group:     "admins",
				Scope:     "servers",
				Lifespan:  "4h",
			},
		},
	})
	if err == nil {
		t.Error("Expected error")
	} else if err.Error() != "No such account my-admin" {
		t.Errorf("Wrong error: %s", err)
	}
}

func TestCompileGrants_NoScope(t *testing.T) {
	admins := &account.Group{Name: "admins", AllMembers: []string{"my-admin"}}
	ctx := Context{
		TokenVerifier: verifier.NewTokenVerifier(),
		Groups: map[string]*account.Group{
			"admins": admins,
		},
		Accounts: map[string]*account.Account{
			"my-admin": {Group: admins},
		},
	}
	err := CompileGrants(&ctx, &Config{
		Grants: map[string]ConfigGrant{
			"test-grant": {
				Privilege: "bootstrap-account",
				Group:     "admins",
				Scope:     "servers",
				Lifespan:  "4h",
			},
		},
	})
	if err == nil {
		t.Error("Expected error")
	} else if err.Error() != "No such group servers" {
		t.Errorf("Wrong error: %s", err)
	}
}

func TestCompileGrants_NoParam(t *testing.T) {
	admins := &account.Group{Name: "admins", AllMembers: []string{"my-admin"}}
	ctx := Context{
		TokenVerifier: verifier.NewTokenVerifier(),
		Groups: map[string]*account.Group{
			"admins": admins,
		},
		Accounts: map[string]*account.Account{
			"my-admin": {Group: admins},
		},
	}
	err := CompileGrants(&ctx, &Config{
		Grants: map[string]ConfigGrant{
			"test-grant": {
				Privilege: "bootstrap-account",
				Group:     "admins",
				Lifespan:  "4h",
			},
		},
	})
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "Missing parameter(s)") {
		t.Errorf("Wrong error: %s", err)
	}
}
