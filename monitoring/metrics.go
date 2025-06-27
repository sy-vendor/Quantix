package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 请求计数器
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "quantix_requests_total",
			Help: "总请求数",
		},
		[]string{"method", "endpoint", "status"},
	)

	// 请求延迟
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "quantix_request_duration_seconds",
			Help:    "请求延迟",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// 数据获取计数器
	DataFetchTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "quantix_data_fetch_total",
			Help: "数据获取次数",
		},
		[]string{"source", "status"},
	)

	// 数据获取延迟
	DataFetchDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "quantix_data_fetch_duration_seconds",
			Help:    "数据获取延迟",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"source"},
	)

	// 预测准确率
	PredictionAccuracy = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "quantix_prediction_accuracy",
			Help: "预测准确率",
		},
		[]string{"model", "stock"},
	)

	// 活跃连接数
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "quantix_active_connections",
			Help: "活跃WebSocket连接数",
		},
	)

	// 缓存命中率
	CacheHitRatio = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "quantix_cache_hit_ratio",
			Help: "缓存命中率",
		},
	)

	// 内存使用量
	MemoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "quantix_memory_usage_bytes",
			Help: "内存使用量",
		},
		[]string{"type"},
	)

	// 错误计数器
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "quantix_errors_total",
			Help: "错误总数",
		},
		[]string{"type", "component"},
	)
)

// RecordRequest 记录请求
func RecordRequest(method, endpoint, status string, duration float64) {
	RequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	RequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// RecordDataFetch 记录数据获取
func RecordDataFetch(source, status string, duration float64) {
	DataFetchTotal.WithLabelValues(source, status).Inc()
	DataFetchDuration.WithLabelValues(source).Observe(duration)
}

// RecordPredictionAccuracy 记录预测准确率
func RecordPredictionAccuracy(model, stock string, accuracy float64) {
	PredictionAccuracy.WithLabelValues(model, stock).Set(accuracy)
}

// RecordActiveConnections 记录活跃连接数
func RecordActiveConnections(count int) {
	ActiveConnections.Set(float64(count))
}

// RecordCacheHitRatio 记录缓存命中率
func RecordCacheHitRatio(ratio float64) {
	CacheHitRatio.Set(ratio)
}

// RecordMemoryUsage 记录内存使用量
func RecordMemoryUsage(memoryType string, bytes int64) {
	MemoryUsage.WithLabelValues(memoryType).Set(float64(bytes))
}

// RecordError 记录错误
func RecordError(errorType, component string) {
	ErrorsTotal.WithLabelValues(errorType, component).Inc()
}
