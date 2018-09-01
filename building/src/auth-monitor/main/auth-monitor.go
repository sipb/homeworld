package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"os"
	"strings"
	"io/ioutil"
	"os/exec"
	"keysystem/keycommon"
	"keysystem/keycommon/server"
	"util/osutil"
	"keysystem/keycommon/reqtarget"
	"crypto/tls"
	"golang.org/x/crypto/ssh"
	"crypto/rsa"
	"crypto/rand"
	"bytes"
)

var (
	registry = prometheus.NewRegistry()

	fetchCheck = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "keysystem",
		Name: "fetch_static_check",
		Help: "Check for whether static files can be fetched",
	})

	keyCheck = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "keysystem",
		Name: "fetch_authority_check",
		Help: "Check for whether authority pubkeys can be fetched",
	})

	authCheck = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "keysystem",
		Name: "gateway_auth_check",
		Help: "Check for whether the keygateway accepts authentication",
	})

	grantCheck = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "keysystem",
		Name: "ssh_grant_check",
		Help: "Check for whether ssh certs can be granted",
	})

	sshCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "keysystem",
		Name: "ssh_access_check",
		Help: "Check for whether servers can be accessed over ssh",
	}, []string {"server"})
)

func attemptSSHAccess(pkey *rsa.PrivateKey, cert *ssh.Certificate, host_ca ssh.PublicKey, machine string) error {
	rawSigner, err := ssh.NewSignerFromKey(pkey)
	if err != nil {
		return err
	}
	signer, err := ssh.NewCertSigner(cert, rawSigner)
	if err != nil {
		return err
	}
	checker := &ssh.CertChecker{
		IsHostAuthority: func(auth ssh.PublicKey, address string) bool {
			return auth.Type() == host_ca.Type() && bytes.Equal(auth.Marshal(), host_ca.Marshal())
		},
	}
	client, err := ssh.Dial("tcp", machine + ":22", &ssh.ClientConfig{
		Timeout: time.Second * 10,
		User: "root",
		HostKeyAlgorithms: []string {ssh.CertAlgoRSAv01},
		HostKeyCallback: checker.CheckHostKey,
		Auth: []ssh.AuthMethod {ssh.PublicKeys(signer)},
	})
	if err != nil {
		return err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	result, err := session.Output("echo 'bidirectionality test'")
	if err != nil {
		return err
	}
	if string(result) != "bidirectionality test\n" {
		return fmt.Errorf("incorrect ssh echo result: '%s'", string(result))
	}
	return nil
}

func attemptSSHAcquire(keyserver *server.Keyserver, config keycommon.Config) (*rsa.PrivateKey, *ssh.Certificate, error) {
	keypair, err := tls.LoadX509KeyPair(config.CertPath, config.KeyPath)
	if err != nil {
		return nil, nil, err
	}

	privkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	pubkey, err := ssh.NewPublicKey(privkey.Public())
	if err != nil {
		return nil, nil, err
	}

	rt, err := keyserver.AuthenticateWithCert(keypair)
	if err != nil {
		return nil, nil, err
	}

	rt, err = reqtarget.Impersonate(rt, "auth-to-kerberos", "metrics@NONEXISTENT.REALM.INVALID")
	if err != nil {
		return nil, nil, err
	}

	resp, err := reqtarget.SendRequest(rt, "access-ssh", string(ssh.MarshalAuthorizedKey(pubkey)))
	if err != nil {
		return nil, nil, err
	}
	if resp == "" {
		return nil, nil, fmt.Errorf("empty ssh cert")
	}

	pkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(resp))
	cert, ok := pkey.(*ssh.Certificate)
	if !ok {
		return nil, nil, fmt.Errorf("not a ssh cert")
	}

	return privkey, cert, nil
}

func attemptKAuth(keyserver *server.Keyserver, config keycommon.Config) error {
	f, err := ioutil.TempFile("", "krb5cc_keygateway_checker_")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	err = f.Close()
	if err != nil {
		return err
	}

	host := strings.Split(config.Keyserver, ":")[0]  // using keyserver's definition to refer to this node

	cmd := exec.Command("kinit", "-l2m", "-k", "host/" + host)
	cmd.Env = osutil.ModifiedEnviron("KRB5CCNAME", f.Name())
	err = cmd.Run()
	if err != nil {
		return err
	}

	contents, err := ioutil.ReadFile(f.Name())
	if err != nil {
		return err
	}
	if len(contents) == 0 {
		return fmt.Errorf("expected ticket cache to be populated")
	}

	rt, err := keyserver.AuthenticateWithKerberosTicketsInCache(f.Name())
	if err != nil {
		return err
	}

	_, err = rt.SendRequests([]reqtarget.Request{})
	if err != nil {
		return err
	}
	return nil
}

func cycle(keyserver *server.Keyserver, config keycommon.Config) {
	// basic functionality testing
	host_ca_pub, err := keyserver.GetPubkey("ssh-host")
	if err != nil {
		keyCheck.Set(0)
		fetchCheck.Set(0)
		authCheck.Set(0)
		grantCheck.Set(0)
		sshCheck.Reset()
		log.Printf("failed fetch of keyserver authority: %v", err)
		return
	} else {
		keyCheck.Set(1)
	}

	machines, err := keyserver.GetStatic("machine.list")
	if err != nil {
		fetchCheck.Set(0)
		authCheck.Set(0)
		grantCheck.Set(0)
		sshCheck.Reset()
		log.Printf("failed fetch of keyserver static: %v", err)
		return
	} else {
		fetchCheck.Set(1)
	}

	// checking kerberos authentication
	err = attemptKAuth(keyserver, config)
	if err != nil {
		authCheck.Set(0)
		log.Printf("failed verification of keygateway connection: %v", err)
	} else {
		authCheck.Set(1)
	}

	// checking SSH authentication
	machine_list := strings.Split(strings.TrimSpace(string(machines)), ",")

	pkey, cert, err := attemptSSHAcquire(keyserver, config)
	if err != nil {
		grantCheck.Set(0)
		for _, machine := range machine_list {
			sshCheck.With(prometheus.Labels{
				"server": machine,
			}).Set(0)
		}
		log.Printf("failed SSH grant check: %v", err)
	} else {
		grantCheck.Set(1)

		host_ca, _, _, _, err := ssh.ParseAuthorizedKey(host_ca_pub)
		if err != nil {
			for _, machine := range machine_list {
				sshCheck.With(prometheus.Labels{
					"server": machine,
				}).Set(0)
			}
			log.Printf("failed to decode host_ca pubkey")
		} else {
			for _, machine := range machine_list {
				err := attemptSSHAccess(pkey, cert, host_ca, machine)
				gauge := sshCheck.With(prometheus.Labels{
					"server": machine,
				})
				if err != nil {
					log.Printf("failed SSH access check for %s: %v", machine, err)
					gauge.Set(0)
				} else {
					gauge.Set(1)
				}
			}
		}
	}
}

func loop(keyserver *server.Keyserver, config keycommon.Config, stopChannel <-chan struct{}) {
	for {
		next_cycle_at := time.Now().Add(time.Second * 15)
		cycle(keyserver, config)

		delta := next_cycle_at.Sub(time.Now())
		if delta < time.Second {
			delta = time.Second
		}

		select {
		case <-stopChannel:
			break
		case <-time.After(delta):
		}
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("expected one argument: configpath")
	}

	ks, config, err := keycommon.LoadKeyserver(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	registry.MustRegister(fetchCheck)
	registry.MustRegister(keyCheck)
	registry.MustRegister(authCheck)
	registry.MustRegister(grantCheck)
	registry.MustRegister(sshCheck)

	stopChannel := make(chan struct{})
	defer close(stopChannel)
	go loop(ks, config, stopChannel)

	address := ":9102"

	log.Printf("Starting metrics server on: %v", address)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	err = http.ListenAndServe(address, nil)
	log.Printf("Stopped metrics server: %v", err)
}
