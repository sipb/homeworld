/*
Copyright 2018 Cel Skeggs (modifications)
Copyright 2017 The Kubernetes Authors (original)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"log"
	"os"
	"strings"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"io"
)

// Initialize the prometheus instrumentation and client related flags.
var (
	etcdScrapeBase string

	httpClient *http.Client
)

// Initialize prometheus metrics to be exported.
var (
	// Register all custom metrics with a dedicated registry to keep them separate.
	customMetricRegistry = prometheus.NewRegistry()

	// Custom etcd version metric since etcd 3.2- does not export one.
	// This will be replaced by https://github.com/coreos/etcd/pull/8960 in etcd 3.3.
	etcdVersion = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "etcd",
			Name:      "version_info",
			Help:      "Etcd server's binary version",
		},
		[]string{"binary_version"})
)

// monitorGatherer is a custom metric gatherer for prometheus that exports custom metrics
// defined by this monitor as well as rewritten etcd metrics.
type monitorGatherer struct {
}

func (m *monitorGatherer) Gather() ([]*dto.MetricFamily, error) {
	etcdMetrics, err := scrapeMetrics()
	if err != nil {
		return nil, err
	}
	custom, err := customMetricRegistry.Gather()
	if err != nil {
		return nil, err
	}
	result := make([]*dto.MetricFamily, 0, len(etcdMetrics)+len(custom))
	for _, mf := range etcdMetrics {
		result = append(result, mf)
	}
	result = append(result, custom...)
	return result, nil
}

// Struct for unmarshalling the json response from etcd's /version endpoint.
type EtcdVersion struct {
	BinaryVersion  string `json:"etcdserver"`
	ClusterVersion string `json:"etcdcluster"`
}

func requestToEtcd(endpoint string) (io.ReadCloser, error) {
	if !strings.HasPrefix(endpoint, "/") {
		return nil, fmt.Errorf("no '/' prefix on endpoint")
	}

	resp, err := httpClient.Get(etcdScrapeBase + endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to receive GET response from etcd: %v", err)
	}

	return resp.Body, nil
}

// Function for fetching etcd version info and feeding it to the prometheus metric.
func getVersion(lastSeenBinaryVersion *string) error {
	body, err := requestToEtcd("/version")
	if err != nil {
		return err
	}
	defer body.Close()

	// Obtain EtcdVersion from the JSON response.
	var version EtcdVersion
	if err := json.NewDecoder(body).Decode(&version); err != nil {
		return fmt.Errorf("Failed to decode etcd version JSON: %v", err)
	}

	// Return without updating the version if it stayed the same since last time.
	if *lastSeenBinaryVersion == version.BinaryVersion {
		return nil
	}

	// Delete the metric for the previous version.
	if *lastSeenBinaryVersion != "" {
		deleted := etcdVersion.Delete(prometheus.Labels{"binary_version": *lastSeenBinaryVersion})
		if !deleted {
			return fmt.Errorf("Failed to delete previous version's metric")
		}
	}

	// Record the new version in a metric.
	etcdVersion.With(prometheus.Labels{
		"binary_version": version.BinaryVersion,
	}).Set(1)
	*lastSeenBinaryVersion = version.BinaryVersion
	return nil
}

// Periodically fetches etcd version info.
func getVersionPeriodically(stopCh <-chan struct{}) {
	lastSeenBinaryVersion := ""
	for {
		if err := getVersion(&lastSeenBinaryVersion); err != nil {
			log.Printf("Failed to fetch etcd version: %v", err)
		}
		select {
		case <-stopCh:
			break
		case <-time.After(15 * time.Second):
		}
	}
}

// scrapeMetrics scrapes the prometheus metrics from the etcd metrics URI.
func scrapeMetrics() (map[string]*dto.MetricFamily, error) {
	body, err := requestToEtcd("/metrics")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	// Parse the metrics in text format to a MetricFamily struct.
	var textParser expfmt.TextParser
	return textParser.TextToMetricFamilies(body)
}

func main() {
	if len(os.Args) != 5 {
		log.Fatal("expected four arguments: baseurl, authority, keyfile, certfile")
	}

	etcdScrapeBase = os.Args[1]

	certPool := x509.NewCertPool()

	authorities, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	if !certPool.AppendCertsFromPEM(authorities) {
		log.Fatal("could not parse PEM cert for CA")
	}

	certCli, err := tls.LoadX509KeyPair(os.Args[4], os.Args[3])
	if err != nil {
		log.Fatal(err)
	}

	httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
				Certificates: []tls.Certificate { certCli },
			},
		},
	}

	// Register the metrics we defined above with prometheus.
	customMetricRegistry.MustRegister(etcdVersion)
	customMetricRegistry.Unregister(prometheus.NewGoCollector())

	// Spawn threads for periodically scraping etcd version metrics.
	stopCh := make(chan struct{})
	defer close(stopCh)
	go getVersionPeriodically(stopCh)

	listenAddress := ":9101"

	// Serve our metrics.
	log.Printf("Listening on: %v", listenAddress)
	http.Handle("/metrics", promhttp.HandlerFor(&monitorGatherer{}, promhttp.HandlerOpts{}))
	log.Printf("Stopped listening/serving metrics: %v", http.ListenAndServe(listenAddress, nil))
}
