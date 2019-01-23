package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/sipb/homeworld/platform/util/fileutil"
	"github.com/sipb/homeworld/platform/util/wraputil"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

func GenerateKeypair(commonname string, dns []string, ips []net.IP, parentkey *rsa.PrivateKey, parentcert *x509.Certificate) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not generate TLS keypair: %s", err.Error())
	}

	serialNumber, err := rand.Int(rand.Reader, (&big.Int{}).Exp(big.NewInt(2), big.NewInt(159), nil))
	if err != nil {
		return nil, nil, fmt.Errorf("Could not generate TLS keypair: %s", err.Error())
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
		NotAfter:  issueat.Add(time.Hour),

		Subject:     pkix.Name{CommonName: commonname},
		DNSNames:    dns,
		IPAddresses: ips,
	}

	if parentcert == nil {
		parentcert = certTemplate
		parentkey = key
	}

	signed_cert, err := x509.CreateCertificate(rand.Reader, certTemplate, parentcert, key.Public(), parentkey)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not generate TLS keypair: %s", err.Error())
	}
	cert, err := x509.ParseCertificate(signed_cert)
	if err != nil {
		return nil, nil, fmt.Errorf("Could not generate TLS keypair: %s", err.Error())
	}
	return key, cert, nil
}

func GenerateKeypairPEMs(commonname string, dns []string, ips []net.IP, parentkeydata []byte, parentcertdata []byte) ([]byte, []byte, error) {
	var parentkey *rsa.PrivateKey
	var parentcert *x509.Certificate
	var err error
	if parentkeydata != nil {
		parentkey, err = wraputil.LoadRSAKeyFromPEM(parentkeydata)
		if err != nil {
			return nil, nil, err
		}
	}
	if parentcertdata != nil {
		parentcert, err = wraputil.LoadX509CertFromPEM(parentcertdata)
		if err != nil {
			return nil, nil, err
		}
	}
	key, cert, err := GenerateKeypair(commonname, dns, ips, parentkey, parentcert)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}),
		nil
}

func GenerateKeypairToFiles(commonname string, dns []string, ips []net.IP, keyout string, certout string, parentkeyin string, parentcertin string) error {
	var parentkey []byte
	var parentcert []byte
	var err error
	if parentkeyin != "" {
		parentkey, err = ioutil.ReadFile(parentkeyin)
		if err != nil {
			return err
		}
	}
	if parentcertin != "" {
		parentcert, err = ioutil.ReadFile(parentcertin)
		if err != nil {
			return err
		}
	}
	key, cert, err := GenerateKeypairPEMs(commonname, dns, ips, parentkey, parentcert)
	if err != nil {
		return err
	}
	err = fileutil.EnsureIsFolder(path.Dir(keyout))
	if err != nil {
		return err
	}
	err = fileutil.EnsureIsFolder(path.Dir(certout))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(keyout, key, os.FileMode(0600))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(certout, cert, os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
}

func Copy(from string, to string, mode os.FileMode) error {
	data, err := ioutil.ReadFile(from)
	if err != nil {
		return err
	}
	err = fileutil.EnsureIsFolder(path.Dir(to))
	if err != nil {
		return err
	}
	return ioutil.WriteFile(to, data, mode)
}

func GenerateSSHKeypair(keyout string) error {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}
	keydata := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	err = ioutil.WriteFile(keyout, keydata, os.FileMode(0600))
	if err != nil {
		return err
	}
	pubkey, err := ssh.NewPublicKey(key.Public())
	if err != nil {
		return err
	}
	err = fileutil.EnsureIsFolder(path.Dir(keyout))
	if err != nil {
		return err
	}
	return ioutil.WriteFile(keyout+".pub", ssh.MarshalAuthorizedKey(pubkey), os.FileMode(0644))
}

func Setup() error {
	err := GenerateKeypairToFiles("localhost-cert", []string{"localhost"}, []net.IP{net.IPv4(127, 0, 0, 1)}, "server/authorities/server.key", "server/authorities/server.pem", "", "")
	if err != nil {
		return err
	}
	err = Copy("server/authorities/server.pem", "client/server.pem", os.FileMode(0644))
	if err != nil {
		return err
	}
	err = GenerateSSHKeypair("server/authorities/ssh_host_ca")
	if err != nil {
		return err
	}
	err = GenerateKeypairToFiles("etcd-ca", nil, nil, "server/authorities/etcd-client.key", "server/authorities/etcd-client.pem", "", "")
	if err != nil {
		return err
	}
	err = GenerateKeypairToFiles("grant-ca", nil, nil, "server/authorities/granting.key", "server/authorities/granting.pem", "", "")
	if err != nil {
		return err
	}
	err = GenerateKeypairToFiles("serviceaccount-ca", nil, nil, "server/authorities/serviceaccount.key", "server/authorities/serviceaccount.pem", "", "")
	if err != nil {
		return err
	}
	err = GenerateKeypairToFiles("admin-test", nil, nil, "admin/auth.key", "admin/auth.pem", "server/authorities/granting.key", "server/authorities/granting.pem")
	if err != nil {
		return err
	}
	err = GenerateSSHKeypair("client/ssh_host_rsa_key")
	if err != nil {
		return err
	}
	return nil
}

func ListRecursiveFiles(directory string) ([]string, error) {
	infos, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, info := range infos {
		if info.IsDir() {
			additional, err := ListRecursiveFiles(path.Join(directory, info.Name()))
			if err != nil {
				return nil, err
			}
			for _, x := range additional {
				out = append(out, path.Join(info.Name(), x))
			}
		} else {
			out = append(out, info.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func CheckRecursiveFiles(directory string, expected []string) error {
	contents, err := ListRecursiveFiles(directory)
	if err != nil {
		return err
	}
	// generate maps
	contentmap := map[string]bool{}
	for _, c := range contents {
		contentmap[c] = true
	}
	expectmap := map[string]bool{}
	for _, e := range expected {
		expectmap[e] = true
	}
	// check
	for _, e := range expected {
		if !contentmap[e] {
			return fmt.Errorf("did not find file: %s", e)
		}
	}
	for _, c := range contents {
		if !expectmap[c] {
			return fmt.Errorf("found extraneous file: %s", c)
		}
	}
	return nil
}

func CheckLog(logfile string, lines []string) error {
	filedata, err := ioutil.ReadFile(logfile)
	if err != nil {
		return err
	}
	content := strings.Split(string(filedata), "\n")
	if content[len(content)-1] == "" {
		content = content[:len(content)-1]
	}
	if len(content) != len(lines) {
		return errors.New("wrong number of lines")
	}
	for i, line := range content {
		lastpart := strings.SplitN(line, ": ", 2)[1]
		if lastpart != lines[i] {
			return fmt.Errorf("line mismatch: '%s' instead of '%s'", lastpart, lines[i])
		}
	}
	return nil
}

func CheckFile(logfile string, expected string) error {
	filedata, err := ioutil.ReadFile(logfile)
	if err != nil {
		return err
	}
	if string(filedata) != expected {
		return fmt.Errorf("mismatch on file content for %s: got '%s' instead of '%s'", logfile, string(filedata), expected)
	}
	return nil
}

func CheckSameFile(file1, file2 string) error {
	filedata1, err := ioutil.ReadFile(file1)
	if err != nil {
		return err
	}
	filedata2, err := ioutil.ReadFile(file2)
	if err != nil {
		return err
	}
	if !bytes.Equal(filedata1, filedata2) {
		return fmt.Errorf("mismatch on file content for %s <-> %s: got '%s' and '%s'", file1, file2, string(filedata1), string(filedata2))
	}
	return nil
}

func ValidateTLSCert(certfile string, keyfile string, authorityfile string, commonname string) error {
	certdata, err := ioutil.ReadFile(certfile)
	if err != nil {
		return err
	}
	cert, err := wraputil.LoadX509CertFromPEM(certdata)
	if err != nil {
		return err
	}
	authoritydata, err := ioutil.ReadFile(authorityfile)
	if err != nil {
		return err
	}
	authority, err := wraputil.LoadX509CertFromPEM(authoritydata)
	if err != nil {
		return err
	}
	keydata, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return err
	}
	key, err := wraputil.LoadRSAKeyFromPEM(keydata)
	if err != nil {
		return err
	}
	if !bytes.Equal(cert.RawIssuer, authority.RawSubject) {
		return errors.New("issuer/subject mismatch")
	}
	if cert.Subject.CommonName != commonname {
		return errors.New("commonname mismatch")
	}
	if cert.PublicKey.(*rsa.PublicKey).N.Cmp(key.N) != 0 {
		return errors.New("expected public keys to match")
	}
	return nil
}

func ValidateSSHCert(certfile string, keyfile string, authorityfile string, commonname string, hostname string) error {
	certdata, err := ioutil.ReadFile(certfile)
	if err != nil {
		return err
	}
	certgen, err := wraputil.ParseSSHTextPubkey(certdata)
	if err != nil {
		return err
	}
	cert := certgen.(*ssh.Certificate)
	authoritydata, err := ioutil.ReadFile(authorityfile)
	if err != nil {
		return err
	}
	authoritygen, err := wraputil.ParseSSHTextPubkey(authoritydata)
	if err != nil {
		return err
	}
	authority := authoritygen.(ssh.PublicKey)
	keydata, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return err
	}
	key, err := wraputil.LoadRSAKeyFromPEM(keydata)
	if err != nil {
		return err
	}

	checker := &ssh.CertChecker{
		IsHostAuthority: func(auth ssh.PublicKey, address string) bool {
			return auth.Type() == authority.Type() && bytes.Equal(auth.Marshal(), authority.Marshal())
		},
	}
	err = checker.CheckHostKey(hostname+":22", nil, cert)
	if err != nil {
		return err
	}
	if cert.KeyId != commonname {
		return errors.New("wrong keyid in ssh")
	}
	if !bytes.Equal(cert.SignatureKey.Marshal(), authority.Marshal()) {
		return errors.New("expected signer key to match")
	}
	n := cert.Key.(ssh.CryptoPublicKey).CryptoPublicKey().(*rsa.PublicKey).N
	if n.Cmp(key.N) != 0 {
		return fmt.Errorf("expected public key to match private key: %v, %v", n, key.N)
	}
	return nil
}

func Check() error {
	err := CheckRecursiveFiles("client", []string{
		"client.yaml",
		"client.log",
		"server.pem",
		"keyclient/granting.key",
		"keyclient/granting.pem",
		"authorities/etcd-client.pem",
		"ssh_host_ca.pub",
		"cluster.conf",
		"local.conf",
		"ssh_host_rsa_key",
		"ssh_host_rsa_key.pub",
		"ssh_host_rsa_key-cert.pub",
		"keys/etcd-client.key",
		"keys/etcd-client.pem",
		"serviceaccount.key",
		"serviceaccount.pem",
	})
	if err != nil {
		return err
	}

	err = CheckRecursiveFiles("server", []string{
		"server.yaml",
		"server.log",
		"static/cluster.conf",
		"authorities/server.key",
		"authorities/server.pem",
		"authorities/ssh_host_ca",
		"authorities/ssh_host_ca.pub",
		"authorities/etcd-client.key",
		"authorities/etcd-client.pem",
		"authorities/granting.key",
		"authorities/granting.pem",
		"authorities/serviceaccount.key",
		"authorities/serviceaccount.pem",
	})
	if err != nil {
		return err
	}

	err = CheckLog("client/client.log", []string{
		"keygranting cert not yet available: no keygranting certificate found",
		"action performed: generate key keyclient/granting.key (4096 bits)",
		"action performed: bootstrap with token API renew-keygrant from path bootstrap.token",
		"action performed: req/renew ssh-host from key ssh_host_rsa_key-cert.pub into cert ssh_host_rsa_key.pub with API 168h0m0s in advance by grant-ssh-host",
		"action performed: generate key keys/etcd-client.key (4096 bits)",
		"action performed: req/renew etcd-client from key keys/etcd-client.pem into cert keys/etcd-client.key with API 168h0m0s in advance by grant-etcd-client",
		"action performed: download to file authorities/etcd-client.pem (mode 644) every 24h0m0s: pubkey for authority etcd-client",
		"action performed: download to file ssh_host_ca.pub (mode 644) every 168h0m0s: pubkey for authority ssh-host",
		"action performed: download to file cluster.conf (mode 644) every 24h0m0s: static file cluster.conf",
		"action performed: download to file local.conf (mode 644) every 24h0m0s: result from api get-local-config",
		"action performed: download to file serviceaccount.pem (mode 644) every 24h0m0s: pubkey for authority serviceaccount",
		"action performed: download to file serviceaccount.key (mode 600) every 24h0m0s: result from api fetch-serviceaccount-key",
		"ACTLOOP STABILIZED",
	})
	if err != nil {
		return err
	}

	err = CheckLog("server/server.log", []string{
		"Attempting to perform API operation bootstrap for admin-test",
		"Operation bootstrap for admin-test succeeded.",
		"Attempting to perform API operation renew-keygrant for localhost-test",
		"Operation renew-keygrant for localhost-test succeeded.",
		"Attempting to perform API operation grant-ssh-host for localhost-test",
		"Operation grant-ssh-host for localhost-test succeeded.",
		"Attempting to perform API operation grant-etcd-client for localhost-test",
		"Operation grant-etcd-client for localhost-test succeeded.",
		"Attempting to perform API operation get-local-config for localhost-test",
		"Operation get-local-config for localhost-test succeeded.",
		"Attempting to perform API operation fetch-serviceaccount-key for localhost-test",
		"Operation fetch-serviceaccount-key for localhost-test succeeded.",
	})
	if err != nil {
		return err
	}

	err = CheckFile("client/cluster.conf", "description=this is a test configuration file\n")
	if err != nil {
		return err
	}

	err = CheckFile("client/local.conf",
		"# generated automatically by keyserver\n"+
			"HOST_NODE=localhost\n"+
			"HOST_DNS=localhost.mit.edu\n"+
			"HOST_IP=127.0.0.1\n"+
			"SCHEDULE_WORK=true\n")
	if err != nil {
		return err
	}

	err = CheckSameFile("client/authorities/etcd-client.pem", "server/authorities/etcd-client.pem")
	if err != nil {
		return err
	}
	err = CheckSameFile("client/ssh_host_ca.pub", "server/authorities/ssh_host_ca.pub")
	if err != nil {
		return err
	}
	err = CheckSameFile("client/serviceaccount.key", "server/authorities/serviceaccount.key")
	if err != nil {
		return err
	}
	err = CheckSameFile("client/serviceaccount.pem", "server/authorities/serviceaccount.pem")
	if err != nil {
		return err
	}
	err = ValidateTLSCert("client/keyclient/granting.pem", "client/keyclient/granting.key", "server/authorities/granting.pem", "localhost-test")
	if err != nil {
		return err
	}
	err = ValidateTLSCert("client/keys/etcd-client.pem", "client/keys/etcd-client.key", "server/authorities/etcd-client.pem", "etcd-client-localhost")
	if err != nil {
		return err
	}
	err = ValidateSSHCert("client/ssh_host_rsa_key-cert.pub", "client/ssh_host_rsa_key", "server/authorities/ssh_host_ca.pub", "admitted-localhost-test", "localhost.mit.edu")
	if err != nil {
		return err
	}

	return nil
}

func Main() error {
	if len(os.Args) < 2 {
		return errors.New("not enough parameters")
	}
	if os.Args[1] == "setup" {
		return Setup()
	} else if os.Args[1] == "check" {
		return Check()
	} else {
		return fmt.Errorf("no such command: %s", os.Args[1])
	}
}

func main() {
	err := Main()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("looks okay!")
}
