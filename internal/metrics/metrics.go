package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Metrics to collect
	Requests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_requests",
			Help: "Number of /rpc requests per method, type and \"by\" parameter.",
		},
		[]string{"path", "method", "type", "by"},
	)
	RequestErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_requests_error",
			Help: "Number of /rpc requests that resulted in an error.",
		},
		[]string{"method", "error"},
	)
	RateLimited = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_requests_rate_limited",
			Help: "Number of /rpc requests that ran into the rate-limit.",
		},
		[]string{},
	)
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "rpc_request_duration_seconds",
		Help: "Duration of /rpc requests.",
	}, []string{"path"})
)

// Prometheus middleware wrapping /rpc calls
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(r.URL.Path))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
	})
}

// RegisterMetrics registers the different metrics that we want to collect
func RegisterMetrics() {
	prometheus.Register(Requests)
	prometheus.Register(RequestErrors)
	prometheus.Register(RateLimited)
	prometheus.Register(httpDuration)

	// remove go specific metrics
	prometheus.Unregister(collectors.NewGoCollector())
}
