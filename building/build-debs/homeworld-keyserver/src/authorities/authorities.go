package authorities

import (
	"net/http"
	"time"
)

type Authority interface {
	Verify(r *http.Request) (string, error)
	GetPublicKey() []byte
}
