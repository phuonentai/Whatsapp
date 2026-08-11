package metrics

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests processed.",
		},
		[]string{"method", "status", "path"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// excludedMetricPaths are endpoints excluded from the request histogram to
// avoid polluting latency signals: the SSE chat stream and /metrics itself.
var excludedMetricPaths = map[string]struct{}{
	"/api/example_cognitive/chat": {},
	"/metrics":                    {},
}

// RequestMetrics records a counter and latency histogram for each request.
func RequestMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if _, excluded := excludedMetricPaths[path]; excluded {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			strconv.Itoa(c.Writer.Status()),
			path,
		).Inc()
		httpRequestDuration.WithLabelValues(
			c.Request.Method,
			path,
		).Observe(time.Since(start).Seconds())
	}
}

func SetupPrometheus(router *gin.Engine) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
