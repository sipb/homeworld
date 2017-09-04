package reqtarget

import (
	"fmt"
	"errors"
)

type impersonatedTarget struct {
	BaseTarget       RequestTarget
	ImpersonationAPI string
	User             string
}

func Impersonate(rt RequestTarget, api string, user string) (RequestTarget, error) {
	if api == "" {
		return nil, errors.New("Invalid API call.")
	}
	if user == "" {
		return nil, errors.New("Invalid user.")
	}
	rt2 := &impersonatedTarget{rt, api, user}
	_, err := rt2.SendRequests(nil)
	if err != nil {
		return nil, fmt.Errorf("While verifying impersonation functionality: %s", err)
	}
	return rt2, nil
}

func (t *impersonatedTarget) SendRequests(reqs []Request) ([]string, error) {
	new_requests := make([]Request, len(reqs)+1)
	for i, req := range reqs {
		new_requests[i+1] = req
	}
	new_requests[0] = Request{API: t.ImpersonationAPI, Body: t.User}
	responses, err := t.BaseTarget.SendRequests(new_requests)
	if err != nil {
		return nil, err
	}
	if len(responses) != len(new_requests) {
		return nil, errors.New("count mismatch during impersonation response verification")
	}
	return responses[1:], nil
}
