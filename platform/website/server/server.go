package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var requestsServed = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "website",
	Name:      "requests_served",
	Help:      "Number of main page requests served by the website instance",
})

func page(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintln(w, "Welcome to the homeworld self-hosting website.")
	requestsServed.Inc()
}

func main() {
	http.HandleFunc("/", page)
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
