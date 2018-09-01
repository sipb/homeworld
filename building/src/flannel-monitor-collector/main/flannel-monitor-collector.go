package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
	"net"
	"net/http"
	"os"
	"pull"
	"time"
)

var (
	registry = prometheus.NewRegistry()

	collectCheck = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "flannel",
		Name:      "collect_enum_check",
		Help:      "Check for whether the flannel-monitor collector can enumerate the monitor containers",
	})
	dupCheck = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "flannel",
		Name:      "collect_enum_dup_check",
		Help:      "Check for whether the flannel-monitor collector successfully found no duplicate reflectors",
	})
	scrapeCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "flannel",
		Name:      "collect_check",
		Help:      "Check for whether the flannel-monitor collector can scrape the monitor containers",
	}, []string{"monitor_hostip"})

	additional_metrics []*io_prometheus_client.MetricFamily
)

func cycle(clientset *kubernetes.Clientset, namespace string, client *http.Client) {
	podlessNodeIPs, anyDuplicateNodes, mapHostIPToPodIP, err := pull.ListAndMatchRunningAppPodsToNodes(clientset, namespace, "flannel-monitor")
	if err != nil {
		scrapeCheck.Reset()
		additional_metrics = nil
		collectCheck.Set(0)
		dupCheck.Set(0)
		log.Printf("could not fetch list of flannel-monitor apps: %v", err)
		return
	}

	if anyDuplicateNodes {
		dupCheck.Set(0)
	} else {
		dupCheck.Set(1)
	}

	for _, nodeIP := range podlessNodeIPs {
		scrapeCheck.With(prometheus.Labels{
			"monitor_hostip": nodeIP,
		}).Set(0)
	}

	var result []*io_prometheus_client.MetricFamily
	textParser := &expfmt.TextParser{}

	for hostIP, podIP := range mapHostIPToPodIP {
		resp, err := client.Get(fmt.Sprintf("http://%s/metrics", podIP))
		pass := float64(0)
		if err != nil {
			log.Printf("failed to fetch metrics from %s: %v", podIP, err)
		} else {
			metrics, err := textParser.TextToMetricFamilies(resp.Body)
			if err != nil {
				log.Printf("failed to parse metrics from %s: %v", podIP, err)
			} else {
				for _, family := range metrics {
					result = append(result, family)
				}
				pass = 1
			}
		}
		scrapeCheck.With(prometheus.Labels{
			"monitor_hostip": hostIP,
		}).Set(pass)
	}

	collectCheck.Set(1)

	additional_metrics = result
}

func gather() ([]*io_prometheus_client.MetricFamily, error) {
	adl := additional_metrics
	gathered, err := registry.Gather()
	if err != nil {
		return nil, err
	}
	temp := make([]*io_prometheus_client.MetricFamily, 0, len(adl)+len(gathered))
	temp = append(temp, gathered...)
	temp = append(temp, adl...)
	return temp, nil
}

func loop(clientset *kubernetes.Clientset, namespace string, client *http.Client, stopChannel <-chan struct{}) {
	for {
		next_cycle_at := time.Now().Add(time.Second * 15)
		cycle(clientset, namespace, client)

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

	registry.MustRegister(collectCheck)
	registry.MustRegister(dupCheck)
	registry.MustRegister(scrapeCheck)

	stopChannel := make(chan struct{})
	defer close(stopChannel)
	go loop(clientset, namespace, client, stopChannel)

	address := ":80"

	log.Printf("Starting metrics server on: %v", address)
	http.Handle("/metrics", promhttp.HandlerFor(prometheus.GathererFunc(gather), promhttp.HandlerOpts{}))
	err = http.ListenAndServe(address, nil)
	log.Printf("Stopped metrics server: %v", err)
}
