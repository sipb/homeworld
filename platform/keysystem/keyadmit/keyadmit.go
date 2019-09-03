package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/keysystem/api/server"
	"github.com/sipb/homeworld/platform/keysystem/keyserver/admit"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig"
)

func ListRequests(rt reqtarget.RequestTarget) (requests map[string]admit.AdmitState, err error) {
	response, err := reqtarget.SendRequest(rt, worldconfig.ListAdmitRequestsAPI, "")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(response), &requests)
	if err != nil {
		return nil, err
	}
	return requests, nil
}

func GetPendingRequest(rt reqtarget.RequestTarget, skip map[string]bool) (string, admit.AdmitState, int, error) {
	reqs, err := ListRequests(rt)
	if err != nil {
		return "", admit.AdmitState{}, 0, err
	}
	for fingerprint, admitState := range reqs {
		if fingerprint == "" {
			return "", admit.AdmitState{}, 0, errors.New("empty fingerprint found")
		}
		if skip[fingerprint] {
			continue
		}
		if admitState.Seen && (!admitState.Approved || admitState.ApprovedPrincipal != admitState.SeenPrincipal) {
			return fingerprint, admitState, len(reqs), nil
		}
	}
	return "", admit.AdmitState{}, len(reqs), nil
}

func Approve(rt reqtarget.RequestTarget, fingerprint string, principal string) error {
	approval := admit.AdmitApproval{
		Principal:   principal,
		Fingerprint: fingerprint,
	}
	data, err := json.Marshal(approval)
	if err != nil {
		return err
	}
	response, err := reqtarget.SendRequest(rt, worldconfig.ApproveAdmitAPI, string(data))
	if err != nil {
		return err
	}
	if response != "" {
		return errors.New("expected no response from approval API")
	}
	return nil
}

func Ask(reader *bufio.Reader, question string) (bool, error) {
	for {
		_, err := fmt.Printf("%s (y/n) ", question)
		if err != nil {
			return false, err
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "y" {
			return true, nil
		}
		if line == "n" {
			return false, nil
		}
	}
}

func Authenticate(authorityPath string, keyserverAddress string) (reqtarget.RequestTarget, error) {
	authoritydata, err := ioutil.ReadFile(authorityPath)
	if err != nil {
		return nil, err
	}
	ks, err := server.NewKeyserver(authoritydata, keyserverAddress)
	if err != nil {
		return nil, err
	}
	rt, err := ks.AuthenticateWithKerberosTickets()
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func main() {
	logger := log.New(os.Stderr, "[keyadmit] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	if len(os.Args) != 3 {
		logger.Print("usage: keyadmit <authority.pem> <keyserver-address>")
	}
	in := bufio.NewReader(os.Stdin)

	rt, err := Authenticate(os.Args[1], os.Args[2])
	if err != nil {
		logger.Fatal(err)
	}

	skipping := map[string]bool{}

	// interactive loop
	for {
		fpr, req, n, err := GetPendingRequest(rt, skipping)
		if err != nil {
			logger.Fatal(err)
		}
		if fpr == "" {
			fmt.Printf("no pending requests (%d total)\n", n)
			check, err := Ask(in, "check again?")
			if err != nil {
				logger.Fatal(err)
			}
			if check {
				continue
			} else {
				break
			}
		}
		fmt.Printf("found pending request (%d total)\n", n)
		fmt.Printf("  PRINCIPAL: %s\n", req.SeenPrincipal)
		if req.Approved {
			fmt.Printf("  PREVIOUS APPROVAL WAS: %s\n", req.ApprovedPrincipal)
		} else {
			fmt.Printf("  (not yet approved)\n")
		}
		fmt.Printf("  SEEN AT: %s\n", req.SeenAt.String())
		fmt.Println("  FINGERPRINT:")
		fmt.Println(admit.WrappedFingerprint(fpr))
		fmt.Println("  Please validate the entire fingerprint shown against the server's physical screen,")
		fmt.Println("  and please confirm that the principal is as expected for that particular server.")
		ok, err := Ask(in, "approve request?")
		if err != nil {
			logger.Fatal(err)
		}
		if ok {
			err := Approve(rt, fpr, req.SeenPrincipal)
			if err != nil {
				logger.Fatal(err)
			}
		} else {
			fmt.Println("skipping fingerprint")
			skipping[fpr] = true
		}
	}
}
