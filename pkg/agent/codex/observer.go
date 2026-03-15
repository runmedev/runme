package codex

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	toolsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/agent/tools/v1"
)

const (
	startupStatusSuccess = "success"
	startupStatusError   = "error"

	toolOutcomeSuccess   = "success"
	toolOutcomeBridgeErr = "bridge_error"
	toolOutcomeToolErr   = "tool_error"
	toolOutcomeEmpty     = "empty_output"
	toolOutcomeNoPayload = "no_payload"

	disconnectReasonReadError = "read_error"
	disconnectReasonClient    = "client_closed"
	disconnectReasonReplaced  = "replaced"
	disconnectReasonShutdown  = "server_shutdown"
)

type observer interface {
	ObserveAppServerStartup(duration time.Duration, status string)
	ObserveMCPToolCall(tool string, duration time.Duration, outcome string)
	IncBridgeDisconnect(reason string)
}

type prometheusObserver struct {
	appServerStartupLatency *prometheus.HistogramVec
	mcpToolCallLatency      *prometheus.HistogramVec
	bridgeDisconnectTotal   *prometheus.CounterVec
}

func newPrometheusObserver() *prometheusObserver {
	o := &prometheusObserver{
		appServerStartupLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "runme",
				Subsystem: "agent_codex",
				Name:      "app_server_startup_duration_seconds",
				Help:      "Latency of codex app-server startup attempts.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"status"},
		),
		mcpToolCallLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "runme",
				Subsystem: "agent_codex",
				Name:      "mcp_tool_call_duration_seconds",
				Help:      "Latency of codex MCP notebook tool calls dispatched through the bridge.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"tool", "outcome"},
		),
		bridgeDisconnectTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "runme",
				Subsystem: "agent_codex",
				Name:      "bridge_disconnect_total",
				Help:      "Count of codex bridge websocket disconnects.",
			},
			[]string{"reason"},
		),
	}

	prometheus.MustRegister(
		o.appServerStartupLatency,
		o.mcpToolCallLatency,
		o.bridgeDisconnectTotal,
	)
	return o
}

func (o *prometheusObserver) ObserveAppServerStartup(duration time.Duration, status string) {
	o.appServerStartupLatency.WithLabelValues(status).Observe(duration.Seconds())
}

func (o *prometheusObserver) ObserveMCPToolCall(tool string, duration time.Duration, outcome string) {
	o.mcpToolCallLatency.WithLabelValues(tool, outcome).Observe(duration.Seconds())
}

func (o *prometheusObserver) IncBridgeDisconnect(reason string) {
	o.bridgeDisconnectTotal.WithLabelValues(reason).Inc()
}

var (
	observerMu      sync.RWMutex
	defaultObserver observer = newPrometheusObserver()
)

func getObserver() observer {
	observerMu.RLock()
	defer observerMu.RUnlock()
	return defaultObserver
}

func observeAppServerStartup(duration time.Duration, err error) {
	status := startupStatusSuccess
	if err != nil {
		status = startupStatusError
	}
	getObserver().ObserveAppServerStartup(duration, status)
}

func observeMCPToolCall(tool string, duration time.Duration, outcome string) {
	getObserver().ObserveMCPToolCall(tool, duration, outcome)
}

func observeBridgeDisconnect(reason string) {
	getObserver().IncBridgeDisconnect(reason)
}

func toolCallOutcome(output *toolsv1.ToolCallOutput, err error) string {
	if err != nil {
		return toolOutcomeBridgeErr
	}
	if output == nil {
		return toolOutcomeEmpty
	}
	if output.GetStatus() == toolsv1.ToolCallOutput_STATUS_FAILED {
		return toolOutcomeToolErr
	}
	if output.GetClientError() != "" {
		return toolOutcomeToolErr
	}
	switch {
	case output.GetListCells() != nil:
		return toolOutcomeSuccess
	case output.GetGetCells() != nil:
		return toolOutcomeSuccess
	case output.GetUpdateCells() != nil:
		return toolOutcomeSuccess
	case output.GetExecuteCells() != nil:
		return toolOutcomeSuccess
	default:
		return toolOutcomeNoPayload
	}
}

func setObserverForTest(o observer) func() {
	observerMu.Lock()
	prev := defaultObserver
	defaultObserver = o
	observerMu.Unlock()

	return func() {
		observerMu.Lock()
		defaultObserver = prev
		observerMu.Unlock()
	}
}
