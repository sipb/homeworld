package account

import (
	"authorities"
	"config"
	"privileges"
	"fmt"
	"log"
	"encoding/json"
	"context"
)

type Account struct {
	Principal         string
	Group             *config.Group
	GrantingAuthority authorities.Authority
	Grants            map[string]Grant
}

type Grant struct {
	API       string
	Privilege privileges.Privilege
}

type Operation struct {
	API string
	body string
}

func (a *Account) InvokeAPIOperationSet(context *config.Context, requestBody []byte) ([]byte, error) {
	operations := make([]Operation, 0)
	err := json.Unmarshal(requestBody, operations)
	if err != nil {
		return nil, err
	}
	results := make([]string, len(operations))
	for i, operation := range operations {
		result, err := a.InvokeAPIOperation(context, operation.API, operation.body)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return json.Marshal(results)
}

func (a *Account) InvokeAPIOperation(context *config.Context, API string, requestBody string) (string, error) {
	grant, found := a.Grants[API]
	if !found {
		return nil, fmt.Errorf("Could not find API request %s", grant)
	}
	log.Println("Attempting to perform API operation %s for %s", API, a.Principal)
	response, err := grant.Privilege(context, requestBody)
	if err != nil {
		log.Println("Operation %s for %s failed with error: %s.", API, a.Principal, err)
		return nil, err
	}
	log.Println("Operation %s for %s succeeded.", API, a.Principal)
	return response, nil
}
