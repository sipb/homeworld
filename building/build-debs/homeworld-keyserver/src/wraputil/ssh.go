package wraputil

import (
	"golang.org/x/crypto/ssh"
	"errors"
	"strings"
)

func ParseSSHTextPubkey(content []byte) (ssh.PublicKey, error) {
	// TODO: make sure there are no extraneous prefixes
	pubkey, _, _, rest, err := ssh.ParseAuthorizedKey(content)
	if err != nil {
		return nil, err
	}
	if len(rest) > 0 {
		return nil, errors.New("extraneous trailing data after end of pubkey")
	}
	if strings.Count(strings.Trim(string(content), "\n"), "\n") > 0 {
		return nil, errors.New("extraneous leading data before pubkey")
	}
	return pubkey, nil
}
