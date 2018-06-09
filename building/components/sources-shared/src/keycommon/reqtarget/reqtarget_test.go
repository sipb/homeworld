package reqtarget

import (
	"errors"
	"testing"
	"util/testutil"
)

func TestSendRequest(t *testing.T) {
	rt := FakeTarget{func(requests []Request) ([]string, error) {
		if len(requests) != 1 {
			return nil, errors.New("count mismatch")
		} else {
			return []string{"alvarids"}, nil
		}
	}}
	result, err := SendRequest(rt, "after", "one-million-days")
	if err != nil {
		t.Fatal(err)
	}
	if result != "alvarids" {
		t.Error("wrong result from a million days in the future")
	}
}

func TestSendRequest_Fail(t *testing.T) {
	rt := FakeTarget{func(requests []Request) ([]string, error) {
		if len(requests) != 1 {
			return nil, errors.New("count mismatch")
		} else {
			return nil, errors.New("purposeful failure")
		}
	}}
	_, err := SendRequest(rt, "after", "one-million-days")
	testutil.CheckError(t, err, "purposeful failure")
}

func TestSendRequest_NotEnoughResults(t *testing.T) {
	rt := FakeTarget{func(requests []Request) ([]string, error) {
		if len(requests) != 1 {
			return nil, errors.New("count mismatch")
		} else {
			return []string{}, nil
		}
	}}
	_, err := SendRequest(rt, "after", "one-million-days")
	testutil.CheckError(t, err, "wrong number of results")
}

func TestSendRequest_TooManyResults(t *testing.T) {
	rt := FakeTarget{func(requests []Request) ([]string, error) {
		if len(requests) != 1 {
			return nil, errors.New("count mismatch")
		} else {
			return []string{"embrans", "vonahi"}, nil
		}
	}}
	_, err := SendRequest(rt, "after", "one-million-days")
	testutil.CheckError(t, err, "wrong number of results")
}
