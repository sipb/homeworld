package config

import (
	"testing"
	"authorities"
	"account"
	"time"
	"reflect"
	"strings"
	"verifier"
	"golang.org/x/crypto/ssh"
	"encoding/pem"
	"crypto/x509"
)

func TestConfigGrant_CompileGrant_Empty(t *testing.T) {
	config := ConfigGrant{Privilege: "test"}
	ctx := Context{}
	cpl, err := config.CompileGrant(map[string]string {}, &ctx)
	if err != nil {
		t.Error(err)
	}
	if cpl.Privilege != "test" {
		t.Error("Wrong privilege.")
	}
	if cpl.Contents != "" || cpl.AllowedNames != nil || cpl.CommonName != "" || cpl.IsHost != nil || cpl.Lifespan != 0 || cpl.Authority != nil || cpl.Scope != nil {
		t.Error("Non-empty result!")
	}
}

func TestConfigGrant_CompileGrant_Everything(t *testing.T) {
	for _, hv := range []bool { false, true } {
		var hvs string
		if hv {
			hvs = "true"
		} else {
			hvs = "false"
		}
		config := ConfigGrant{
			Privilege: "test2",
			Contents: "hello (mothership) world",
			Authority: "test-authority",
			Lifespan: "12h",
			IsHost: hvs,
			CommonName: "name-in-(language)",
			AllowedNames: []string { "mydomain.(language).solar.system" },
			Group: "test-group",
			Scope: "test-group",
		}
		ctx := Context{
			Authorities: map[string]authorities.Authority {
				"test-authority": &FakeAuthority{},
			},
			Groups: map[string]*account.Group {
				"test-group": {Name: "test-group"},
			},
		}
		cpl, err := config.CompileGrant(map[string]string { "mothership": "giant hand", "language": "scribbles" }, &ctx)
		if err != nil {
			t.Fatal(err)
		}
		expected := CompiledGrant{
			Privilege: "test2",
			Scope: ctx.Groups["test-group"],
			AllowedNames: []string { "mydomain.scribbles.solar.system" },
			CommonName: "name-in-scribbles",
			IsHost: cpl.IsHost, // checked separately
			Lifespan: time.Hour * 12,
			Authority: ctx.Authorities["test-authority"],
			Contents: "hello giant hand world",
		}
		if !reflect.DeepEqual(expected, *cpl) {
			t.Error("Result mismatch")
		}
		if cpl.IsHost == nil || *cpl.IsHost != hv {
			t.Errorf("Wrong IsHost.")
		}
	}
}

func TestConfigGrant_CompileGrant_FailPrivilege(t *testing.T) {
	config := ConfigGrant{
		Privilege: "",
	}
	ctx := Context{}
	_, err := config.CompileGrant(map[string]string {}, &ctx)
	if err == nil {
		t.Fatal("Expected error")
	} else if !strings.Contains(err.Error(), "Expected privilege") {
		t.Error("Wrong error.")
	}
}

func TestConfigGrant_CompileGrant_FailScope(t *testing.T) {
	config := ConfigGrant{
		Privilege: "test",
		Scope: "missing",
	}
	ctx := Context{}
	_, err := config.CompileGrant(map[string]string {}, &ctx)
	if err == nil {
		t.Fatal("Expected error")
	} else if !strings.Contains(err.Error(), "No such group") {
		t.Errorf("Wrong error: %s.", err)
	}
}

func TestConfigGrant_CompileGrant_FailAuthority(t *testing.T) {
	config := ConfigGrant{
		Privilege: "test",
		Authority: "missing",
	}
	ctx := Context{}
	_, err := config.CompileGrant(map[string]string {}, &ctx)
	if err == nil {
		t.Fatal("Expected error")
	} else if !strings.Contains(err.Error(), "No such authority") {
		t.Errorf("Wrong error: %s.", err)
	}
}

func TestConfigGrant_CompileGrant_FailIsHost(t *testing.T) {
	config := ConfigGrant{
		Privilege: "test",
		IsHost: "maybe",
	}
	ctx := Context{}
	_, err := config.CompileGrant(map[string]string {}, &ctx)
	if err == nil {
		t.Fatal("Expected error")
	} else if !strings.Contains(err.Error(), "invalid syntax") {
		t.Errorf("Wrong error: %s.", err)
	}
}

func TestConfigGrant_CompileGrant_FailLifespan_Bad(t *testing.T) {
	config := ConfigGrant{
		Privilege: "test",
		Lifespan: "4 hours",
	}
	ctx := Context{}
	_, err := config.CompileGrant(map[string]string {}, &ctx)
	if err == nil {
		t.Fatal("Expected error")
	} else if !strings.Contains(err.Error(), "unknown unit  hours") {
		t.Errorf("Wrong error: %s.", err)
	}
}

func TestConfigGrant_CompileGrant_FailLifespan_Negative(t *testing.T) {
	config := ConfigGrant{
		Privilege: "test",
		Lifespan: "-1h",
	}
	ctx := Context{}
	_, err := config.CompileGrant(map[string]string {}, &ctx)
	if err == nil {
		t.Fatal("Expected error")
	} else if err.Error() != "Nonpositive lifespans are not supported." {
		t.Errorf("Wrong error: %s.", err)
	}
}

func TestConfigGrant_CompileGrant_FailCommonName(t *testing.T) {
	config := ConfigGrant{
		Privilege: "test",
		CommonName: "(badvar1)",
	}
	ctx := Context{}
	_, err := config.CompileGrant(map[string]string {}, &ctx)
	if err == nil {
		t.Fatal("Expected error")
	} else if !strings.Contains(err.Error(), "badvar1") {
		t.Errorf("Wrong error: %s.", err)
	}
}

func TestConfigGrant_CompileGrant_FailAllowedNames(t *testing.T) {
	config := ConfigGrant{
		Privilege: "test",
		AllowedNames: []string { "(badvar2)" },
	}
	ctx := Context{}
	_, err := config.CompileGrant(map[string]string {}, &ctx)
	if err == nil {
		t.Fatal("Expected error")
	} else if !strings.Contains(err.Error(), "badvar2") {
		t.Errorf("Wrong error: %s.", err)
	}
}

func TestConfigGrant_CompileGrant_FailContents(t *testing.T) {
	config := ConfigGrant{
		Privilege: "test",
		Contents: "(badvar3)",
	}
	ctx := Context{}
	_, err := config.CompileGrant(map[string]string {}, &ctx)
	if err == nil {
		t.Fatal("Expected error")
	} else if !strings.Contains(err.Error(), "badvar3") {
		t.Errorf("Wrong error: %s.", err)
	}
}

func TestBootstrapAccount(t *testing.T) {
	ctx := Context{
		TokenVerifier: verifier.NewTokenVerifier(),
	}
	priv, err := (&CompiledGrant{
		Privilege: "bootstrap-account",
		Scope: &account.Group{
			Members: []string { "test1", "test2" },
		},
		Lifespan: time.Minute * 53,
	}).CompileToPrivilege(&ctx)
	if err != nil {
		t.Error(err)
	} else {
		tok, err := priv(nil,"test2")
		if err != nil {
			t.Error(err)
		} else {
			stok, err := ctx.TokenVerifier.Registry.LookupToken(tok)
			if err != nil {
				t.Error(err)
			} else {
				if stok.Subject != "test2" {
					t.Error("Wrong subject!")
				}
				if stok.Claim() != nil {
					t.Error("Broken claiming!")
				}
			}
		}
	}
}

const TEST_CSR = "-----BEGIN CERTIFICATE REQUEST-----\nMIIBVTCBvwIBADAWMRQwEgYDVQQDDAtjbGllbnQtdGVzdDCBnzANBgkqhkiG9w0B\nAQEFAAOBjQAwgYkCgYEAtKukT2LT/PJ/i1pbqfe4Vm9iN2yMFoiKj0em7FFOrAeU\n/5onq8fZEXhUruN+OhjMr+K1c2qy7noqbzD3Fz/vi2frB9DUFMA9rkj3teRIEXKB\nBDzb1cbDSTL0HxH47/tURxzxzGCVfTCc1xUY+dqMsd8SvowxuEptU4SO9H8CR2MC\nAwEAAaAAMA0GCSqGSIb3DQEBCwUAA4GBALCOKX+QHmNLGrrSCWB8p2iMuS+aPOcW\nYI9c1VaaTSQ43HOjF1smvGIa1iicM2L5zTBOEG36kI+sKFDOF2cXclhQF1WfLcxC\nIi/JSV+W7hbS6zWvJOnmoi15hzvVa1MRk8HZH+TpiMxO5uqQdDiEkV1sJ50v0ZtR\nTMuSBjdmmJ1t\n-----END CERTIFICATE REQUEST-----"
const TEST_PUB = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC5zWmwv8NiKfVkt9KHZ6vAWDnKonUGVbjE+REDhPZwU4obzMEjcx8Ha8mQHZSDzbW835DF9fvsJDARBnCIh/2AB1iUL0jdM2cRKKmqdzGrbHQmet4FgJoWCu7rQKgt4JTAxQVc0qGSBqBlKn2QCKtHUs9PJOEDHSz4l4LwiZ/E2xxD+5/M7EKdlcRXyBOZE6oAwIdV9JNjL0FiqN/QPWijZcFN0AWTql0NRxMq9EagOz9XhHLXdf3rPQzJ/IP/zK6ZB6DAQ53QDLfJ87PAeC/YmFWsB25lHGOV6X5bcyT0HDxfL1bYNCB0oNA417iDp5+yqYoFdDW1Ioj5P2QJbYm1 user@host"

func TestSignSSH(t *testing.T) {
	authority, err := (&ConfigAuthority{Type:"SSH", Key:"test2", Cert:"test2.pub"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	false := false
	priv, err := (&CompiledGrant{
		Privilege: "sign-ssh",
		Authority: authority,
		IsHost: &false,
		Lifespan: time.Minute * 53,
		CommonName: "my-keyid",
		AllowedNames: []string {"test-name"},
	}).CompileToPrivilege(nil)
	if err != nil {
		t.Error(err)
	} else {
		signed, err := priv(nil,TEST_PUB)
		if err != nil {
			t.Error(err)
		} else {
			cert, _, _, _, err := ssh.ParseAuthorizedKey([]byte(signed))
			if err != nil {
				t.Fatal(err)
			}
			if cert.(*ssh.Certificate).KeyId != "my-keyid" {
				t.Error("Wrong keyid.")
			}
		}
	}
}

func TestSignTLS(t *testing.T) {
	authority, err := (&ConfigAuthority{Type:"TLS", Key:"test1.key", Cert:"test1.pem"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	false := false
	priv, err := (&CompiledGrant{
		Privilege: "sign-tls",
		Authority: authority,
		IsHost: &false,
		Lifespan: time.Minute * 53,
		CommonName: "my-common-name",
		AllowedNames: []string {"test-dns.mit.edu"},
	}).CompileToPrivilege(nil)
	if err != nil {
		t.Error(err)
	} else {
		signed, err := priv(nil,TEST_CSR)
		if err != nil {
			t.Error(err)
		} else {
			block, rest := pem.Decode([]byte(signed))
			if len(rest) != 0 {
				t.Fatal(err)
			}
			if block.Type != "CERTIFICATE" {
				t.Error("Wrong block type.")
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				t.Fatal(err)
			}
			if cert.Subject.CommonName != "my-common-name" {
				t.Error("Wrong common name.")
			}
		}
	}
}

func TestImpersonate(t *testing.T) {
	scope := &account.Group{Members: []string {"member1"}}
	acnt := &account.Account{
		Principal: "member1",
	}
	context := Context{
		Accounts: map[string]*account.Account {
			"member1": acnt,
		},
	}
	priv, err := (&CompiledGrant{
		Privilege: "impersonate",
		Scope: scope,
	}).CompileToPrivilege(&context)
	if err != nil {
		t.Error(err)
	} else {
		opctx := account.OperationContext{}
		result, err := priv(&opctx,"member1")
		if err != nil {
			t.Error(err)
		} else {
			if result != "" {
				t.Error("Non-empty result")
			}
			if opctx.Account != acnt {
				t.Error("Expected account switch.")
			}
		}
	}
}

func TestConstructConfiguration(t *testing.T) {
	priv, err := (&CompiledGrant{
		Privilege: "construct-configuration",
		Contents: "hello world",
	}).CompileToPrivilege(nil)
	if err != nil {
		t.Error(err)
	} else {
		result, err := priv(nil,"")
		if err != nil {
			t.Error(err)
		} else if result != "hello world" {
			t.Error("Wrong result.")
		}
	}
}

func TestInvalidPrivilegeName(t *testing.T) {
	_, err := (&CompiledGrant{
		Privilege: "invalid-priv",
	}).CompileToPrivilege(nil)
	if err == nil {
		t.Error("Expected error.")
	} else if !strings.Contains(err.Error(), "No such privilege kind") {
		t.Error("Expected no such privilege kind message.")
	}
}

func TestExtraneousParamsToBootstrapAccount(t *testing.T) {
	ctx := Context{
		TokenVerifier: verifier.NewTokenVerifier(),
	}
	false := false
	for i, grant := range []CompiledGrant {
		{ // first one is valid
			Privilege: "bootstrap-account",
			Scope: &account.Group{
				Members: []string { "test1", "test2" },
			},
			Lifespan: time.Minute * 53,
		},
		{
			Privilege: "bootstrap-account",
			Scope: &account.Group{
				Members: []string { "test1", "test2" },
			},
			Lifespan: time.Minute * 53,
			CommonName: "extra-common-name",
		},
		{
			Privilege: "bootstrap-account",
			Scope: &account.Group{
				Members: []string { "test1", "test2" },
			},
			Lifespan: time.Minute * 53,
			AllowedNames: []string { "test" },
		},
		{
			Privilege: "bootstrap-account",
			Scope: &account.Group{
				Members: []string { "test1", "test2" },
			},
			Lifespan: time.Minute * 53,
			Authority: &FakeAuthority{},
		},
		{
			Privilege: "bootstrap-account",
			Scope: &account.Group{
				Members: []string { "test1", "test2" },
			},
			Lifespan: time.Minute * 53,
			IsHost: &false,
		},
		{
			Privilege: "bootstrap-account",
			Scope: &account.Group{
				Members: []string { "test1", "test2" },
			},
			Lifespan: time.Minute * 53,
			Contents: "content",
		},
	} {
		_, err := (&grant).CompileToPrivilege(&ctx)
		if i == 0 { // first one is valid
			if err != nil {
				t.Error("Unexpected error.")
			}
		} else {
			if err == nil {
				t.Error("Expected an error")
			} else if !strings.Contains(err.Error(), "Extraneous parameter(s)") {
				t.Error("Expected extraneous parameter errors.")
			}
		}
	}
}

func TestExtraneousParamsToSignSSH(t *testing.T) {
	authority, err := (&ConfigAuthority{Type:"SSH", Key:"test2", Cert:"test2.pub"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	false := false
	for i, grant := range []CompiledGrant {
		{ // first one is valid
			Privilege: "sign-ssh",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-ssh",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
			Scope: &account.Group{},
		},
		{
			Privilege: "sign-ssh",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
			Contents: "test",
		},
	} {
		_, err := (&grant).CompileToPrivilege(nil)
		if i == 0 { // first one is valid
			if err != nil {
				t.Errorf("Unexpected error %s.", err)
			}
		} else {
			if err == nil {
				t.Error("Expected an error")
			} else if !strings.Contains(err.Error(), "Extraneous parameter(s)") {
				t.Error("Expected extraneous parameter errors.")
			}
		}
	}
}

func TestExtraneousParamsToSignTLS(t *testing.T) {
	authority, err := (&ConfigAuthority{Type:"TLS", Key:"test1.key", Cert:"test1.pem"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	false := false
	for i, grant := range []CompiledGrant {
		{ // first one is valid
			Privilege: "sign-tls",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-tls",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
			Scope: &account.Group{},
		},
		{
			Privilege: "sign-tls",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
			Contents: "test",
		},
	} {
		_, err := (&grant).CompileToPrivilege(nil)
		if i == 0 { // first one is valid
			if err != nil {
				t.Errorf("Unexpected error %s.", err)
			}
		} else {
			if err == nil {
				t.Error("Expected an error")
			} else if !strings.Contains(err.Error(), "Extraneous parameter(s)") {
				t.Error("Expected extraneous parameter errors.")
			}
		}
	}
}

func TestExtraneousParamsToImpersonate(t *testing.T) {
	false := false
	for i, grant := range []CompiledGrant {
		{ // first one is valid
			Privilege: "impersonate",
			Scope: &account.Group{},
		},
		{
			Privilege: "impersonate",
			Scope: &account.Group{},
			Contents: "content",
		},
		{
			Privilege: "impersonate",
			Scope: &account.Group{},
			IsHost: &false,
		},
		{
			Privilege: "impersonate",
			Scope: &account.Group{},
			Authority: &FakeAuthority{},
		},
		{
			Privilege: "impersonate",
			Scope: &account.Group{},
			AllowedNames: []string { "name" },
		},
		{
			Privilege: "impersonate",
			Scope: &account.Group{},
			CommonName: "common-name",
		},
		{
			Privilege: "impersonate",
			Scope: &account.Group{},
			Lifespan: time.Hour,
		},
	} {
		_, err := (&grant).CompileToPrivilege(nil)
		if i == 0 { // first one is valid
			if err != nil {
				t.Errorf("Unexpected error %s.", err)
			}
		} else {
			if err == nil {
				t.Error("Expected an error")
			} else if !strings.Contains(err.Error(), "Extraneous parameter(s)") {
				t.Error("Expected extraneous parameter errors.")
			}
		}
	}
}

func TestExtraneousParamsToConstructConfiguration(t *testing.T) {
	false := false
	for i, grant := range []CompiledGrant {
		{ // first one is valid
			Privilege: "construct-configuration",
			Contents: "hello world",
		},
		{
			Privilege: "construct-configuration",
			Contents: "hello world",
			Scope: &account.Group{},
		},
		{
			Privilege: "construct-configuration",
			Contents: "hello world",
			IsHost: &false,
		},
		{
			Privilege: "construct-configuration",
			Contents: "hello world",
			Authority: &FakeAuthority{},
		},
		{
			Privilege: "construct-configuration",
			Contents: "hello world",
			AllowedNames: []string { "name" },
		},
		{
			Privilege: "construct-configuration",
			Contents: "hello world",
			CommonName: "common-name",
		},
		{
			Privilege: "construct-configuration",
			Contents: "hello world",
			Lifespan: time.Hour,
		},
	} {
		_, err := (&grant).CompileToPrivilege(nil)
		if i == 0 { // first one is valid
			if err != nil {
				t.Errorf("Unexpected error %s.", err)
			}
		} else {
			if err == nil {
				t.Error("Expected an error")
			} else if !strings.Contains(err.Error(), "Extraneous parameter(s)") {
				t.Error("Expected extraneous parameter errors.")
			}
		}
	}
}

func TestInsufficientParamsToBootstrapAccount(t *testing.T) {
	ctx := Context{
		TokenVerifier: verifier.NewTokenVerifier(),
	}
	for i, grant := range []CompiledGrant {
		{ // first one is valid
			Privilege: "bootstrap-account",
			Scope: &account.Group{
				Members: []string { "test1", "test2" },
			},
			Lifespan: time.Minute * 53,
		},
		{
			Privilege: "bootstrap-account",
			Scope: &account.Group{
				Members: []string { "test1", "test2" },
			},
		},
		{
			Privilege: "bootstrap-account",
			Lifespan: time.Minute * 53,
		},
	} {
		_, err := (&grant).CompileToPrivilege(&ctx)
		if i == 0 { // first one is valid
			if err != nil {
				t.Error("Unexpected error.")
			}
		} else {
			if err == nil {
				t.Error("Expected an error")
			} else if !strings.Contains(err.Error(), "Missing parameter(s)") {
				t.Error("Expected insufficient parameter errors.")
			}
		}
	}
}

func TestInsufficientParamsToSignSSH(t *testing.T) {
	authority, err := (&ConfigAuthority{Type:"SSH", Key:"test2", Cert:"test2.pub"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	false := false
	for i, grant := range []CompiledGrant {
		{ // first one is valid
			Privilege: "sign-ssh",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-ssh",
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-ssh",
			Authority: authority,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-ssh",
			Authority: authority,
			IsHost: &false,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-ssh",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-ssh",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
		},
	} {
		_, err := (&grant).CompileToPrivilege(nil)
		if i == 0 { // first one is valid
			if err != nil {
				t.Errorf("Unexpected error %s.", err)
			}
		} else {
			if err == nil {
				t.Error("Expected an error")
			} else if !strings.Contains(err.Error(), "Missing parameter(s)") {
				t.Error("Expected insufficient parameter errors.")
			}
		}
	}
}

func TestInsufficientParamsToSignTLS(t *testing.T) {
	authority, err := (&ConfigAuthority{Type:"TLS", Key:"test1.key", Cert:"test1.pem"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	false := false
	for i, grant := range []CompiledGrant {
		{ // first one is valid
			Privilege: "sign-tls",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-tls",
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-tls",
			Authority: authority,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-tls",
			Authority: authority,
			IsHost: &false,
			CommonName: "my-keyid",
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-tls",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			AllowedNames: []string {"test-name"},
		},
		{
			Privilege: "sign-tls",
			Authority: authority,
			IsHost: &false,
			Lifespan: time.Minute * 53,
			CommonName: "my-keyid",
		},
	} {
		_, err := (&grant).CompileToPrivilege(nil)
		if i == 0 { // first one is valid
			if err != nil {
				t.Errorf("Unexpected error %s.", err)
			}
		} else {
			if err == nil {
				t.Error("Expected an error")
			} else if !strings.Contains(err.Error(), "Missing parameter(s)") {
				t.Error("Expected insufficient parameter errors.")
			}
		}
	}
}

func TestInsufficientParamsToImpersonate(t *testing.T) {
	for i, grant := range []CompiledGrant {
		{ // first one is valid
			Privilege: "impersonate",
			Scope: &account.Group{},
		},
		{
			Privilege: "impersonate",
		},
	} {
		_, err := (&grant).CompileToPrivilege(nil)
		if i == 0 { // first one is valid
			if err != nil {
				t.Errorf("Unexpected error %s.", err)
			}
		} else {
			if err == nil {
				t.Error("Expected an error")
			} else if !strings.Contains(err.Error(), "Missing parameter(s)") {
				t.Error("Expected insufficient parameter errors.")
			}
		}
	}
}

func TestInsufficientParamsToConstructConfiguration(t *testing.T) {
	for i, grant := range []CompiledGrant {
		{ // first one is valid
			Privilege: "construct-configuration",
			Contents: "hello world",
		},
		{
			Privilege: "construct-configuration",
		},
	} {
		_, err := (&grant).CompileToPrivilege(nil)
		if i == 0 { // first one is valid
			if err != nil {
				t.Errorf("Unexpected error %s.", err)
			}
		} else {
			if err == nil {
				t.Error("Expected an error")
			} else if !strings.Contains(err.Error(), "Missing parameter(s)") {
				t.Error("Expected insufficient parameter errors.")
			}
		}
	}
}
