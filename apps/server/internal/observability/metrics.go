package observability

import (
	"fmt"
	"io"
	"sync/atomic"
)

type Metrics struct {
	httpRequests      atomic.Uint64
	httpServerErrors  atomic.Uint64
	rateLimited       atomic.Uint64
	detectionsCreated atomic.Uint64
	detectionReplays  atomic.Uint64
	sseConnections    atomic.Int64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) IncHTTPRequest() {
	m.httpRequests.Add(1)
}

func (m *Metrics) IncHTTPServerError() {
	m.httpServerErrors.Add(1)
}

func (m *Metrics) IncRateLimited() {
	m.rateLimited.Add(1)
}

func (m *Metrics) IncDetectionCreated() {
	m.detectionsCreated.Add(1)
}

func (m *Metrics) IncDetectionReplay() {
	m.detectionReplays.Add(1)
}

func (m *Metrics) AddSSEConnection(delta int64) {
	m.sseConnections.Add(delta)
}

func (m *Metrics) WritePrometheus(w io.Writer) error {
	counters := []struct {
		name  string
		help  string
		value uint64
	}{
		{
			name:  "platewatch_http_requests_total",
			help:  "Total HTTP requests handled by the Go API.",
			value: m.httpRequests.Load(),
		},
		{
			name:  "platewatch_http_server_errors_total",
			help:  "Total HTTP responses with a 5xx status.",
			value: m.httpServerErrors.Load(),
		},
		{
			name:  "platewatch_rate_limited_total",
			help:  "Total requests rejected by rate limiting.",
			value: m.rateLimited.Load(),
		},
		{
			name:  "platewatch_detections_created_total",
			help:  "Total unique detection events created.",
			value: m.detectionsCreated.Load(),
		},
		{
			name:  "platewatch_detection_replays_total",
			help:  "Total idempotent detection retries served from existing events.",
			value: m.detectionReplays.Load(),
		},
	}

	for _, metric := range counters {
		if _, err := fmt.Fprintf(
			w,
			"# HELP %s %s\n# TYPE %s counter\n%s %d\n",
			metric.name,
			metric.help,
			metric.name,
			metric.name,
			metric.value,
		); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(
		w,
		"# HELP platewatch_sse_connections Current Server-Sent Events connections.\n"+
			"# TYPE platewatch_sse_connections gauge\n"+
			"platewatch_sse_connections %d\n",
		m.sseConnections.Load(),
	); err != nil {
		return err
	}
	return nil
}
