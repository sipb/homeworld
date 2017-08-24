package keyclient

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"keyclient/util"
	"log"
	"os"
	"path"
)

type TLSKeygenAction struct {
	keypath string
	logger  log.Logger
}

func (ka TLSKeygenAction) Perform() error {
	if util.Exists(ka.keypath) {
		return ErrNothingToDo // already exists
	}

	dirname := path.Dir(ka.keypath)
	err := util.EnsureIsFolder(dirname)
	if err != nil {
		return fmt.Errorf("Failed to prepare directory %s for generated key: %s", dirname, err)
	}

	private_key, err := rsa.GenerateKey(rand.Reader, RSA_BITS)
	if err != nil {
		return fmt.Errorf("Failed to generate RSA key (%d bits) for %s: %s", RSA_BITS, ka.keypath, err)
	}

	keydata := x509.MarshalPKCS1PrivateKey(private_key)

	file_out, err := os.OpenFile(ka.keypath, os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("Failed to create file for generated key: %s", err)
	}
	succeeded := false
	defer func() {
		if !succeeded {
			file_out.Close()
			err := os.Remove(ka.keypath)
			ka.logger.Printf("While aborting key generation and removing wedged file: %s", err)
		}
	}()
	err = pem.Encode(file_out, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keydata})
	if err != nil {
		return fmt.Errorf("Could not successfully write file for generated key: %s", err)
	} else {
		err := file_out.Close()
		if err != nil {
			return fmt.Errorf("Could not successfully write file for generated key: %s", err)
		}
		succeeded = true
		return nil
	}
}
