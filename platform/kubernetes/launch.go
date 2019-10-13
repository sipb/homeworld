package main

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sipb/homeworld/platform/keysystem/worldconfig/paths"
	"github.com/sipb/homeworld/platform/kubernetes/wrapper"
)

func getVerbosityArgument() string {
	verbosityBytes, err := ioutil.ReadFile("/etc/homeworld/config/verbosity")
	if err != nil {
		return "--v=0"
	}
	verbosity := strings.TrimSpace(string(verbosityBytes))
	if _, err := strconv.ParseUint(verbosity, 10, 32); err != nil {
		log.Printf("cannot parse verbosity field value: '%s'\n", verbosity)
		return "--v=0"
	}
	log.Printf("using launch verbosity %s\n", verbosity)
	return "--v=" + verbosity
}

func kubeConfigPath(suffix string) string {
	return "/etc/homeworld/config/kubeconfig-" + suffix
}

func LaunchAPIServer() error {
	clusterConf, err := wrapper.GetClusterConf()
	if err != nil {
		return errors.Wrap(err, "while loading cluster.conf")
	}
	localConf, err := wrapper.GetLocalConf()
	if err != nil {
		return errors.Wrap(err, "while loading local.conf")
	}
	// parse the keys before using them for the sake of validation
	apiserverCount := clusterConf["APISERVER_COUNT"]
	if _, err := strconv.ParseUint(apiserverCount, 10, 32); err != nil {
		return errors.Wrap(err, "while validating APISERVER_COUNT")
	}
	serviceCidr := clusterConf["SERVICE_CIDR"]
	if _, _, err := net.ParseCIDR(serviceCidr); err != nil {
		return errors.Wrap(err, "while parsing SERVICE_CIDR")
	}
	hostIP := localConf["HOST_IP"]
	if net.ParseIP(hostIP) == nil {
		return fmt.Errorf("invalid IP '%s'", hostIP)
	}
	etcdEndpoints := clusterConf["ETCD_ENDPOINTS"]
	if strings.Count(etcdEndpoints, "://") < 1 { // just a sniff test, because the format is complex
		return fmt.Errorf("missing or invalid ETCD_ENDPOINTS: '%s'", etcdEndpoints)
	}
	cmd := exec.Command(
		"/usr/bin/hyperkube", "kube-apiserver",
		// role-based access control
		"--authorization-mode", "AlwaysAllow",
		// number of api servers
		"--apiserver-count", apiserverCount,
		// public addresses
		"--bind-address", "0.0.0.0", "--advertise-address", hostIP,
		// IP range
		"--service-cluster-ip-range", serviceCidr,
		// use standard HTTPS port for secure port
		"--secure-port", "443",
		// etcd cluster to use
		"--etcd-servers", etcdEndpoints,
		// allow privileged containers to run
		"--allow-privileged", "true",
		// disallow anonymous users
		"--anonymous-auth", "false",
		// various plugins for limitations and protection
		"--admission-control", "NamespaceLifecycle,LimitRanger,ServiceAccount,TaintNodesByCondition,Priority,DefaultTolerationSeconds,DefaultStorageClass,StorageObjectInUseProtection,PersistentVolumeClaimResize,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,RuntimeClass,ResourceQuota,DenyEscalatingExec,SecurityContextDeny",
		// authenticate clients properly
		"--client-ca-file", paths.KubernetesCAPath,
		// do HTTPS properly
		"--tls-cert-file", paths.KubernetesMasterCert, "--tls-private-key-file", paths.KubernetesMasterKey,
		// make sure account deletion works
		"--service-account-lookup",
		// no cloud provider
		"--cloud-provider=",
		// authenticate the etcd cluster to us
		"--etcd-cafile", "/etc/homeworld/authorities/etcd-server.pem",
		// authenticate us to the etcd cluster
		"--etcd-certfile", "/etc/homeworld/keys/etcd-client.pem", "--etcd-keyfile", "/etc/homeworld/keys/etcd-client.key",
		// disallow insecure port
		"--insecure-port", "0",
		// authenticate kubelet to us
		"--kubelet-certificate-authority", paths.KubernetesCAPath,
		// authenticate us to kubelet
		"--kubelet-client-certificate", paths.KubernetesMasterCert, "--kubelet-client-key", paths.KubernetesMasterKey,
		// let controller manager's service tokens work for us
		"--service-account-key-file", "/etc/homeworld/keys/serviceaccount.key",

		getVerbosityArgument(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "while running apiserver")
	}
	return nil
}

func LaunchControllerManager() error {
	clusterConf, err := wrapper.GetClusterConf()
	if err != nil {
		return errors.Wrap(err, "while loading cluster.conf")
	}
	serviceCidr := clusterConf["SERVICE_CIDR"]
	if _, _, err := net.ParseCIDR(serviceCidr); err != nil {
		return errors.Wrap(err, "while parsing SERVICE_CIDR")
	}
	clusterCidr := clusterConf["CLUSTER_CIDR"]
	if _, _, err := net.ParseCIDR(clusterCidr); err != nil {
		return errors.Wrap(err, "while parsing CLUSTER_CIDR")
	}

	kubeconfig := kubeConfigPath("controller-manager")
	err = wrapper.GenerateKubeConfigToFile(paths.KubernetesWorkerKey, paths.KubernetesWorkerCert, kubeconfig)
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"/usr/bin/hyperkube", "kube-controller-manager",

		"--kubeconfig", kubeconfig,

		"--cluster-cidr", clusterCidr,
		"--node-cidr-mask-size", "24",
		"--service-cluster-ip-range", serviceCidr,
		"--cluster-name", "hyades",

		"--leader-elect",
		"--allocate-node-cidrs",

		// granting service tokens
		"--service-account-private-key-file", "/etc/homeworld/keys/serviceaccount.key",
		"--root-ca-file", "/etc/homeworld/authorities/kubernetes.pem",

		getVerbosityArgument(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "while running controller-manager")
	}
	return nil
}

func LaunchKubelet() error {
	clusterConf, err := wrapper.GetClusterConf()
	if err != nil {
		return errors.Wrap(err, "while loading cluster.conf")
	}
	localConf, err := wrapper.GetLocalConf()
	if err != nil {
		return errors.Wrap(err, "while loading cluster.conf")
	}
	serviceDNS := clusterConf["SERVICE_DNS"]
	if net.ParseIP(serviceDNS) == nil {
		return errors.Wrap(err, "while parsing SERVICE_DNS")
	}
	clusterDomain := clusterConf["CLUSTER_DOMAIN"]
	if strings.Count(clusterDomain, ".") < 1 { // sniff test because it's not immediately clear what's okay
		return errors.Wrap(err, "while validating CLUSTER_DOMAIN")
	}
	scheduleWork, err := strconv.ParseBool(localConf["SCHEDULE_WORK"])
	if err != nil {
		return errors.Wrap(err, "while parsing SCHEDULE_WORK")
	}

	kubeconfig := kubeConfigPath("kubelet")
	err = wrapper.GenerateKubeConfigToFile(paths.KubernetesWorkerKey, paths.KubernetesWorkerCert, kubeconfig)
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"/usr/bin/hyperkube", "kubelet",

		"--kubeconfig", kubeconfig,

		"--register-schedulable="+strconv.FormatBool(scheduleWork),
		// turn off anonymous authentication
		"--anonymous-auth=false",
		// add kubelet auth certs
		"--tls-cert-file", paths.KubernetesWorkerCert, "--tls-private-key-file", paths.KubernetesWorkerKey,
		// add client certificate authority
		"--client-ca-file", paths.KubernetesCAPath,
		// turn off cloud provider detection
		"--cloud-provider=",
		// use CRI-O
		"--container-runtime", "remote", "--container-runtime-endpoint", "unix:///var/run/crio/crio.sock",
		// DNS
		"--cluster-dns", serviceDNS, "--cluster-domain", clusterDomain,

		getVerbosityArgument(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "while running kubelet")
	}
	return nil
}

func LaunchProxy() error {
	kubeconfig := kubeConfigPath("proxy")
	err := wrapper.GenerateKubeConfigToFile(paths.KubernetesWorkerKey, paths.KubernetesWorkerCert, kubeconfig)
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"/usr/bin/hyperkube", "kube-proxy",

		"--kubeconfig", kubeconfig,
		// synchronize every minute (TODO: IS THIS A GOOD AMOUNT OF TIME?)
		"--config-sync-period", "1m",

		getVerbosityArgument(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "while running proxy")
	}
	return nil
}

func LaunchScheduler() error {
	kubeconfig := kubeConfigPath("scheduler")
	err := wrapper.GenerateKubeConfigToFile(paths.KubernetesWorkerKey, paths.KubernetesWorkerCert, kubeconfig)
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"/usr/bin/hyperkube", "kube-scheduler",

		"--kubeconfig", kubeconfig,
		"--leader-elect",

		getVerbosityArgument(),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "while running scheduler")
	}
	return nil
}

func LaunchService(name string) error {
	switch name {
	case "apiserver":
		return LaunchAPIServer()
	case "controller-manager":
		return LaunchControllerManager()
	case "kubelet":
		return LaunchKubelet()
	case "proxy":
		return LaunchProxy()
	case "scheduler":
		return LaunchScheduler()
	default:
		return fmt.Errorf("no such service %s", name)
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: kube-launch <service>\n")
	}
	err := LaunchService(os.Args[1])
	if err != nil {
		log.Fatalf("kube-launch error: %s\n", err.Error())
	}
}
