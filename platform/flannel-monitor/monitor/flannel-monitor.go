package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/sipb/homeworld/platform/flannel-monitor/common"
)

var (
	registry = prometheus.NewRegistry()

	talkCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "flannel",
		Name:      "talk_check",
		Help:      "Check for whether flannel supports container communication",
	}, []string{"ping_from_host", "ping_to_host"})

	monRecency = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "flannel",
		Name:      "monitor_recency",
		Help:      "Timestamp for the oldest currently reported metric",
	}, []string{"ping_from_host"})

	talkTiming = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "flannel",
		Name:      "talk_timing",
		Help:      "Timing for flannel communication",
		Buckets:   []float64{0.1, 0.2, 0.5, 1, 2, 5, 10},
	}, []string{"ping_from_host", "ping_to_host"})

	monCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "flannel",
		Name:      "monitor_check",
		Help:      "Whether flannel was able to be monitored from this host (and, therefore, the other metrics are valid)",
	}, []string{"ping_from_host"})

	dupCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "flannel",
		Name:      "duplicate_check",
		Help:      "Whether flannel monitoring successfully found no duplicate reflectors",
	}, []string{"ping_from_host"})
)

func cycle(clientset *kubernetes.Clientset, namespace string, source_hostip string, client *http.Client) {
	valid := monCheck.With(prometheus.Labels{"ping_from_host": source_hostip})
	dupOkay := dupCheck.With(prometheus.Labels{"ping_from_host": source_hostip})
	monRecencySpec := monRecency.With(prometheus.Labels{"ping_from_host": source_hostip})

	podlessNodeIPs, anyDuplicateNodes, mapHostIPToPodIP, err := common.ListAndMatchRunningAppPodsToNodes(clientset, namespace, "flannel-monitor-reflector")
	if err != nil {
		talkCheck.Reset()
		talkTiming.Reset()
		valid.Set(0)
		dupOkay.Set(0)
		log.Printf("could not fetch list of flannel-monitor-reflector apps: %v", err)
		return
	}

	for _, nodeIP := range podlessNodeIPs {
		talkCheck.With(prometheus.Labels{
			"ping_from_host": source_hostip,
			"ping_to_host":   nodeIP,
		}).Set(0)
	}

	if anyDuplicateNodes {
		dupOkay.Set(0)
	} else {
		dupOkay.Set(1)
	}

	valid.Set(1)

	recency_timestamp := time.Now() // such that none of the httpping-derived metrics are *older* than this timestamp

	for hostIP, podIP := range mapHostIPToPodIP {
		start := time.Now()
		resp, err := client.Get(fmt.Sprintf("http://%s/ping", podIP))
		end := time.Now()

		pass := float64(0)
		if err != nil {
			log.Printf("failed to HTTP-ping %s: %v", podIP, err)
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Printf("failed to read HTTP-ping response to %s: %v", podIP, err)
			} else {
				if string(body) != "PONG" {
					log.Printf("invalid HTTP-ping response to %s: '%s'", podIP, string(body))
				} else {
					pass = 1
					talkTiming.With(prometheus.Labels{
						"ping_from_host": source_hostip,
						"ping_to_host":   hostIP,
					}).Observe(end.Sub(start).Seconds())
				}
			}
		}
		talkCheck.With(prometheus.Labels{
			"ping_from_host": source_hostip,
			"ping_to_host":   hostIP,
		}).Set(pass)
	}

	monRecencySpec.Set(recency_timestamp.Sub(time.Unix(0, 0)).Seconds())
}

func loop(clientset *kubernetes.Clientset, namespace string, source_hostip string, client *http.Client, stopChannel <-chan struct{}) {
	for {
		next_cycle_at := time.Now().Add(time.Second * 15)
		cycle(clientset, namespace, source_hostip, client)

		delta := next_cycle_at.Sub(time.Now())
		if delta < time.Second {
			delta = time.Second
		}

		select {
		case <-stopChannel:
			break
		case <-time.After(delta):
		}
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
	selfpod, err := clientset.CoreV1().Pods(namespace).Get(name, v1.GetOptions{})
	if err != nil {
		log.Fatal("cannot get info for current reflector pod")
	}
	source_hostip := selfpod.Status.HostIP

	registry.MustRegister(talkCheck)
	registry.MustRegister(talkTiming)
	registry.MustRegister(monCheck)
	registry.MustRegister(monRecency)
	registry.MustRegister(dupCheck)

	client := &http.Client{
		Timeout: time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			IdleConnTimeout:       time.Minute,
			ResponseHeaderTimeout: time.Second,
			MaxIdleConns:          100,
		},
	}

	stopChannel := make(chan struct{})
	defer close(stopChannel)
	go loop(clientset, namespace, source_hostip, client, stopChannel)

	address := ":80"

	log.Printf("Starting metrics server on: %v", address)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	err = http.ListenAndServe(address, nil)
	log.Printf("Stopped metrics server: %v", err)
}
