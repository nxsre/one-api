package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// RatioFallback 费率回退打点 (model_ratio, completion_ratio)
	RatioFallback = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oneapi_model_ratio_fallback_total",
			Help: "Total number of model ratio fallback events.",
		},
		[]string{"model", "kind"}, // kind: model / completion
	)

	// RelayRequestTotal 转发请求打点
	RelayRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oneapi_relay_request_total",
			Help: "Total number of relay requests.",
		},
		[]string{"channel_type", "model", "status"}, // status: success / fail
	)

	// RelayRequestDuration 转发请求耗时
	RelayRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "oneapi_relay_request_duration_seconds",
			Help:    "Duration of relay requests in seconds.",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"channel_type", "model"},
	)

	// ClientDisconnectTotal 客户端断连打点
	ClientDisconnectTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oneapi_relay_client_disconnect_total",
			Help: "Total number of client disconnect events (e.g. broken pipe).",
		},
		[]string{"channel_id", "model"},
	)

	// CircuitFailureTotal 熔断打点
	CircuitFailureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oneapi_channel_circuit_failure_total",
			Help: "Total number of channel circuit failure events.",
		},
		[]string{"channel_id"},
	)

	// QuotaConsumedTotal 消耗额度打点（高基数，默认可选）
	QuotaConsumedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oneapi_quota_consumed_total",
			Help: "Total quota consumed.",
		},
		[]string{"user_id"},
	)
)

func init() {
	prometheus.MustRegister(RatioFallback)
	prometheus.MustRegister(RelayRequestTotal)
	prometheus.MustRegister(RelayRequestDuration)
	prometheus.MustRegister(ClientDisconnectTotal)
	prometheus.MustRegister(CircuitFailureTotal)
	prometheus.MustRegister(QuotaConsumedTotal)
}
