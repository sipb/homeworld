package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"os"
)

var (
	registry = prometheus.NewRegistry()

	ociCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "oci",
		Name:      "pull_check",
		Help:      "Check for whether OCIs can be pulled",
	}, []string{"image"})

	execCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
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

	execTiming = prometheus.NewHistogramVec(prometheus.HistogramOpts{
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
	execGauge := execCheck.With(prometheus.Labels{
		"image": image,
	})
	execHisto := execTiming.With(prometheus.Labels{
		"image": image,
	})
	time_taken, err := refetch(image)
	if err != nil {
		log.Println(err)
		ociGauge.Set(0)
		execGauge.Set(0)
		return
	}
	ociGauge.Set(1)
	ociHisto.Observe(time_taken)

	time_taken_for_exec, err := attemptEcho(image)
	if err != nil {
		log.Println(err)
		execGauge.Set(0)
		return
	}
	execGauge.Set(1)
	execHisto.Observe(time_taken_for_exec)
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
	registry.MustRegister(execCheck)
	registry.MustRegister(ociTiming)
	registry.MustRegister(execTiming)

	stopChannel := make(chan struct{})
	defer close(stopChannel)
	go loop(image, stopChannel)

	address := ":9103"

	log.Printf("Starting metrics server on: %v", address)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	err := http.ListenAndServe(address, nil)
	log.Printf("Stopped metrics server: %v", err)
}
