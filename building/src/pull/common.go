package pull

import (
	"k8s.io/client-go/kubernetes"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	core_v1 "k8s.io/api/core/v1"
)

func ListNodeIPs(cs *kubernetes.Clientset) ([]string, error) {
	nodelist, err := cs.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var ip_list []string
	for _, item := range nodelist.Items {
		nodeIP := ""
		for _, addr := range item.Status.Addresses {
			if addr.Type == core_v1.NodeInternalIP {
				if nodeIP != "" && nodeIP != addr.Address {
					return nil, fmt.Errorf("got multiple IPs for node: %s", item.Name)
				}
				nodeIP = addr.Address
			}
		}
		if nodeIP == "" {
			return nil, fmt.Errorf("could not get IP for node: %s", item.Name)
		}
		ip_list = append(ip_list, nodeIP)
	}
	return ip_list, err
}

// WARNING: DO NOT PASS UNTRUSTED INPUT TO THE 'APP' PARAMETER
// returns map[podip]hostip
func ListRunningAppPods(cs *kubernetes.Clientset, namespace string, app string) (map[string]string, error) {
	podlist, err := cs.CoreV1().Pods(namespace).List(meta_v1.ListOptions{LabelSelector: "app=" + app})
	if err != nil {
		return nil, err
	}
	mapPodIPToHostIP := map[string]string{}
	for _, item := range podlist.Items {
		if item.Status.Phase != "Running" {
			continue
		}
		hostIP := item.Status.HostIP
		podIP := item.Status.PodIP
		_, found := mapPodIPToHostIP[podIP]
		if found {
			return nil, fmt.Errorf("duplicate pod for podIP: %s for %s", item.Name, podIP)
		}
		mapPodIPToHostIP[podIP] = hostIP
	}
	return mapPodIPToHostIP, nil
}

// returns ips_not_found, has_any_duplicates, map_nodeip_to_podip, error
func ListAndMatchRunningAppPodsToNodes(cs *kubernetes.Clientset, namespace string, app string) ([]string, bool, map[string]string, error) {
	node_ips, err := ListNodeIPs(cs)
	if err != nil {
		return nil, false, nil, err
	}
	mapPodIPToHostIP, err := ListRunningAppPods(cs, namespace, app)
	if err != nil {
		return nil, false, nil, err
	}
	has_any_duplicates := false

	mapHostIPToPodIP := map[string]string{}

	for podIP, hostIP := range mapPodIPToHostIP {
		if _, present := mapHostIPToPodIP[hostIP]; present {
			has_any_duplicates = true
		}
		mapHostIPToPodIP[hostIP] = podIP
	}

	not_found := []string{}
	for _, nodeIP := range node_ips {
		if _, present := mapHostIPToPodIP[nodeIP]; !present {
			not_found = append(not_found, nodeIP)
		}
	}

	return not_found, has_any_duplicates, mapHostIPToPodIP, nil
}
