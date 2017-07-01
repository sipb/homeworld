package main 

import (
	"os"
	"fmt"
	"log"
	"time"
	"bytes"
	"io/ioutil"
	"encoding/base64"
	"crypto/rand"
	"golang.org/x/crypto/ssh"
)

func main() {
	sshClient := os.Getenv("SSH_CONNECTION")
	sshPid := os.Getppid()

	if sshClient == "" { log.Fatal("SSH_CONNECTION is not available") }

	rawPubkey, err := ioutil.ReadAll(os.Stdin)
	if err != nil { log.Fatal(err) }
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(rawPubkey)
	if err != nil { log.Fatal(err) }

	nonce := make([]byte, 32)
	_, err = rand.Read(nonce)
	if err != nil { log.Fatal(err) }
	cert := &ssh.Certificate{
		Key: pubkey,
		KeyId: fmt.Sprintf("issued:%s.%d@%d", sshClient, sshPid, uint64(time.Now().Unix())),
		CertType: ssh.UserCert,
		ValidAfter: uint64(time.Now().Unix()),
		ValidBefore: uint64(time.Now().Add(4*time.Hour).Unix()),
		ValidPrincipals: []string{"root"},
		Permissions: ssh.Permissions{
			Extensions: map[string]string{
				"permit-X11-forwarding": "",
				"permit-agent-forwarding": "",
				"permit-port-forwarding": "",
				"permit-pty": "",
				"permit-user-rc": "",
			},
		},
		Nonce: nonce,
	}

	caKeyFile, err := os.Open("ca_key")
	if err != nil { log.Fatal(err) }
	rawCaKey, err := ioutil.ReadAll(caKeyFile)
	if err != nil { log.Fatal(err) }
	caSigner, err := ssh.ParsePrivateKey(rawCaKey)
	if err != nil { log.Fatal(err) }

	err = cert.SignCert(rand.Reader, caSigner)
	if err != nil { log.Fatal(err) }

	os.Stdout.Write(MarshalCert(cert))
}

// MarshalCert serializes cert for on-disk storage.  The return value
// ends with newline.
func MarshalCert(cert *ssh.Certificate) []byte {
	b := &bytes.Buffer{}
	b.WriteString(cert.Type())
	b.WriteByte(' ')
	e := base64.NewEncoder(base64.StdEncoding, b)
	e.Write(cert.Marshal())
	e.Close()
	b.WriteByte('\n')
	return b.Bytes()
}
