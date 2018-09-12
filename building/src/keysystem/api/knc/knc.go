package knc

import (
	"encoding/json"
	"errors"
	"fmt"
	"keysystem/api/reqtarget"
	"os/exec"
	"util/osutil"
)

type KncServer struct {
	Hostname            string
	KerberosTicketCache string
}

func (k KncServer) kncRequest(data []byte) ([]byte, error) {
	cmd := exec.Command("/usr/bin/knc", fmt.Sprintf("host@%s", k.Hostname), "20575")

	if k.KerberosTicketCache != "" {
		cmd.Env = osutil.ModifiedEnviron("KRB5CCNAME", k.KerberosTicketCache)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	go func() {
		defer stdin.Close()
		stdin.Write(data)
	}()

	response, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			err = fmt.Errorf("%s\n--- knc stderr start ---\n%s\n--- knc stderr end ---\n", ee.Error(), string(ee.Stderr))
		}
		return nil, err
	}

	return response, nil
}

func (k KncServer) SendRequests(reqs []reqtarget.Request) ([]string, error) {
	raw_reqs, err := json.Marshal(reqs)
	if err != nil {
		return nil, fmt.Errorf("while packing json: %s", err)
	}

	raw_resps, err := k.kncRequest(raw_reqs)
	if err != nil {
		return nil, fmt.Errorf("while performing request: %s", err)
	}

	if len(raw_resps) == 0 {
		return nil, fmt.Errorf("empty response, likely because the server does not recognize your Kerberos identity")
	}

	resps := []string{}
	err = json.Unmarshal(raw_resps, &resps)
	if err != nil {
		return nil, fmt.Errorf("while unpacking json: %s ('%s' -> '%s')", err, raw_reqs, raw_resps)
	}

	if len(resps) != len(reqs) {
		return nil, errors.New("Wrong number of results")
	}

	return resps, nil
}
