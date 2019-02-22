package reqtarget

import (
	"errors"
	"testing"

	"github.com/sipb/homeworld/platform/util/testutil"
)

type FakeTarget struct {
	cb func([]Request) ([]string, error)
}

func (f FakeTarget) SendRequests(r []Request) ([]string, error) {
	return f.cb(r)
}

func TestImpersonate(t *testing.T) {
	rt, err := Impersonate(FakeTarget{func(r []Request) ([]string, error) {
		if len(r) == 1 {
			// verification phase
			if r[0].API != "become-doppelganger" {
				t.Error("Wrong impersonation API")
			}
			if r[0].Body != "mephistopheles" {
				t.Error("Wrong impersonation body")
			}
			return []string{""}, nil
		}
		if len(r) != 3 {
			t.Error("wrong number of requests")
			return nil, errors.New("bad request count")
		} else {
			if r[0].API != "become-doppelganger" {
				t.Error("Wrong impersonation API")
			}
			if r[0].Body != "mephistopheles" {
				t.Error("Wrong impersonation body")
			}
			if r[1].API != "perform-action-1" {
				t.Error("Wrong perform API")
			}
			if r[1].Body != "parameter-A" {
				t.Error("Wrong perform body")
			}
			if r[2].API != "perform-action-2" {
				t.Error("Wrong perform API")
			}
			if r[2].Body != "parameter-B" {
				t.Error("Wrong perform body")
			}
			return []string{"", "result-A", "result-B"}, nil
		}
	}}, "become-doppelganger", "mephistopheles")
	if err != nil {
		t.Fatal(err)
	}
	results, err := rt.SendRequests([]Request{{"perform-action-1", "parameter-A"}, {"perform-action-2", "parameter-B"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("wrong result count %d", len(results))
	}
	if results[0] != "result-A" {
		t.Error("Wrong first result.")
	}
	if results[1] != "result-B" {
		t.Error("Wrong second result.")
	}
}

func TestImpersonate_NoAPI(t *testing.T) {
	_, err := Impersonate(nil, "", "daemon")
	testutil.CheckError(t, err, "Invalid API")
}

func TestImpersonate_NoUser(t *testing.T) {
	_, err := Impersonate(nil, "daemonize", "")
	testutil.CheckError(t, err, "Invalid user")
}

func TestImpersonate_Broken(t *testing.T) {
	_, err := Impersonate(FakeTarget{func(requests []Request) ([]string, error) {
		return nil, errors.New("invalid endpoint, somehow")
	}}, "daemonize", "daemon")
	testutil.CheckError(t, err, "While verifying impersonation functionality: invalid endpoint, somehow")
}

func TestImpersonate_Malformed(t *testing.T) {
	_, err := Impersonate(FakeTarget{func(requests []Request) ([]string, error) {
		return []string{"a", "b", "c", "d"}, nil
	}}, "daemonize", "daemon")
	testutil.CheckError(t, err, "While verifying impersonation functionality: count mismatch during impersonation response verification")
}

func TestImpersonate_FailOnPurpose(t *testing.T) {
	rt, err := Impersonate(FakeTarget{func(r []Request) ([]string, error) {
		if len(r) == 1 {
			// verification phase
			if r[0].API != "become-doppelganger" {
				t.Error("Wrong impersonation API")
			}
			if r[0].Body != "mephistopheles" {
				t.Error("Wrong impersonation body")
			}
			return []string{""}, nil
		}
		if len(r) != 3 {
			t.Error("wrong number of requests")
			return nil, errors.New("bad request count")
		} else {
			if r[0].API != "become-doppelganger" {
				t.Error("Wrong impersonation API")
			}
			if r[0].Body != "mephistopheles" {
				t.Error("Wrong impersonation body")
			}
			if r[1].API != "perform-action-1" {
				t.Error("Wrong perform API")
			}
			if r[1].Body != "parameter-A" {
				t.Error("Wrong perform body")
			}
			if r[2].API != "perform-action-2" {
				t.Error("Wrong perform API")
			}
			if r[2].Body != "parameter-B" {
				t.Error("Wrong perform body")
			}
			return nil, errors.New("purposeful failure")
		}
	}}, "become-doppelganger", "mephistopheles")
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.SendRequests([]Request{{"perform-action-1", "parameter-A"}, {"perform-action-2", "parameter-B"}})
	testutil.CheckError(t, err, "purposeful failure")
}
