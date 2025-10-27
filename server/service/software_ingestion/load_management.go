package software_ingestion

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"golang.org/x/time/rate"
)

// LoadManagedService wraps the SoftwareIngestionService with database load management
type LoadManagedService struct {
	inner               SoftwareIngestionService
	rateLimiter         *rate.Limiter
	circuitBreaker      *CircuitBreaker
	batchProcessor      *BatchProcessor
	metrics             *IngestionMetrics
	logger              log.Logger
	maxConcurrentHosts  int
	semaphore           chan struct{}
}

// LoadManagementConfig configures database load management
type LoadManagementConfig struct {
	// Rate limiting
	MaxRequestsPerSecond   float64       `yaml:"max_requests_per_second"`
	BurstSize             int           `yaml:"burst_size"`

	// Circuit breaker
	FailureThreshold      int           `yaml:"failure_threshold"`
	RecoveryTimeout       time.Duration `yaml:"recovery_timeout"`

	// Batching
	BatchSize             int           `yaml:"batch_size"`
	BatchTimeout          time.Duration `yaml:"batch_timeout"`
	MaxBatchDelay         time.Duration `yaml:"max_batch_delay"`

	// Concurrency
	MaxConcurrentHosts    int           `yaml:"max_concurrent_hosts"`

	// Timeouts
	DatabaseTimeout       time.Duration `yaml:"database_timeout"`

	// Async processing
	EnableAsyncProcessing bool          `yaml:"enable_async_processing"`
	AsyncQueueSize        int           `yaml:"async_queue_size"`
}

// DefaultLoadManagementConfig provides safe defaults
func DefaultLoadManagementConfig() LoadManagementConfig {
	return LoadManagementConfig{
		MaxRequestsPerSecond:   50.0,  // 50 software ingestions per second
		BurstSize:             100,    // Allow bursts up to 100
		FailureThreshold:      5,      // Circuit breaker trips after 5 failures
		RecoveryTimeout:       30 * time.Second,
		BatchSize:             10,     // Process 10 hosts per batch
		BatchTimeout:          100 * time.Millisecond,
		MaxBatchDelay:         1 * time.Second,
		MaxConcurrentHosts:    20,     // Limit concurrent host processing
		DatabaseTimeout:       10 * time.Second,
		EnableAsyncProcessing: true,
		AsyncQueueSize:        1000,
	}
}

// NewLoadManagedService creates a service with database load management
func NewLoadManagedService(
	inner SoftwareIngestionService,
	config LoadManagementConfig,
	logger log.Logger,
) *LoadManagedService {
	return &LoadManagedService{
		inner:              inner,
		rateLimiter:        rate.NewLimiter(rate.Limit(config.MaxRequestsPerSecond), config.BurstSize),
		circuitBreaker:     NewCircuitBreaker(config.FailureThreshold, config.RecoveryTimeout),
		batchProcessor:     NewBatchProcessor(config, logger),
		metrics:            NewIngestionMetrics(),
		logger:             logger,
		maxConcurrentHosts: config.MaxConcurrentHosts,
		semaphore:          make(chan struct{}, config.MaxConcurrentHosts),
	}
}

// IngestOsquerySoftware with load management
func (s *LoadManagedService) IngestOsquerySoftware(ctx context.Context, hostID uint, host *fleet.Host, softwareRows []map[string]string) error {
	// Rate limiting
	if err := s.rateLimiter.Wait(ctx); err != nil {
		s.metrics.RecordRateLimited()
		return ctxerr.Wrap(ctx, err, "rate limited software ingestion")
	}

	// Concurrency limiting
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		s.metrics.RecordRejected()
		return ctx.Err()
	}

	// Circuit breaker protection
	return s.circuitBreaker.Execute(func() error {
		start := time.Now()
		err := s.inner.IngestOsquerySoftware(ctx, hostID, host, softwareRows)

		duration := time.Since(start)
		s.metrics.RecordIngestion(duration, len(softwareRows), err)

		if err != nil {
			level.Warn(s.logger).Log(
				"msg", "software ingestion failed",
				"host_id", hostID,
				"software_count", len(softwareRows),
				"duration", duration,
				"err", err,
			)
		}

		return err
	})
}

// IngestMDMSoftware with load management
func (s *LoadManagedService) IngestMDMSoftware(ctx context.Context, hostID uint, host *fleet.Host, software []fleet.Software) error {
	// Apply same load management as osquery ingestion
	if err := s.rateLimiter.Wait(ctx); err != nil {
		s.metrics.RecordRateLimited()
		return ctxerr.Wrap(ctx, err, "rate limited MDM software ingestion")
	}

	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		s.metrics.RecordRejected()
		return ctx.Err()
	}

	return s.circuitBreaker.Execute(func() error {
		start := time.Now()
		err := s.inner.IngestMDMSoftware(ctx, hostID, host, software)

		duration := time.Since(start)
		s.metrics.RecordIngestion(duration, len(software), err)

		return err
	})
}

// CircuitBreaker protects against database overload
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitBreakerState
	failureCount     int
	lastFailureTime  time.Time
	failureThreshold int
	recoveryTimeout  time.Duration
}

type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

func NewCircuitBreaker(failureThreshold int, recoveryTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		recoveryTimeout:  recoveryTimeout,
		state:           CircuitBreakerClosed,
	}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.canExecute() {
		return ctxerr.New(context.Background(), "circuit breaker open - database overloaded")
	}

	err := fn()
	cb.recordResult(err)
	return err
}

func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		return time.Since(cb.lastFailureTime) > cb.recoveryTimeout
	case CircuitBreakerHalfOpen:
		return true
	}
	return false
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.failureCount >= cb.failureThreshold {
			cb.state = CircuitBreakerOpen
		}
	} else {
		// Success - reset failure count
		cb.failureCount = 0
		if cb.state == CircuitBreakerHalfOpen {
			cb.state = CircuitBreakerClosed
		}
	}
}

// BatchProcessor groups ingestion requests to reduce database load
type BatchProcessor struct {
	config LoadManagementConfig
	logger log.Logger
	// TODO: Implement batching logic for grouping multiple host updates
}

func NewBatchProcessor(config LoadManagementConfig, logger log.Logger) *BatchProcessor {
	return &BatchProcessor{
		config: config,
		logger: logger,
	}
}

// IngestionMetrics tracks performance and health metrics
type IngestionMetrics struct {
	mu                   sync.RWMutex
	totalRequests        int64
	successfulRequests   int64
	failedRequests       int64
	rateLimitedRequests  int64
	rejectedRequests     int64
	totalDuration        time.Duration
	maxDuration          time.Duration
	totalSoftwareItems   int64
}

func NewIngestionMetrics() *IngestionMetrics {
	return &IngestionMetrics{}
}

func (m *IngestionMetrics) RecordIngestion(duration time.Duration, softwareCount int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++
	m.totalDuration += duration
	m.totalSoftwareItems += int64(softwareCount)

	if duration > m.maxDuration {
		m.maxDuration = duration
	}

	if err != nil {
		m.failedRequests++
	} else {
		m.successfulRequests++
	}
}

func (m *IngestionMetrics) RecordRateLimited() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rateLimitedRequests++
}

func (m *IngestionMetrics) RecordRejected() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rejectedRequests++
}

func (m *IngestionMetrics) GetStats() IngestionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var avgDuration time.Duration
	if m.totalRequests > 0 {
		avgDuration = m.totalDuration / time.Duration(m.totalRequests)
	}

	return IngestionStats{
		TotalRequests:       m.totalRequests,
		SuccessfulRequests:  m.successfulRequests,
		FailedRequests:      m.failedRequests,
		RateLimitedRequests: m.rateLimitedRequests,
		RejectedRequests:    m.rejectedRequests,
		AverageDuration:     avgDuration,
		MaxDuration:         m.maxDuration,
		TotalSoftwareItems:  m.totalSoftwareItems,
	}
}

type IngestionStats struct {
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	RateLimitedRequests int64         `json:"rate_limited_requests"`
	RejectedRequests    int64         `json:"rejected_requests"`
	AverageDuration     time.Duration `json:"average_duration"`
	MaxDuration         time.Duration `json:"max_duration"`
	TotalSoftwareItems  int64         `json:"total_software_items"`
	ErrorRate           float64       `json:"error_rate"`
}

func (s IngestionStats) String() string {
	errorRate := float64(s.FailedRequests) / float64(s.TotalRequests) * 100
	return fmt.Sprintf(
		"Ingestion Stats: %d total, %d successful, %d failed (%.1f%% error rate), %d rate limited, avg duration: %v",
		s.TotalRequests, s.SuccessfulRequests, s.FailedRequests, errorRate, s.RateLimitedRequests, s.AverageDuration,
	)
}