package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ripple/internal/metrics"
)

func TestMetricsCollection(t *testing.T) {
	metrics.FanoutLatency.WithLabelValues("project_1", "push").Observe(0.045)
	metrics.QueueDepth.WithLabelValues("project_1", "fanout_queue").Set(12)
	metrics.DeliveryErrors.WithLabelValues("project_1", "webhook", "timeout").Inc()
	metrics.DLQMessagesTotal.WithLabelValues("project_1", "webhook", "max_retries_exceeded").Inc()
	metrics.WebSocketConnections.WithLabelValues("project_1").Inc()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handler := metrics.Handler()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	expectedMetrics := []string{
		"fanout_latency_seconds",
		"queue_depth",
		"delivery_errors_total",
		"dlq_messages_total",
		"websocket_connections_active",
	}

	for _, metricName := range expectedMetrics {
		if !strings.Contains(body, metricName) {
			t.Errorf("expected metrics response to contain %s", metricName)
		}
	}
}
