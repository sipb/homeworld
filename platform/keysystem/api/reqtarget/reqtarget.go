package reqtarget

import "fmt"

type Request struct {
	API  string `json:"api"`
	Body string `json:"body"`
}

type RequestTarget interface {
	SendRequests([]Request) ([]string, error)
}

func SendRequest(a RequestTarget, api string, body string) (string, error) {
	strs, err := a.SendRequests([]Request{{API: api, Body: body}})
	if err != nil {
		return "", err
	}
	if len(strs) != 1 {
		return "", fmt.Errorf("wrong number of results: %d != 1", len(strs))
	}
	return strs[0], nil
}
