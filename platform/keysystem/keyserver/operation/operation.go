package operation

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/sipb/homeworld/platform/keysystem/keyserver/account"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/config"
)

type OperationForbiddenError struct {
	Principal string
	API       string
}

func (o *OperationForbiddenError) Error() string {
	return fmt.Sprintf("account %s does not have access to API call %s", o.Principal, o.API)
}

func InvokeAPIOperationSet(a *account.Account, context *config.Context, requestBody []byte, logger *log.Logger) ([]byte, error) {
	var ops []map[string]string
	err := json.Unmarshal(requestBody, &ops)
	if err != nil {
		return nil, err
	}
	ctx := &account.OperationContext{a}
	results := make([]string, len(ops))
	for i, operation := range ops {
		api, found := operation["api"]
		if !found {
			return nil, errors.New("missing API request in JSON")
		}
		body, found := operation["body"]
		if !found {
			return nil, errors.New("missing body request in JSON")
		}
		result, err := InvokeAPIOperation(ctx, context, api, body, logger)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return json.Marshal(results)
}

func InvokeAPIOperation(ctx *account.OperationContext, gctx *config.Context, API string, requestBody string, logger *log.Logger) (string, error) {
	grant, found := gctx.Grants[API]
	if !found {
		return "", fmt.Errorf("could not find API request '%s'", API)
	}
	if ctx.Account == nil {
		return "", errors.New("missing account during request")
	}
	princ := ctx.Account.Principal
	priv, found := grant[princ]
	if !found {
		return "", &OperationForbiddenError{
			Principal: princ,
			API:       API,
		}
	}
	logger.Printf("attempting to perform API operation %s for %s", API, princ)
	response, err := priv(ctx, requestBody)
	if err != nil {
		logger.Printf("operation %s for %s failed with error: %s", API, princ, err)
		return "", err
	}
	logger.Printf("operation %s for %s succeeded", API, princ)
	return response, nil
}
