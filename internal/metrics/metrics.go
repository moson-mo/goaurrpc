package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Metrics to collect
	Requests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_requests",
			Help: "Number of /rpc requests per method, type and \"by\" parameter.",
		},
		[]string{"method", "type", "by"},
	)
	RequestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_requests_error",
			Help: "Number of /rpc requests that resulted in an error.",
		},
		[]string{"error"},
	)
	RateLimited = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_requests_rate_limited",
			Help: "Number of /rpc requests that ran into the rate-limit.",
		},
		[]string{},
	)
	HttpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rpc_request_duration_seconds",
			Help:    "Duration of /rpc requests.",
			Buckets: []float64{0.0001, 0.0003, 0.0005, 0.0007, 0.0009, 0.001, 0.003, 0.005, 0.01, 0.03, 0.05, 0.07, 0.1, 0.5, 1, 10},
		},
		[]string{})
	LastRefresh = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rpc_data_last_refresh",
			Help: "Last metadata refresh.",
		},
	)
	ResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rpc_response_size_bytes",
			Help:    "Response size of /rpc requests.",
			Buckets: []float64{500, 1000, 5000, 10000, 50000, 100000, 1000000, 2000000},
		},
		[]string{"type"})
)

// RegisterMetrics registers the different metrics that we want to collect
func RegisterMetrics() {
	prometheus.Register(Requests)
	prometheus.Register(RequestErrors)
	prometheus.Register(RateLimited)
	prometheus.Register(HttpDuration)
	prometheus.Register(LastRefresh)
	prometheus.Register(ResponseSize)
}
