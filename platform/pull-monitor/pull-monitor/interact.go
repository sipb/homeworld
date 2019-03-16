package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// returns whether or not a particular image exists according to CRI-O. also returns false on error.
func doesImageExist(image string) bool {
	panic("todo")
}

// deletes an image from CRI-O's image store
func deleteImage(image string) error {
	panic("todo")
}

// pulls an image and saves it into CRI-O's image store
func pullImage(image string) error {
	panic("todo")
}

// attempts to delete and re-pull an image, to test whether image pulling is functional
// (and to set up for image execution testing now that the image is present)
func refetch(image string) (time_taken float64, err error) {
	if doesImageExist(image) {
		if err := deleteImage(image); err != nil {
			return 0, errors.Wrap(err, "failed to remove previous image")
		}
	}
	time_start := time.Now()
	err = pullImage(image)
	time_end := time.Now()
	if err != nil {
		return 0, errors.Wrap(err, "failed to fetch new image")
	}
	if !doesImageExist(image) {
		return 0, errors.New("pulled image, yet it does not exist")
	}
	time_taken = time_end.Sub(time_start).Seconds()
	return time_taken, nil
}

// instruct CRI-O to create a new Pod Sandbox in which we can launch our container
// returns the Pod ID of the created sandbox, if no error occurs
func startPodSandbox() (id string, err error) {
	panic("todo")
}

// instruct CRI-O to tear down an existing Pod Sandbox
func deletePodSandbox(podid string) error {
	panic("todo")
}

// instruct CRI-O to create (but not start) a container within an existing Pod Sandbox, by ID
// returns the Container ID of the created container, if no error occurs
func createContainer(image string, podid string, args []string) (containerid string, err error) {
	panic("todo")
}

// instruct CRI-O to start a container, by ID, that has already been created
func startContainer(containerid string) error {
	panic("todo")
}

// ask CRI-O whether a particular container, referenced by ID, is running or not
func isContainerRunning(containerid string) (bool, error) {
	panic("todo")
}

// retrieve the container logs from CRI-O for a particular container, by ID
// returns the retrieved logs if no error
func getContainerLogs(containerid string) (result string, err error) {
	panic("todo")
}

// inside a specified Pod Sandbox (by ID), create and start a new container with particular arguments, then wait for it
// to complete and return the contents of its logging output, if no error occurs
// the arguments must include the name of the executable to invoke
func runContainer(image string, podid string, args []string) (result string, err error) {
	containerid, err := createContainer(image, podid, args)
	if err != nil {
		return "", errors.Wrap(err, "creating container")
	}
	err = startContainer(containerid)
	if err != nil {
		return "", errors.Wrap(err, "starting container")
	}
	// now we wait for it to complete
	running := true
	for running {
		running, err = isContainerRunning(containerid)
		if err != nil {
			return "", errors.Wrap(err, "monitoring container")
		}
	}
	logs, err := getContainerLogs(containerid)
	if err != nil {
		return "", errors.Wrap(err, "read logs from container")
	}
	return logs, nil
}

// launch a container and its surrounding pod sandbox (with specified arguments), wait for it to finish
// the arguments must include the name of the program to launch within the container
// return the contents of its logging output, or an error
func runPod(image string, args ...string) (result string, err error) {
	podid, err := startPodSandbox()
	if err != nil {
		return "", errors.Wrap(err, "starting pod sandbox")
	}
	result, err = runContainer(image, podid, args)
	if err != nil {
		err = errors.Wrap(err, "running container")
	}
	err2 := deletePodSandbox(podid)
	if err2 != nil {
		err = multierror.Append(err, errors.Wrap(err2, "deleting pod sandbox"))
	}
	return result, err
}

// run the pullcheck container and confirm that it is launched correctly by passing a token through its arguments and
// receiving it through the logging output
func attemptEcho(image string) (float64, error) {
	echo_data := fmt.Sprintf("!%d!", rand.Uint64())
	time_rkt_start := time.Now()
	output, err := runPod(image, "/usr/bin/pullcheck", echo_data)
	time_rkt_end := time.Now()
	if err != nil {
		return 0, errors.Wrap(err, "failed to exec new image")
	}
	if strings.Trim(output, "\n") != fmt.Sprintf("hello container world [%s]", echo_data) {
		return 0, errors.Errorf("mismatched output from container: '%s' instead of '%s'", output, echo_data)
	}
	return time_rkt_end.Sub(time_rkt_start).Seconds(), nil
}
