package api

import (
	"net/http"

	"llm-chat-service/internal/api/handlers"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// SetupRouter configures HTTP routes
func SetupRouter(handler *handlers.Handler, logger *zap.Logger) *mux.Router {
	router := mux.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return LoggingMiddleware(logger, next)
	})

	router.HandleFunc("/health", handler.HealthHandler).Methods("GET")
	router.HandleFunc("/chat", handler.ChatHandler).Methods("GET", "POST")

	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

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
