package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"os"
	"strings"
	"os/exec"
	"math/rand"
	"fmt"
)

var (
	registry = prometheus.NewRegistry()

	aciCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "aci",
		Name: "pull_check",
		Help: "Check for whether ACIs can be pulled",
	}, []string {"image"})

	aciHashes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "aci",
		Name: "pull_hash",
		Help: "Counters for each ACI hash found",
	}, []string {"image", "hash"})

	rktCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "aci",
		Name: "rkt_check",
		Help: "Check for whether ACIs can be launched",
	}, []string {"image"})

	aciTiming = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "aci",
		Name: "pull_timing_seconds",
		Help: "Timing for pulling ACIs",
		Buckets: []float64 {8, 10, 11, 12, 13, 14, 15, 16, 18, 20, 30, 40},
	}, []string {"image"})

	rktTiming = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "aci",
		Name: "rkt_timing_seconds",
		Help: "Timing for launching ACIs",
		Buckets: []float64 {5, 6, 7, 8, 9, 10, 15, 30, 60},
	}, []string {"image"})
)

func cycle() {
	image := "homeworld.mit.edu/pullcheck"
	aciGauge := aciCheck.With(prometheus.Labels{
		"image": image,
	})
	aciHisto := aciTiming.With(prometheus.Labels{
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
		aciGauge.Set(0)
		rktGauge.Set(0)
		return
	}
	cmd = exec.Command("rkt", "fetch", image, "--full")
	time_start := time.Now()
	hash_raw, err := cmd.Output()
	time_end := time.Now()
	if err != nil {
		log.Printf("failed to fetch new image: %v", err)
		aciGauge.Set(0)
		rktGauge.Set(0)
		return
	}
	aciGauge.Set(1)
	time_taken := time_end.Sub(time_start).Seconds()
	hash := strings.TrimSpace(string(hash_raw))

	aciHashes.With(prometheus.Labels{
		"image": image,
		"hash": hash,
	}).Inc()

	aciHisto.Observe(time_taken)

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
	if strings.Contains(lines[len(lines) - 1], "rkt: obligatory restart") && len(lines) > 1 {
		lines = lines[:len(lines) - 1]
	}
	parts := strings.SplitN(strings.Trim(lines[len(lines) - 1], "\000\r\n"), ": ", 2)
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

func loop(stopChannel <-chan struct{}) {
	for {
		next_cycle_at := time.Now().Add(time.Second * 15)
		cycle()

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
	if len(os.Args) != 1 {
		log.Fatal("expected no arguments")
	}

	registry.MustRegister(aciCheck)
	registry.MustRegister(aciHashes)
	registry.MustRegister(rktCheck)
	registry.MustRegister(aciTiming)
	registry.MustRegister(rktTiming)

	stopChannel := make(chan struct{})
	defer close(stopChannel)
	go loop(stopChannel)

	address := ":9103"

	log.Printf("Starting metrics server on: %v", address)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	err := http.ListenAndServe(address, nil)
	log.Printf("Stopped metrics server: %v", err)
}
