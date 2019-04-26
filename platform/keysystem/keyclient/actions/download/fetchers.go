package download

import (
	"fmt"
	"github.com/pkg/errors"

	"github.com/sipb/homeworld/platform/keysystem/api/endpoint"
	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/keyclient/actloop"
)

func fetchAuthority(authority string) (FetchFunc, string) {
	info := fmt.Sprintf("pubkey for authority %s", authority)
	fetch := func(nac *actloop.NewActionContext) ([]byte, error) {
		result, err := nac.State.Keyserver.GetPubkey(authority)
		if err != nil {
			return nil, err
		}
		if len(result) == 0 {
			return nil, errors.New("empty response")
		}
		return result, nil
	}
	return fetch, info
}

func fetchStatic(static string) (FetchFunc, string) {
	info := fmt.Sprintf("static file %s", static)
	fetch := func(nac *actloop.NewActionContext) ([]byte, error) {
		result, err := nac.State.Keyserver.GetStatic(static)
		if err != nil {
			return nil, err
		}
		if len(result) == 0 {
			return nil, errors.New("empty response")
		}
		return result, nil
	}
	return fetch, info
}

func fetchAPI(api string) (FetchFunc, string) {
	info := fmt.Sprintf("result from api %s", api)
	fetch := func(nac *actloop.NewActionContext) ([]byte, error) {
		if !nac.State.CanRetry(api) {
			// nothing to do
			return nil, nil
		}
		if nac.State.Keygrant == nil {
			nac.Blocked(errors.New("no keygranting certificate ready"))
			return nil, nil
		}
		rt, err := nac.State.Keyserver.AuthenticateWithCert(*nac.State.Keygrant)
		if err != nil {
			return nil, err // no actual way for this part to fail
		}
		resp, err := reqtarget.SendRequest(rt, api, "")
		if err != nil {
			if _, is := errors.Cause(err).(endpoint.OperationForbidden); is {
				nac.State.RetryFailed(api)
			}
			return nil, err
		}
		if len(resp) == 0 {
			return nil, errors.New("empty response")
		}
		return []byte(resp), nil
	}
	return fetch, info
}
