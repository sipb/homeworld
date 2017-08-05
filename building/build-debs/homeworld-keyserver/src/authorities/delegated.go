package authorities

import (
	"net/http"
	"fmt"
	"time"
)

type DelegatedAuthority struct {
	Name string
}

func NewDelegatedAuthority(name string) Authority {
	return &DelegatedAuthority{name}
}

func (d *DelegatedAuthority) Verify(r *http.Request) (string, error) {
	return "", fmt.Errorf("Cannot directly authenticate against delegated authorities.")
}

func (d *DelegatedAuthority) Sign(_ []byte, _ bool, _ time.Duration, _ string, _ []string) ([]byte, error) {
	return nil, fmt.Errorf("Cannot sign messages with a delegated authority.")
}

func (d *DelegatedAuthority) GetPublicKey() []byte {
	return []byte(d.Name) // stub
}