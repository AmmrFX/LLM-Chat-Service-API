package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// SetupRouter configures HTTP routes
func SetupRouter(handler *Handler, logger *zap.Logger) *mux.Router {
	router := mux.NewRouter()

	// Apply logging middleware
	router.Use(func(next http.Handler) http.Handler {
		return LoggingMiddleware(logger, next)
	})

	// Health check
	router.HandleFunc("/health", handler.HealthHandler).Methods("GET")

	// Chat endpoint
	router.HandleFunc("/chat", handler.ChatHandler).Methods("POST")

	// Metrics endpoint (bonus)
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Register Prometheus metrics
	registerMetrics()

	return router
}

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	chatRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "chat_requests_total",
			Help: "Total number of chat requests",
		},
		[]string{"stream_type"},
	)
)

func registerMetrics() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(chatRequestsTotal)
}
