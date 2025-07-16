package collector

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var gauge *prometheus.GaugeVec

func initGauge() {
	gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "training_gauge",
		Help: "Monitoring training fitness",
	}, []string{"node", "namespace"})

	prometheus.MustRegister(gauge)
}

func serve() error {

	initGauge()

	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":5000", nil)
	if err != nil {
		return fmt.Errorf("unable to start server: %v", err)
	}

	return nil
}
