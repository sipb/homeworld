package main

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func refetch(image string) (time_taken float64, err error) {
	// if the image isn't found, still returns exit code zero
	cmd := exec.Command("rkt", "image", "rm", image)
	if err := cmd.Run(); err != nil {
		return 0, errors.Wrap(err, "failed to remove previous image")
	}
	cmd = exec.Command("rkt", "fetch", "docker://"+image, "--full", "--insecure-options=image")
	time_start := time.Now()
	err = cmd.Run()
	time_end := time.Now()
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch new image")
	}
	time_taken = time_end.Sub(time_start).Seconds()
	return time_taken, nil
}

func attemptEcho(image string) (float64, error) {
	echo_data := fmt.Sprintf("!%d!", rand.Uint64())
	cmd := exec.Command("rkt", "run", image, "--", echo_data)
	time_rkt_start := time.Now()
	output, err := cmd.Output()
	time_rkt_end := time.Now()
	if err != nil {
		return 0, errors.Wrap(err, "failed to exec new image")
	}
	lines := strings.Split(strings.Trim(string(output), "\000\r\n"), "\n")
	if strings.Contains(lines[len(lines)-1], "rkt: obligatory restart") && len(lines) > 1 {
		lines = lines[:len(lines)-1]
	}
	parts := strings.SplitN(strings.Trim(lines[len(lines)-1], "\000\r\n"), ": ", 2)
	if !strings.Contains(parts[0], "] pullcheck[") || len(parts) != 2 {
		return 0, errors.Errorf("output from rkt did not match expected format: '%s'", string(output))
	}
	if parts[1] != fmt.Sprintf("hello container world [%s]", echo_data) {
		return 0, errors.Errorf("output from rkt did not match expectation: '%s' (%v) instead of '%s'", parts[1], []byte(parts[1]), echo_data)
	}
	return time_rkt_end.Sub(time_rkt_start).Seconds(), nil
}
