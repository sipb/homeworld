package authorities

import "testing"

func TestDelegatedStub(t *testing.T) {
	key := NewDelegatedAuthority("test").GetPublicKey()
	if string(key) != "test" {
		t.Error("Delegated authority pubkey mismatch")
	}
}
