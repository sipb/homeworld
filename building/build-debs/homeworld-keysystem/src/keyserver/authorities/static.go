package authorities

import (
	"errors"
)

type StaticAuthority struct {
	key  []byte
	cert []byte
}

func LoadStaticAuthority(keydata []byte, certdata []byte) (Authority, error) {
	if keydata == nil || certdata == nil {
		return nil, errors.New("no data in static authority")
	}
	return &StaticAuthority{keydata, certdata}, nil
}

func (t *StaticAuthority) GetPublicKey() []byte {
	return t.cert
}

func (t *StaticAuthority) GetPrivateKey() []byte {
	return t.key
}
