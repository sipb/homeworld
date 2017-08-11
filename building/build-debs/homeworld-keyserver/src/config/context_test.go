package config

import (
	"testing"
	"authorities"
	"io/ioutil"
	"bytes"
	"strings"
	"account"
)

func TestConfigAuthority_Load_TLS(t *testing.T) {
	auth, err := (&ConfigAuthority{Type:"TLS", Key: "test1.key", Cert: "test1.pem"}).Load("testdir")
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
	auth, err := (&ConfigAuthority{Type:"SSH", Key: "test2", Cert: "test2.pub"}).Load("testdir")
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
	_, err := (&ConfigAuthority{Type:"TLS", Key: "nokey.key", Cert: "test1.pem"}).Load("testdir")
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "nokey.key") {
		t.Error("Expected error to mention missing file.")
	}
}

func TestConfigAuthority_Load_NoCert(t *testing.T) {
	_, err := (&ConfigAuthority{Type:"TLS", Key: "test1.key", Cert: "nokey.pem"}).Load("testdir")
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "nokey.pem") {
		t.Error("Expected error to mention missing file.")
	}
}

func TestConfigAuthority_Load_EmptyDir(t *testing.T) {
	_, err := (&ConfigAuthority{Type:"TLS", Key: "test1.key", Cert: "test1.pem"}).Load("")
	if err == nil {
		t.Error("Expected error")
	} else if !strings.Contains(err.Error(), "Empty") {
		t.Errorf("Expected error to mention empty field, not %s.", err)
	}
}

func TestContext_GetAccount(t *testing.T) {
	ac := &account.Account{ Principal: "test-account" }
	ctx := &Context{Accounts: map[string]*account.Account {
		"test-account": ac,
		"wrong-princ": ac,
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
	ctx := &Context{Authorities: map[string]authorities.Authority {
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
	tls, err := (&ConfigAuthority{Type:"TLS", Key:"test1.key", Cert:"test1.pem"}).Load("testdir")
	if err != nil {
		t.Fatal(err)
	}
	ctx := &Context{Authorities: map[string]authorities.Authority {
		"fake-authority": &FakeAuthority{},
		"tls-authority": tls,
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
