package keygen

import (
	"keyserver/config"
	"os"
	"crypto/rsa"
	"crypto/rand"
	"encoding/pem"
	"crypto/x509"
	"io/ioutil"
	"path"
	"math/big"
	"crypto/x509/pkix"
	"time"
	"golang.org/x/crypto/ssh"
	"errors"
)

const AUTHORITY_BITS = 4096

func GenerateTLSSelfSignedCert(key *rsa.PrivateKey, name string, present_as []string) ([]byte, error) {
	serialNumber, err := rand.Int(rand.Reader, (&big.Int{}).Exp(big.NewInt(2), big.NewInt(159), nil))
	if err != nil {
		return nil, err
	}

	issueat := time.Now()

	extKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}

	certTemplate := &x509.Certificate{
		SignatureAlgorithm: x509.SHA256WithRSA,

		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: extKeyUsage,

		BasicConstraintsValid: true,
		IsCA:       true,
		MaxPathLen: 1,

		SerialNumber: serialNumber,

		NotBefore: issueat,
		NotAfter:  time.Unix(time.Now().Unix() + 86400 * 1000000, 0), // one million days in the future

		Subject:     pkix.Name{CommonName: "homeworld-authority-" + name},
		DNSNames:    present_as,
	}

	cert, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, key.Public(), key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}), nil
}

func GenerateKeys(cfg *config.Config, dir string, keyserver_group string) error {
	if info, err := os.Stat(dir); err != nil {
		return err
	} else if !info.IsDir() {
		return errors.New("expected authority directory, not authority file")
	}

	for name, authority := range cfg.Authorities {
		// private key
		privkey, err := rsa.GenerateKey(rand.Reader, AUTHORITY_BITS)
		if err != nil {
			return err
		}
		privkeybytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privkey)})
		err = ioutil.WriteFile(path.Join(dir, authority.Key), privkeybytes, os.FileMode(0600))
		if err != nil {
			return err
		}
		if authority.Type == "TLS" {
			present_as := []string{}
			if name == cfg.ServerTLS {
				for _, account := range cfg.Accounts {
					if account.Group == keyserver_group {
						present_as = append(present_as, account.Principal)
					}
				}
			}
			// self-signed cert
			cert, err := GenerateTLSSelfSignedCert(privkey, name, present_as)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(path.Join(dir, authority.Cert), cert, os.FileMode(0644))
			if err != nil {
				return err
			}
		} else {
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
		}
	}
	return nil
}