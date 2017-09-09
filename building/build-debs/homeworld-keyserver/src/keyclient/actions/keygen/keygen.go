package keygen

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"util/fileutil"
	"log"
	"os"
	"path"
	"keyclient/config"
	"errors"
	"keyclient/actloop"
)

const RSA_BITS = 4096

type TLSKeygenAction struct {
	keypath string
	bits int
}

func PrepareKeygenAction(k config.ConfigKey) (actloop.Action, error) {
	switch k.Type {
	case "tls":
		return TLSKeygenAction{keypath: k.Key, bits: RSA_BITS}, nil
	case "ssh":
		// should probably include creating a .pub file as well
		return nil, errors.New("unimplemented operation: ssh key generation")
	case "ssh-pubkey":
		return nil, nil // key is pregenerated
	default:
		return nil, fmt.Errorf("unrecognized key type: %s", k.Type)
	}
}

func (ka TLSKeygenAction) Pending() (bool, error) {
	// it's acceptable for the directory to not exist, because we'll just create it later
	return !fileutil.Exists(ka.keypath), nil
}

func (ka TLSKeygenAction) CheckBlocker() error {
	return nil
}

func (ka TLSKeygenAction) Perform(_ *log.Logger) error {
	dirname := path.Dir(ka.keypath)
	err := fileutil.EnsureIsFolder(dirname)
	if err != nil {
		return fmt.Errorf("failed to prepare directory %s for generated key: %s", dirname, err)
	}

	private_key, err := rsa.GenerateKey(rand.Reader, ka.bits)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key (%d bits) for %s: %s", RSA_BITS, ka.keypath, err)
	}

	keydata := x509.MarshalPKCS1PrivateKey(private_key)

	err = fileutil.CreateFile(ka.keypath, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keydata}), os.FileMode(0600))
	if err != nil {
		return fmt.Errorf("failed to create file for generated key: %s", err)
	}
	return nil
}
