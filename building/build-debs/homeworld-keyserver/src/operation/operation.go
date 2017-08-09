package operation

import (
	"encoding/json"
	"fmt"
	"log"

	"account"
	"config"
)

type Operation struct {
	API  string
	body string
}

func InvokeAPIOperationSet(a *account.Account, context *config.Context, requestBody []byte) ([]byte, error) {
	ops := []Operation{}
	err := json.Unmarshal(requestBody, &ops)
	if err != nil {
		return nil, err
	}
	ctx := &account.OperationContext{a}
	results := make([]string, len(ops))
	for i, operation := range ops {
		result, err := InvokeAPIOperation(ctx, context, operation.API, operation.body)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return json.Marshal(results)
}

func InvokeAPIOperation(ctx *account.OperationContext, gctx *config.Context, API string, requestBody string) (string, error) {
	grant, found := gctx.Grants[API]
	if !found {
		return "", fmt.Errorf("Could not find API request %s", grant)
	}
	princ := ctx.Account.Principal
	priv, found := grant.PrivilegeByAccount[princ]
	if !found {
		return "", fmt.Errorf("Account %s does not have access to API call %s", princ, API)
	}
	log.Printf("Attempting to perform API operation %s for %s", API, princ)
	response, err := priv(ctx, requestBody)
	if err != nil {
		log.Printf("Operation %s for %s failed with error: %s.", API, princ, err)
		return "", err
	}
	log.Printf("Operation %s for %s succeeded.", API, princ)
	return response, nil
}
