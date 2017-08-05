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

type Operation struct {
	API string
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
	operations := make([]Operation, 0)
	err := json.Unmarshal(requestBody, operations)
	if err != nil {
		return nil, err
	}
	results := make([]string, len(operations))
	for i, operation := range operations {
		result, err := a.InvokeAPIOperation(operation.API, operation.body)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return json.Marshal(results)
}

func (a *Account) InvokeAPIOperation(API string, requestBody string) (string, error) {
	grant, found := a.Grants[API]
	if !found {
		return "", fmt.Errorf("Could not find API request %s", grant)
	}
	log.Println("Attempting to perform API operation %s for %s", API, a.Principal)
	response, err := grant.Privilege(requestBody)
	if err != nil {
		log.Println("Operation %s for %s failed with error: %s.", API, a.Principal, err)
		return "", err
	}
	log.Println("Operation %s for %s succeeded.", API, a.Principal)
	return response, nil
}
