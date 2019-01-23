package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"net"
	"net/http"
	"os"
)

func getIP(network *net.IPNet) (net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var found net.IP
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := addr.(*net.IPNet).IP
			if network.Contains(ip) {
				if found != nil {
					return nil, fmt.Errorf("multiple IPs found that match network %v: %v, %v", network, found, ip)
				}
				found = ip
			}
		}
	}
	if found == nil {
		return nil, fmt.Errorf("could not find matching local IP for network %v", network)
	} else {
		return found, nil
	}
}

func main() {

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientset := kubernetes.NewForConfigOrDie(config)
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		log.Fatal("no namespace found in POD_NAMESPACE")
	}
	name := os.Getenv("POD_NAME")
	if name == "" {
		log.Fatal("no name found in POD_NAME")
	}
	network_raw := os.Getenv("FLANNEL_NETWORK")
	if name == "" {
		log.Fatal("no network found in FLANNEL_NETWORK")
	}
	_, network, err := net.ParseCIDR(network_raw)
	if err != nil {
		log.Fatal(err)
	}
	pod, err := clientset.CoreV1().Pods(namespace).Get(name, v1.GetOptions{})
	if err != nil {
		log.Fatal("cannot get info for current reflector pod")
	}
	podIP := net.ParseIP(pod.Status.PodIP)
	if podIP == nil {
		log.Fatal("invalid pod IP")
	}
	if !network.Contains(podIP) {
		log.Fatal("pod IP not in expected network")
	}
	localIP, err := getIP(network)
	if err != nil {
		log.Fatal("could not get local IP")
	}
	if !localIP.Equal(podIP) {
		log.Fatal("discovered IP mismatch")
	}
	log.Fatal(http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/ping" {
			w.Write([]byte("PONG"))
		} else {
			http.Error(w, "not found", 404)
		}
	})))
}
