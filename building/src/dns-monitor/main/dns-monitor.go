package main

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	registry = prometheus.NewRegistry()

	internalCheck = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "dns",
		Name:      "lookup_internal_check",
		Help:      "Check for whether in-cluster dns lookups work",
	}, []string{"hostname"})

	internalCheckTiming = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "dns",
		Name:      "lookup_internal_timing",
		Help:      "Timing for flannel communication",
		Buckets:   []float64{0.1, 0.2, 0.5, 1, 2, 5, 10},
	}, []string{"hostname"})

	monRecency = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "dns",
		Name:      "lookup_recency",
		Help:      "Timestamp for the oldest currently reported metric",
	})

	// TODO: monitor external lookups as well

	resolver = net.Resolver{
		PreferGo: true,
	}
)

func grabOneIPv4(host string) (net.IP, time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	start := time.Now()
	ipAddresses, err := resolver.LookupIPAddr(ctx, host)
	end := time.Now()
	if err != nil {
		return nil, 0, err
	}
	if len(ipAddresses) != 1 {
		return nil, 0, fmt.Errorf("expected exactly one IP address")
	}
	return ipAddresses[0].IP, end.Sub(start), nil
}

func cycle(expectations map[string]net.IP) {
	recencyTimestamp := time.Now() // such that none of the DNS-derived metrics are *older* than this timestamp

	for hostname, expectIP := range expectations {
		foundIP, duration, err := grabOneIPv4(hostname)
		ic := internalCheck.With(prometheus.Labels{"hostname": hostname})
		ict := internalCheckTiming.With(prometheus.Labels{"hostname": hostname})
		if err != nil {
			log.Printf("failed to check DNS result for %s: %v", hostname, err)
			ic.Set(0)
		} else if !expectIP.Equal(foundIP) {
			ic.Set(0)
			log.Printf("DNS mismatch between result %v for %s and expectation %v", foundIP, hostname, expectIP)
		} else {
			ic.Set(1)
			ict.Observe(duration.Seconds())
		}
	}

	monRecency.Set(recencyTimestamp.Sub(time.Unix(0, 0)).Seconds())
}

func loop(expectations map[string]net.IP, stopChannel <-chan struct{}) {
	for {
		next_cycle_at := time.Now().Add(time.Second * 15)
		cycle(expectations)

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

func parseArgs() (map[string]net.IP, error) {
	expectations := map[string]net.IP{}
	for _, arg := range os.Args[1:] {
		parts := strings.Split(arg, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("found improperly formatted parameter: %s", arg)
		}
		hostname := parts[0]
		ipaddr := net.ParseIP(parts[1])
		if ipaddr == nil {
			return nil, fmt.Errorf("found invalid IP address: %s", parts[1])
		}
		if _, found := expectations[hostname]; found {
			return nil, fmt.Errorf("duplicate hostname: %s", hostname)
		}
		expectations[hostname] = ipaddr
	}
	if len(expectations) == 0 {
		return nil, fmt.Errorf("no parameters provided")
	}
	return expectations, nil
}

func main() {
	expectations, err := parseArgs()
	if err != nil {
		log.Fatalf("failed to parse arguments: %v", err)
	}

	registry.MustRegister(internalCheck)
	registry.MustRegister(internalCheckTiming)
	registry.MustRegister(monRecency)

	stopChannel := make(chan struct{})
	defer close(stopChannel)
	go loop(expectations, stopChannel)

	address := ":80"

	log.Printf("Starting metrics server on: %v", address)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	err = http.ListenAndServe(address, nil)
	log.Printf("Stopped metrics server: %v", err)
}
