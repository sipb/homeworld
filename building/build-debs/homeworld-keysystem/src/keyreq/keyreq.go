package main

import (
	"log"
	"os"
	"keycommon"
	"os/user"
	"path"
	"keycommon/reqtarget"
	"io/ioutil"
	"keycommon/server"
	"strings"
	"golang.org/x/crypto/ssh"
	"encoding/base64"
	"fmt"
	"errors"
	"crypto/rsa"
	"crypto/rand"
	"encoding/pem"
	"crypto/x509"
	"util/csrutil"
)

func usage(logger *log.Logger) {
	logger.Fatal("invalid usage\nusage: keyreq ssh\n  request a SSH cert for ~/.ssh/id_rsa.pub, placing it in ~/.ssh/id_rsa-cert.pub\n" +
		"usage: keyreq ssh-host\n  place the @ca-certificates directive into ~/.ssh/known_hosts\n" +
		"usage: keyreq kube\n  generate a key and request a kubernetes auth cert, placing them in ~/.homeworld/kube.{pem,key}\n" +
		"  (also, put the kubernetes CA cert in ~/.homeworld/kube-ca.pem)\n" +
		"usage: keyreq etcd\n  generate a key and request an etcd auth cert, placing them in ~/.homeworld/etcd.{pem,key}\n" +
		"  (also, put the etcd CA cert in ~/.homeworld/etcd-ca.pem)\n" +
		"usage: keyreq bootstrap <server-principal>\n  request a bootstrap token for a server\n\n" +
	    "these depend on ~/.homeworld/keyreq.yaml being configured properly\n")
}

func homedir(logger *log.Logger) string {
	usr, err := user.Current()
	if err != nil {
		logger.Fatal(err)
	}
	return usr.HomeDir
}

func get_keyserver(logger *log.Logger) *server.Keyserver {
	ks, _, err := keycommon.LoadKeyserver(path.Join(homedir(logger), ".homeworld", "keyreq.yaml"))
	if err != nil {
		logger.Fatal(err)
	}
	return ks
}

func auth_kerberos(logger *log.Logger) (*server.Keyserver, reqtarget.RequestTarget) {
	ks := get_keyserver(logger)
	rt, err := ks.AuthenticateWithKerberosTickets()
	if err != nil {
		logger.Fatal(err)
	}
	// confirm that connection works
	_, err = rt.SendRequests([]reqtarget.Request{})
	if err != nil {
		logger.Fatal("failed to check connection: ", err)
	}
	return ks, rt
}

func replace_cert_authority(hostlines []string, machs string, pubkey []byte) ([]string, error) {
	remove_index := -1
	for i, line := range hostlines {
		marker, _, _, comment, _, err := ssh.ParseKnownHosts([]byte(line))
		if err != nil {
			continue
		}
		if marker != "cert-authority" {
			continue
		}
		if comment != "homeworld-keydef" {
			continue
		}
		remove_index = i
	}
	if remove_index != -1 {
		hostlines = append(hostlines[:remove_index], hostlines[remove_index+1:]...)
	}
	public, _, _, _, err := ssh.ParseAuthorizedKey(pubkey)
	if err != nil {
		return nil, err
	}
	if hostlines[len(hostlines) - 1] == "" {
		hostlines = hostlines[:len(hostlines) - 1]
	}
	if strings.Contains(machs, "\n") || strings.Contains(machs, " ") {
		return nil, errors.New("bad spacing on machine list")
	}
	entry := fmt.Sprintf("@cert-authority %s %s %s homeworld-keydef", machs, public.Type(), base64.StdEncoding.EncodeToString(public.Marshal()))
	hostlines = append(hostlines, entry, "")
	return hostlines, nil
}

func main() {
	logger := log.New(os.Stderr, "[keyreq] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) < 2 {
		usage(logger)
		return
	}
	switch os.Args[1] {
	case "ssh":
		_, rt := auth_kerberos(logger)
		id_rsa_pub, err := ioutil.ReadFile(path.Join(homedir(logger), ".ssh", "id_rsa.pub"))
		if err != nil {
			logger.Fatal(err)
		}
		req, err := reqtarget.SendRequest(rt, "access-ssh", string(id_rsa_pub))
		if err != nil {
			logger.Fatal(err)
		}
		if req == "" {
			logger.Fatal("empty result")
		}
		err = ioutil.WriteFile(path.Join(homedir(logger), ".ssh", "id_rsa-cert.pub"), []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
	case "ssh-host":
		ks := get_keyserver(logger)
		pubkey, err := ks.GetPubkey("ssh-host")
		if err != nil {
			logger.Fatal(err)
		}
		machines, err := ks.GetStatic("machine.list")
		if err != nil {
			logger.Fatal(err)
		}
		machs := strings.TrimSpace(string(machines))
		logger.Println("Machines:", string(machines))

		known_hosts := path.Join(homedir(logger), ".ssh", "known_hosts")
		hosts, err := ioutil.ReadFile(known_hosts)
		if err != nil {
			if os.IsNotExist(err) {
				hosts = nil
			} else {
				logger.Fatal(err)
			}
		}
		new_hosts, err := replace_cert_authority(strings.Split(string(hosts), "\n"), machs, pubkey)
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(known_hosts, []byte(strings.Join(new_hosts, "\n")), os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
	case "kube":
		ks, rt := auth_kerberos(logger)
		pkey, err := rsa.GenerateKey(rand.Reader, 2048) // smaller key sizes are okay, because these are limited to a short period
		if err != nil {
			logger.Fatal(err)
		}
		privkey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pkey)})
		csr, err := csrutil.BuildTLSCSR(privkey)
		if err != nil {
			logger.Fatal(err)
		}
		req, err := reqtarget.SendRequest(rt, "access-kubernetes", string(csr))
		if err != nil {
			logger.Fatal(err)
		}
		if req == "" {
			logger.Fatal("empty result")
		}
		err = ioutil.WriteFile(path.Join(homedir(logger), ".homeworld", "kube.key"), privkey, os.FileMode(0600))
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(path.Join(homedir(logger), ".homeworld", "kube.pem"), []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
		ca, err := ks.GetPubkey("kubernetes")
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(path.Join(homedir(logger), ".homeworld", "kube-ca.pem"), ca, os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
	case "etcd":
		// TODO: deduplicate code
		ks, rt := auth_kerberos(logger)
		pkey, err := rsa.GenerateKey(rand.Reader, 2048) // smaller key sizes are okay, because these are limited to a short period
		if err != nil {
			logger.Fatal(err)
		}
		privkey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(pkey)})
		csr, err := csrutil.BuildTLSCSR(privkey)
		if err != nil {
			logger.Fatal(err)
		}
		req, err := reqtarget.SendRequest(rt, "access-etcd", string(csr))
		if err != nil {
			logger.Fatal(err)
		}
		if req == "" {
			logger.Fatal("empty result")
		}
		err = ioutil.WriteFile(path.Join(homedir(logger), ".homeworld", "etcd.key"), privkey, os.FileMode(0600))
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(path.Join(homedir(logger), ".homeworld", "etcd.pem"), []byte(req), os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
		ca, err := ks.GetPubkey("etcd-server")
		if err != nil {
			logger.Fatal(err)
		}
		err = ioutil.WriteFile(path.Join(homedir(logger), ".homeworld", "etcd-ca.pem"), ca, os.FileMode(0644))
		if err != nil {
			logger.Fatal(err)
		}
	case "bootstrap":
		if len(os.Args) < 3 {
			logger.Fatal("bootstrap requires a principal")
			return
		}
		_, rt := auth_kerberos(logger)
		token, err := reqtarget.SendRequest(rt, "bootstrap", os.Args[2])
		if err != nil {
			logger.Fatal(err)
		}
		os.Stdout.WriteString(token + "\n")
	default:
		usage(logger)
	}
}
