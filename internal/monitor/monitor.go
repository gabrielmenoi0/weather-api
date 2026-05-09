package monitor

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "weather_api_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "endpoint", "status"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "weather_api_request_duration_seconds",
			Help: "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	ExternalAPIErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "weather_api_external_errors_total",
			Help: "Total errors from external weather API calls.",
		},
		[]string{"service"},
	)

	ExternalAPIDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "weather_api_external_duration_seconds",
			Help: "Duration of external weather API calls.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "operation"},
	)
)

func init() {
	prometheus.MustRegister(
		RequestsTotal,
		RequestDuration,
		ExternalAPIErrors,
		ExternalAPIDuration,
	)
}

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unknown"
		}

		status := http.StatusText(c.Writer.Status())
		RequestsTotal.WithLabelValues(c.Request.Method, endpoint, status).Inc()
		RequestDuration.WithLabelValues(c.Request.Method, endpoint).Observe(time.Since(start).Seconds())
	}
}

func RegisterRoutes(r *gin.Engine, appName, appEnv string) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "UP",
			"app": appName,
			"env": appEnv,
		})
	})

	r.GET("/health/liveness", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	r.GET("/health/readiness", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
