package keygen

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
	"github.com/sipb/homeworld/platform/util/certutil"
)

const AuthorityBits = 4096

func GenerateTLSSelfSignedCert(key *rsa.PrivateKey, name string, present_as []string) ([]byte, error) {
	issueat := time.Now()

	certTemplate := &x509.Certificate{
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},

		BasicConstraintsValid: true,
		IsCA:       true,
		MaxPathLen: 1,

		NotBefore: issueat,
		NotAfter:  time.Unix(issueat.Unix()+86400*1000000, 0), // one million days in the future

		Subject:  pkix.Name{CommonName: "homeworld-authority-" + name},
		DNSNames: present_as,
	}

	return certutil.FinishCertificate(certTemplate, certTemplate, key.Public(), key)
}

func GenerateKeys(setup *worldconfig.SpireSetup, dir string) error {
	if info, err := os.Stat(dir); err != nil {
		return err
	} else if !info.IsDir() {
		return errors.New("expected authority directory, not authority file")
	}

	authorities := worldconfig.GenerateAuthorities(setup)

	for name, authority := range authorities {
		// private key
		privkey, err := rsa.GenerateKey(rand.Reader, AuthorityBits)
		if err != nil {
			return err
		}
		privkeybytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privkey)})
		err = ioutil.WriteFile(path.Join(dir, authority.Key), privkeybytes, os.FileMode(0600))
		if err != nil {
			return err
		}
		if authority.Type == "TLS" || authority.Type == "static" {
			// self-signed cert
			cert, err := GenerateTLSSelfSignedCert(privkey, name, authority.PresentAs)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(path.Join(dir, authority.Cert), cert, os.FileMode(0644))
			if err != nil {
				return err
			}
		} else if authority.Type == "SSH" {
			// SSH authorities are just pubkeys
			pkey, err := ssh.NewPublicKey(privkey.Public())
			if err != nil {
				return err
			}
			pubkey := ssh.MarshalAuthorizedKey(pkey)
			err = ioutil.WriteFile(path.Join(dir, authority.Cert), pubkey, os.FileMode(0644))
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("invalid authority type: %s", authority.Type)
		}
	}
	return nil
}
