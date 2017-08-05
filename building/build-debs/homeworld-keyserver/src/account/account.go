package account

import (
	"authorities"
	"fmt"
	"log"
	"encoding/json"
)

type Account struct {
	Principal         string
	Group             *Group
	GrantingAuthority authorities.Authority
	Grants            map[string]*Grant
	Metadata          map[string]string
}

type Group struct {
	Name    string
	Members []string
	Inherit *Group
}

type Grant struct {
	API       string
	Privilege Privilege
}

type OperationContext struct {
	Account *Account
}

type Operation struct {
	API  string
	body string
}

func (g *Group) AllMembers() []string {
	members := make([]string, 0, 10)
	for g != nil {
		for _, member := range g.Members {
			members = append(members, member)
		}
		g = g.Inherit
	}
	return members
}

func (a *Account) InvokeAPIOperationSet(requestBody []byte) ([]byte, error) {
	ops := []Operation{}
	err := json.Unmarshal(requestBody, &ops)
	if err != nil {
		return nil, err
	}
	ctx := &OperationContext{a}
	results := make([]string, len(ops))
	for i, operation := range ops {
		result, err := ctx.InvokeAPIOperation(operation.API, operation.body)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return json.Marshal(results)
}

func (ctx *OperationContext) InvokeAPIOperation(API string, requestBody string) (string, error) {
	grant, found := ctx.Account.Grants[API]
	if !found {
		return "", fmt.Errorf("Could not find API request %s", grant)
	}
	princ := ctx.Account.Principal
	log.Println("Attempting to perform API operation %s for %s", API, princ)
	response, err := grant.Privilege(ctx, requestBody)
	if err != nil {
		log.Println("Operation %s for %s failed with error: %s.", API, princ, err)
		return "", err
	}
	log.Println("Operation %s for %s succeeded.", API, princ)
	return response, nil
}
