package prometheus

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sifan077/PowerURL/config"
)

const (
	readHeaderTimeout = 5 * time.Second
	writeTimeout      = 10 * time.Second
	defaultPort       = 9090
)

// NewServer builds a basic HTTP server that exposes /metrics for Prometheus scraping.
func NewServer(cfg config.PrometheusConfig) *http.Server {
	port := cfg.Port
	if port == 0 {
		port = defaultPort
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
	}
}
