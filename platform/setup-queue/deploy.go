package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/kubernetes/wrapper"
)

const KubeConfigPath = "/etc/homeworld/config/kubeconfig-setup-queue"
const QueueDir = "/etc/homeworld/deployqueue/"

func DeploySingle(kubeconfig string, file string) error {
	cmd := exec.Command("hyperkube", "kubectl", "apply", "--kubeconfig", kubeconfig, "-f", file)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func DeployAll(dir string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		log.Println("nothing to queue")
		return nil
	}
	err = wrapper.GenerateKubeConfigToFile(paths.KubernetesWorkerKey, paths.KubernetesWorkerCert, KubeConfigPath)
	if err != nil {
		return err
	}
	for _, file := range files {
		name := path.Join(dir, file.Name())

		log.Printf("deploying %s\n", name)
		err := DeploySingle(KubeConfigPath, name)
		if err != nil {
			return err
		}

		err = os.Remove(name)
		if err != nil {
			return err
		}
	}
	log.Printf("finished deploying %d specs\n", len(files))
	return nil
}

func main() {
	err := DeployAll(QueueDir)
	if err != nil {
		log.Fatalf("error in setup-queue: %s\n", err.Error())
	}
}
