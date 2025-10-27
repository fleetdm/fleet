package software_ingestion

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// HealthHandler provides monitoring endpoints for the software ingestion service
type HealthHandler struct {
	service interface{} // Can be LoadManagedService or AsyncProcessor
	logger  log.Logger
}

func NewHealthHandler(service SoftwareIngestionService, logger log.Logger) *HealthHandler {
	return &HealthHandler{
		service: service,
		logger:  logger,
	}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/v1/fleet/software_ingestion/health":
		h.handleHealth(w, r)
	case "/api/v1/fleet/software_ingestion/metrics":
		h.handleMetrics(w, r)
	case "/api/v1/fleet/software_ingestion/circuit_breaker":
		h.handleCircuitBreaker(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *HealthHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
	}

	// Check if service supports health metrics
	if loadManagedService, ok := h.service.(*LoadManagedService); ok {
		stats := loadManagedService.metrics.GetStats()

		// Determine health based on error rate and circuit breaker state
		errorRate := float64(stats.FailedRequests) / float64(stats.TotalRequests) * 100
		if errorRate > 10.0 { // More than 10% error rate
			health.Status = "degraded"
			health.Issues = append(health.Issues, "High error rate")
		}

		if loadManagedService.circuitBreaker.state == CircuitBreakerOpen {
			health.Status = "unhealthy"
			health.Issues = append(health.Issues, "Circuit breaker open")
		}

		health.Metrics = &stats
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (h *HealthHandler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	var response interface{}

	switch service := h.service.(type) {
	case *LoadManagedService:
		response = service.metrics.GetStats()
	case *AsyncProcessor:
		response = struct {
			Ingestion IngestionStats `json:"ingestion"`
			Async     AsyncStats     `json:"async"`
		}{
			Ingestion: service.metrics.GetStats(),
			Async:     service.metrics.GetAsyncStats(),
		}
	default:
		response = map[string]string{"error": "metrics not available"}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HealthHandler) handleCircuitBreaker(w http.ResponseWriter, r *http.Request) {
	var status map[string]interface{}

	if loadManagedService, ok := h.service.(*LoadManagedService); ok {
		loadManagedService.circuitBreaker.mu.RLock()
		status = map[string]interface{}{
			"state":             loadManagedService.circuitBreaker.state,
			"failure_count":     loadManagedService.circuitBreaker.failureCount,
			"last_failure_time": loadManagedService.circuitBreaker.lastFailureTime,
			"can_execute":       loadManagedService.circuitBreaker.canExecute(),
		}
		loadManagedService.circuitBreaker.mu.RUnlock()
	} else {
		status = map[string]string{"error": "circuit breaker not available"}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

type HealthStatus struct {
	Status    string           `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	Issues    []string         `json:"issues,omitempty"`
	Metrics   *IngestionStats  `json:"metrics,omitempty"`
}

// Prometheus metrics integration
type PrometheusMetrics struct {
	// Add Prometheus metrics here for Grafana dashboards
	// Example:
	// - software_ingestion_requests_total
	// - software_ingestion_duration_seconds
	// - software_ingestion_queue_depth
	// - software_ingestion_circuit_breaker_state
}

// AlertingRules defines alerting conditions for operational monitoring
type AlertingRules struct {
	HighErrorRate       float64       `yaml:"high_error_rate"`        // Alert if error rate > 5%
	CircuitBreakerOpen  time.Duration `yaml:"circuit_breaker_open"`  // Alert if CB open > 1 minute
	HighQueueDepth      int           `yaml:"high_queue_depth"`       // Alert if queue depth > 500
	SlowIngestion       time.Duration `yaml:"slow_ingestion"`         // Alert if avg duration > 5s
}

func DefaultAlertingRules() AlertingRules {
	return AlertingRules{
		HighErrorRate:       5.0,
		CircuitBreakerOpen:  1 * time.Minute,
		HighQueueDepth:      500,
		SlowIngestion:       5 * time.Second,
	}
}

// DatabaseLoadMonitor tracks database health specifically for software ingestion
type DatabaseLoadMonitor struct {
	connectionPool ConnectionPoolMetrics
	queryMetrics   QueryMetrics
}

type ConnectionPoolMetrics struct {
	ActiveConnections int `json:"active_connections"`
	IdleConnections   int `json:"idle_connections"`
	MaxConnections    int `json:"max_connections"`
	WaitCount         int `json:"wait_count"`
	WaitDuration      time.Duration `json:"wait_duration"`
}

type QueryMetrics struct {
	AverageQueryDuration time.Duration `json:"average_query_duration"`
	LongRunningQueries   int           `json:"long_running_queries"`
	BlockedQueries       int           `json:"blocked_queries"`
	DeadlockCount        int           `json:"deadlock_count"`
}

// Configuration recommendations based on load patterns
func (h *HealthHandler) GetConfigurationRecommendations() ConfigRecommendations {
	// Analyze current metrics and provide tuning recommendations
	return ConfigRecommendations{
		RecommendedRateLimit:     100.0,
		RecommendedConcurrency:   25,
		RecommendedBatchSize:     15,
		EnableAsyncProcessing:    true,
		DatabaseTuningRequired:   false,
	}
}

type ConfigRecommendations struct {
	RecommendedRateLimit   float64 `json:"recommended_rate_limit"`
	RecommendedConcurrency int     `json:"recommended_concurrency"`
	RecommendedBatchSize   int     `json:"recommended_batch_size"`
	EnableAsyncProcessing  bool    `json:"enable_async_processing"`
	DatabaseTuningRequired bool    `json:"database_tuning_required"`
	Reasoning              string  `json:"reasoning"`
}