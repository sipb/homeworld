package main

import (
	rand2 "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// returns whether or not a particular image exists according to CRI-O. also returns false on error.
func doesImageExist(image string) bool {
	cmd := exec.Command("crictl", "inspecti", image)
	return cmd.Run() == nil
}

// deletes an image from CRI-O's image store
func deleteImage(image string) error {
	cmd := exec.Command("crictl", "rmi", image)
	return cmd.Run()
}

// pulls an image and saves it into CRI-O's image store
func pullImage(image string) error {
	cmd := exec.Command("crictl", "pull", image)
	return cmd.Run()
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

type PodSandboxMetadata struct {
	Name      string `json:"name"`
	Uid       string `json:"uid"`
	Namespace string `json:"namespace"`
	Attempt   uint32 `json:"attempt"`
}

type PodSandboxConfig struct {
	Metadata     *PodSandboxMetadata `json:"metadata"`
	LogDirectory string              `json:"log_directory"`
}

type ContainerMetadata struct {
	Name string `json:"name"`
}

type ImageSpec struct {
	Image string `json:"image"`
}

type ContainerConfig struct {
	Metadata *ContainerMetadata `json:"metadata"`
	Image    *ImageSpec         `json:"image"`
	Command  []string           `json:"command"`
	Args     []string           `json:"args"`
	LogPath  string             `json:"log_path"`
}

type ContainerStatus struct {
	State string `json:"state"`
}

type Inspection struct {
	Status *ContainerStatus `json:"status"`
}

func generateUID() (string, error) {
	randomID := make([]byte, 16)
	if _, err := rand2.Read(randomID[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(randomID[:]), nil
}

// create a temporary file with a JSON-encoded structure as its only content, and return a path to it.
// the caller is responsible for deleting the file once they are done.
func createJSONTempfile(contents interface{}) (path string, err error) {
	file, err := ioutil.TempFile("", "pullcheck-manifest-")
	if err != nil {
		return "", errors.Wrap(err, "creating tempfile")
	}
	defer func() {
		if err2 := file.Close(); err2 != nil {
			err = multierror.Append(err, err2)
		}
	}()

	enc := json.NewEncoder(file)
	if err := enc.Encode(contents); err != nil {
		err = multierror.Append(err, os.Remove(file.Name()))
		return "", errors.Wrap(err, "writing JSON")
	}

	return file.Name(), nil
}

// instruct CRI-O to create a new Pod Sandbox in which we can launch our container
// returns the Pod ID of the created sandbox, if no error occurs
func startPodSandbox() (id string, configpath string, err error) {
	uid, err := generateUID()
	if err != nil {
		return "", "", errors.Wrap(err, "generating UID")
	}
	config := PodSandboxConfig{
		Metadata: &PodSandboxMetadata{
			Name:      "pullcheck-sandbox",
			Namespace: "pullcheck",
			Attempt:   1,
			Uid:       "pullcheck-" + uid,
		},
		LogDirectory: "/tmp/pullcheck-logs/" + uid + "/",
	}
	configpath, err = createJSONTempfile(config)
	if err != nil {
		return "", "", errors.Wrap(err, "creating pod config")
	}
	cmd := exec.Command("crictl", "runp", configpath)
	result, err := cmd.Output()
	if err != nil {
		err = multierror.Append(err, os.Remove(configpath))
		return "", "", errors.Wrap(err, "invoking runp")
	}
	idstr := strings.TrimSpace(string(result))
	if len(idstr) != 64 {
		err = multierror.Append(err, os.Remove(configpath))
		return "", "", errors.Errorf("not a valid ID; not 64 characters long: %s", idstr)
	}
	return idstr, configpath, nil
}

// instruct CRI-O to tear down an existing Pod Sandbox
func deletePodSandbox(podid string) error {
	cmd := exec.Command("crictl", "stopp", podid)
	if _, err := cmd.Output(); err != nil {
		return err
	}
	cmd = exec.Command("crictl", "rmp", podid)
	if _, err := cmd.Output(); err != nil {
		return err
	}
	return nil
}

// instruct CRI-O to create (but not start) a container within an existing Pod Sandbox, by ID
// returns the Container ID of the created container, if no error occurs
func createContainer(image string, podconfigpath string, podid string, args []string) (containerid string, err error) {
	config := &ContainerConfig{
		Metadata: &ContainerMetadata{
			Name: "pullcheck",
		},
		Image: &ImageSpec{
			Image: image,
		},
		Command: args,
		LogPath: "container.log",
	}
	configpath, err := createJSONTempfile(config)
	if err != nil {
		return "", errors.Wrap(err, "creating container config")
	}
	defer func() {
		err2 := os.Remove(configpath)
		if err2 != nil {
			err = multierror.Append(err, err2)
		}
	}()
	cmd := exec.Command("crictl", "create", podid, configpath, podconfigpath)
	result, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "invoking create")
	}
	idstr := strings.TrimSpace(string(result))
	if len(idstr) != 64 {
		return "", errors.Errorf("not a valid ID; not 64 characters long: %s", idstr)
	}
	return idstr, nil
}

// instruct CRI-O to start a container, by ID, that has already been created
func startContainer(containerid string) error {
	cmd := exec.Command("crictl", "start", containerid)
	_, err := cmd.Output()
	return err
}

// ask CRI-O whether a particular container, referenced by ID, is running or not
func isContainerRunning(containerid string) (bool, error) {
	cmd := exec.Command("crictl", "inspect", containerid)
	result, err := cmd.Output()
	if err != nil {
		return false, errors.Wrap(err, "inspecting container")
	}
	var inspection Inspection
	err = json.Unmarshal(result, &inspection)
	if err != nil {
		return false, errors.Wrap(err, "reading inspection response")
	}
	if inspection.Status.State == "CONTAINER_RUNNING" {
		return true, nil
	} else if inspection.Status.State == "CONTAINER_EXITED" {
		return false, nil
	} else {
		return false, errors.Errorf("unknown state: '%s'", inspection.Status.State)
	}
}

// retrieve the container logs from CRI-O for a particular container, by ID
// returns the retrieved logs if no error
func getContainerLogs(containerid string) (result string, err error) {
	cmd := exec.Command("crictl", "logs", containerid)
	output, err := cmd.Output()
	return string(output), err
}

// inside a specified Pod Sandbox (by ID), create and start a new container with particular arguments, then wait for it
// to complete and return the contents of its logging output, if no error occurs
// the arguments must include the name of the executable to invoke
func runContainer(image string, podconfigpath string, podid string, args []string) (result string, err error) {
	containerid, err := createContainer(image, podconfigpath, podid, args)
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
	podid, podconfig, err := startPodSandbox()
	if err != nil {
		return "", errors.Wrap(err, "starting pod sandbox")
	}
	defer func() {
		err2 := os.Remove(podconfig)
		if err2 != nil {
			err = multierror.Append(err, errors.Wrap(err2, "removing pod configuration"))
		}
	}()
	result, err = runContainer(image, podconfig, podid, args)
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
