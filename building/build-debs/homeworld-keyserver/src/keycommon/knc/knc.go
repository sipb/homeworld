package knc

import (
	"fmt"
	"os/exec"
)

func Request(host string, data []byte) ([]byte, error) {
	cmd := exec.Command("/usr/bin/knc", fmt.Sprintf("host@%s", host), "20575")

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
		return nil, err
	}

	return response, nil
}
