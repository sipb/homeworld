package verifier

import "net/http"

/*
 * A verifier is used for verifying that a particular client connecting to the keyserver holds a credential that
 * authenticates it as a particular account.
 */

type Verifier interface {
	HasAttempt(request *http.Request) bool
	Verify(request *http.Request) (principal string, err error)
}
