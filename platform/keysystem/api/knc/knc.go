package knc

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"os/exec"

	"github.com/sipb/homeworld/platform/keysystem/api/reqtarget"
	"github.com/sipb/homeworld/platform/util/osutil"
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
		return nil, errors.Wrap(err, "while packing json")
	}

	raw_resps, err := k.kncRequest(raw_reqs)
	if err != nil {
		return nil, errors.Wrap(err, "while performing request")
	}

	if len(raw_resps) == 0 {
		return nil, errors.New("empty response, likely because the server does not recognize your Kerberos identity")
	}

	resps := []string{}
	err = json.Unmarshal(raw_resps, &resps)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("while unpacking json ('%s' -> '%s')", raw_reqs, raw_resps))
	}

	if len(resps) != len(reqs) {
		return nil, errors.New("wrong number of results")
	}

	return resps, nil
}
