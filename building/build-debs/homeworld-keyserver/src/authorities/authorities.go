package authorities

import (
	"net/http"
)

type Authority interface {
	Verify(r *http.Request) (string, error)
	GetPublicKey() []byte
}
