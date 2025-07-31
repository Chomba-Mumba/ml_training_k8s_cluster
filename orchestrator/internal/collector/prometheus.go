package collector

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func initGauge() *prometheus.GaugeVec {
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "training_gauge",
		Help: "Monitoring training fitness",
	}, []string{"node", "namespace"})

	prometheus.MustRegister(gauge)

	return gauge
}

func servePromConn() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":5000", nil)
		if err != nil {
			panic(err)
		}
	}()
}
