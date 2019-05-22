package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sipb/homeworld/platform/keysystem/api"
	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/util/osutil"
)

var (
	registry = prometheus.NewRegistry()

	fetchCheck = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "keysystem",
		Name:      "fetch_static_check",
		Help:      "Check for whether static files can be fetched",
	})

	keyCheck = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "keysystem",
		Name:      "fetch_authority_check",
		Help:      "Check for whether authority pubkeys can be fetched",
	})

	authCheck = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "keysystem",
		Name:      "gateway_auth_check",
		Help:      "Check for whether the keygateway accepts authentication",
	})

	grantCheck = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "keysystem",
		Name:      "ssh_grant_check",
		Help:      "Check for whether ssh certs can be granted",
	})

	sshCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "keysystem",
		Name:      "ssh_access_check",
		Help:      "Check for whether servers can be accessed over ssh",
	}, []string{"server"})
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
	client, err := ssh.Dial("tcp", machine+":22", &ssh.ClientConfig{
		Timeout:           time.Second * 10,
		User:              "root",
		HostKeyAlgorithms: []string{ssh.CertAlgoRSAv01},
		HostKeyCallback:   checker.CheckHostKey,
		Auth:              []ssh.AuthMethod{ssh.PublicKeys(signer)},
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

func attemptSSHAcquire(keyserver *server.Keyserver) (*rsa.PrivateKey, *ssh.Certificate, error) {
	keypair, err := tls.LoadX509KeyPair(paths.GrantingCertPath, paths.GrantingKeyPath)
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

	rt, err = reqtarget.Impersonate(rt, worldconfig.ImpersonateKerberosAPI, "metrics@NONEXISTENT.REALM.INVALID")
	if err != nil {
		return nil, nil, err
	}

	resp, err := reqtarget.SendRequest(rt, worldconfig.AccessSSHAPI, string(ssh.MarshalAuthorizedKey(pubkey)))
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

func attemptKAuth(keyserver *server.Keyserver) error {
	f, err := ioutil.TempFile("", "krb5cc_keygateway_checker_")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	err = f.Close()
	if err != nil {
		return err
	}

	hostport, err := paths.GetKeyserver()
	if err != nil {
		return err
	}
	host := strings.Split(hostport, ":")[0]

	cmd := exec.Command("kinit", "-l2m", "-k", "host/"+host)
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

func cycle(keyserver *server.Keyserver) {
	// basic functionality testing
	host_ca_pub, err := keyserver.GetPubkey(worldconfig.SSHHostAuthority)
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

	// just checking to make sure we can fetch statics
	clusterconf, err := keyserver.GetStatic(worldconfig.ClusterConfStatic)
	if err != nil {
		fetchCheck.Set(0)
		log.Printf("failed fetch of keyserver static: %v", err)
	} else {
		expected, err := ioutil.ReadFile(worldconfig.ClusterConfigPath)
		if err != nil {
			fetchCheck.Set(0)
			log.Printf("failed to load config for static comparison: %v", err)
		} else if !bytes.Equal(clusterconf, expected) {
			fetchCheck.Set(0)
			log.Printf("mismatch between static cluster config (%d bytes) and expected cluster config (%d bytes)", len(clusterconf), len(expected))
		} else {
			fetchCheck.Set(1)
		}
	}

	// checking kerberos authentication
	err = attemptKAuth(keyserver)
	if err != nil {
		authCheck.Set(0)
		log.Printf("failed verification of keygateway connection: %v", err)
	} else {
		authCheck.Set(1)
	}

	// checking SSH authentication
	setupconfig, err := worldconfig.LoadSpireSetup(paths.SpireSetupPath)
	if err != nil {
		grantCheck.Set(0)
		sshCheck.Reset()
		log.Printf("failed to parse list of nodes: %v", err)
		return
	}

	pkey, cert, err := attemptSSHAcquire(keyserver)
	if err != nil {
		grantCheck.Set(0)
		for _, machine := range setupconfig.Nodes {
			sshCheck.With(prometheus.Labels{
				"server": machine.DNS(),
			}).Set(0)
		}
		log.Printf("failed SSH grant check: %v", err)
	} else {
		grantCheck.Set(1)

		host_ca, _, _, _, err := ssh.ParseAuthorizedKey(host_ca_pub)
		if err != nil {
			for _, machine := range setupconfig.Nodes {
				sshCheck.With(prometheus.Labels{
					"server": machine.DNS(),
				}).Set(0)
			}
			log.Printf("failed to decode host_ca pubkey")
		} else {
			for _, machine := range setupconfig.Nodes {
				err := attemptSSHAccess(pkey, cert, host_ca, machine.DNS())
				gauge := sshCheck.With(prometheus.Labels{
					"server": machine.DNS(),
				})
				if err != nil {
					log.Printf("failed SSH access check for %s: %v", machine.DNS(), err)
					gauge.Set(0)
				} else {
					gauge.Set(1)
				}
			}
		}
	}
}

func loop(keyserver *server.Keyserver, stopChannel <-chan struct{}) {
	for {
		next_cycle_at := time.Now().Add(time.Second * 15)
		cycle(keyserver)

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
	ks, err := api.LoadDefaultKeyserver()
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
	go loop(ks, stopChannel)

	address := ":9102"

	log.Printf("Starting metrics server on: %v", address)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	err = http.ListenAndServe(address, nil)
	log.Printf("Stopped metrics server: %v", err)
}
