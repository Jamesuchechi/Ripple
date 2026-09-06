package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// FanoutLatency tracks p50/p95/p99 latency for push and pull fanout execution.
	FanoutLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fanout_latency_seconds",
			Help:    "Latency of feed fanout operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"project_id", "fanout_type"},
	)

	// QueueDepth tracks current depth of queue partitions / batch buffers.
	QueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "queue_depth",
			Help: "Current depth of queues and batch buffers",
		},
		[]string{"project_id", "queue_name"},
	)

	// DeliveryErrors tracks total dispatch errors per channel and error type.
	DeliveryErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "delivery_errors_total",
			Help: "Total number of failed notification delivery attempts",
		},
		[]string{"project_id", "channel", "error_type"},
	)

	// DLQMessagesTotal tracks total messages routed to dead-letter storage.
	DLQMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dlq_messages_total",
			Help: "Total number of messages routed to the dead letter queue",
		},
		[]string{"project_id", "channel", "reason"},
	)

	// WebSocketConnections tracks active real-time WebSocket connections.
	WebSocketConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "websocket_connections_active",
			Help: "Number of active real-time WebSocket client connections",
		},
		[]string{"project_id"},
	)
)

func init() {
	prometheus.MustRegister(
		FanoutLatency,
		QueueDepth,
		DeliveryErrors,
		DLQMessagesTotal,
		WebSocketConnections,
	)
}

// Handler returns the http.Handler for scraping Prometheus metrics.
func Handler() http.Handler {
	return promhttp.Handler()
}
