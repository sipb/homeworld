package main

import (
	"fmt"
	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/kubernetes/wrapper"
	"os"
	"os/exec"
	"strconv"
)

const KubeConfigPath = "/etc/homeworld/config/kubeconfig-kube-state-metrics"
const MetricsPort = 9104
const MetaMetricsPort = 9105

func LaunchKSM(kubeconfig string, port, metaport int) error {
	portStr := strconv.FormatInt(int64(port), 10)
	metaportStr := strconv.FormatInt(int64(metaport), 10)
	cmd := exec.Command("/usr/bin/kube-state-metrics",
		"--kubeconfig", kubeconfig, "--port", portStr, "--telemetry-port", metaportStr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ConfigureAndLaunch() error {
	kubeconfig := KubeConfigPath
	err := wrapper.GenerateKubeConfigToFile(paths.KubernetesSupervisorKey, paths.KubernetesSupervisorCert, kubeconfig)
	if err != nil {
		return err
	}
	return LaunchKSM(kubeconfig, MetricsPort, MetaMetricsPort)
}

func main() {
	err := ConfigureAndLaunch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kube-state-metrics failed: %s\n", err.Error())
		os.Exit(1)
	}
}
