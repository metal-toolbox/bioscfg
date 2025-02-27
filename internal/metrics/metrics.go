package metrics

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	rctypes "github.com/metal-toolbox/rivets/v2/condition"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	Endpoint = "0.0.0.0:9090"
)

var (
	EventsCounter *prometheus.CounterVec

	ConditionRunTimeSummary *prometheus.SummaryVec
	StoreQueryErrorCount    *prometheus.CounterVec

	NATSErrors *prometheus.CounterVec
)

func init() {
	EventsCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bioscfg_events_received",
			Help: "A counter metric to measure the total count of events received",
		},
		[]string{"valid", "response"}, // valid is true/false, response is ack/nack
	)

	ConditionRunTimeSummary = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "bioscfg_condition_duration_seconds",
			Help: "A summary metric to measure the total time spent in completing each condition",
		},
		[]string{"condition", "state"},
	)

	StoreQueryErrorCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bioscfg_store_query_error_count",
			Help: "A counter metric to measure the total count of errors querying the asset store.",
		},
		[]string{"storeKind", "queryKind"},
	)

	NATSErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bioscfg_nats_errors",
			Help: "A count of errors while trying to use NATS.",
		},
		[]string{"operation"},
	)
}

// ListenAndServe exposes prometheus metrics as /metrics
func ListenAndServe() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              Endpoint,
			ReadHeaderTimeout: 2 * time.Second, // nolint:gomnd // time duration value is clear as is.
		}

		if err := server.ListenAndServe(); err != nil {
			slog.Error("Failed to start metrics server", "error", err)
			os.Exit(1)
		}
	}()
}

// RegisterSpanEvent adds a span event along with the given attributes.
//
// event here is arbitrary and can be in the form of strings like - publishCondition, updateCondition etc
func RegisterSpanEvent(span trace.Span, condition *rctypes.Condition, controllerID, serverID, event string) {
	span.AddEvent(event, trace.WithAttributes(
		attribute.String("controllerID", controllerID),
		attribute.String("serverID", serverID),
		attribute.String("conditionID", condition.ID.String()),
		attribute.String("conditionKind", string(condition.Kind)),
	))
}

func NATSError(op string) {
	NATSErrors.WithLabelValues(op).Inc()
}
