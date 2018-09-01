package certutil

import (
	"crypto/x509"
	"crypto/rsa"
	"math/big"
	"encoding/pem"
	"crypto/rand"
	"crypto"
)

func FinishCertificate(template *x509.Certificate, parent *x509.Certificate, pubkey crypto.PublicKey, signer *rsa.PrivateKey) ([]byte, error) {
	var err error
	template.SignatureAlgorithm = x509.SHA256WithRSA
	template.SerialNumber, err = rand.Int(rand.Reader, (&big.Int{}).Exp(big.NewInt(2), big.NewInt(159), nil))
	if err != nil {
		return nil, err
	}

	signed_cert, err := x509.CreateCertificate(rand.Reader, template, parent, pubkey, signer)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: signed_cert}), nil
}
