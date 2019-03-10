package main

import (
	"net/http"
	"time"

	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
)

var (
	registry = prometheus.NewRegistry()

	ociCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "oci",
		Name:      "pull_check",
		Help:      "Check for whether OCIs can be pulled",
	}, []string{"image"})

	ociHashes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "oci",
		Name:      "pull_hash",
		Help:      "Counters for each OCI hash found",
	}, []string{"image", "hash"})

	rktCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "oci",
		Name:      "exec_check",
		Help:      "Check for whether OCIs can be launched",
	}, []string{"image"})

	ociTiming = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "oci",
		Name:      "pull_timing_seconds",
		Help:      "Timing for pulling OCIs",
		Buckets:   []float64{8, 10, 11, 12, 13, 14, 15, 16, 18, 20, 30, 40},
	}, []string{"image"})

	rktTiming = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "oci",
		Name:      "exec_timing_seconds",
		Help:      "Timing for launching OCIs",
		Buckets:   []float64{5, 6, 7, 8, 9, 10, 15, 30, 60},
	}, []string{"image"})
)

func cycle(image string) {
	ociGauge := ociCheck.With(prometheus.Labels{
		"image": image,
	})
	ociHisto := ociTiming.With(prometheus.Labels{
		"image": image,
	})
	rktGauge := rktCheck.With(prometheus.Labels{
		"image": image,
	})
	rktHisto := rktTiming.With(prometheus.Labels{
		"image": image,
	})
	// if the image isn't found, still returns exit code zero
	cmd := exec.Command("rkt", "image", "rm", image)
	if err := cmd.Run(); err != nil {
		log.Printf("failed to remove previous image: %v", err)
		ociGauge.Set(0)
		rktGauge.Set(0)
		return
	}
	args := []string{
		"fetch", image, "--full",
	}
	if strings.HasPrefix(image, "docker://") {
		args = append(args, "--insecure-options=image")
	}
	cmd = exec.Command("rkt", args...)
	time_start := time.Now()
	hash_raw, err := cmd.Output()
	time_end := time.Now()
	if err != nil {
		log.Printf("failed to fetch new image: %v", err)
		ociGauge.Set(0)
		rktGauge.Set(0)
		return
	}
	ociGauge.Set(1)
	time_taken := time_end.Sub(time_start).Seconds()
	hash := strings.TrimSpace(string(hash_raw))

	ociHashes.With(prometheus.Labels{
		"image": image,
		"hash":  hash,
	}).Inc()

	ociHisto.Observe(time_taken)

	echo_data := fmt.Sprintf("!%d!", rand.Uint64())
	cmd = exec.Command("rkt", "run", image, "--", echo_data)
	time_rkt_start := time.Now()
	output, err := cmd.Output()
	time_rkt_end := time.Now()
	if err != nil {
		log.Printf("failed to exec new image: %v", err)
		rktGauge.Set(0)
		return
	}
	lines := strings.Split(strings.Trim(string(output), "\000\r\n"), "\n")
	if strings.Contains(lines[len(lines)-1], "rkt: obligatory restart") && len(lines) > 1 {
		lines = lines[:len(lines)-1]
	}
	parts := strings.SplitN(strings.Trim(lines[len(lines)-1], "\000\r\n"), ": ", 2)
	if !strings.Contains(parts[0], "] pullcheck[") || len(parts) != 2 {
		log.Printf("output from rkt did not match expected format: '%s'", string(output))
		rktGauge.Set(0)
		return
	}
	if parts[1] != fmt.Sprintf("hello container world [%s]", echo_data) {
		log.Printf("output from rkt did not match expectation: '%s' (%v) instead of '%s'", parts[1], []byte(parts[1]), echo_data)
		rktGauge.Set(0)
		return
	}
	rktGauge.Set(1)

	time_rkt_taken := time_rkt_end.Sub(time_rkt_start).Seconds()

	rktHisto.Observe(time_rkt_taken)
}

func loop(image string, stopChannel <-chan struct{}) {
	for {
		next_cycle_at := time.Now().Add(time.Second * 15)
		cycle(image)

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
	if len(os.Args) != 2 {
		log.Fatal("expected exactly one argument: <image-to-pull>")
	}
	image := os.Args[1]

	registry.MustRegister(ociCheck)
	registry.MustRegister(ociHashes)
	registry.MustRegister(rktCheck)
	registry.MustRegister(ociTiming)
	registry.MustRegister(rktTiming)

	stopChannel := make(chan struct{})
	defer close(stopChannel)
	go loop(image, stopChannel)

	address := ":9103"

	log.Printf("Starting metrics server on: %v", address)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	err := http.ListenAndServe(address, nil)
	log.Printf("Stopped metrics server: %v", err)
}
