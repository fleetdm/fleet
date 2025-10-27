package software_ingestion

import (
	"context"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// AsyncProcessor handles asynchronous software ingestion to reduce database pressure
type AsyncProcessor struct {
	inner           SoftwareIngestionService
	queue           chan IngestionRequest
	workerPool      *WorkerPool
	batchAggregator *BatchAggregator
	metrics         *IngestionMetrics
	logger          log.Logger
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

// IngestionRequest represents a single software ingestion request
type IngestionRequest struct {
	Type         IngestionType
	HostID       uint
	Host         *fleet.Host
	SoftwareRows []map[string]string  // For osquery
	Software     []fleet.Software     // For MDM
	ResultChan   chan error           // For synchronous response if needed
	Priority     Priority
	Timestamp    time.Time
}

type IngestionType int

const (
	IngestionTypeOsquery IngestionType = iota
	IngestionTypeMDM
)

type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
)

// NewAsyncProcessor creates an async processor with configurable workers
func NewAsyncProcessor(
	inner SoftwareIngestionService,
	config LoadManagementConfig,
	logger log.Logger,
) *AsyncProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	ap := &AsyncProcessor{
		inner:           inner,
		queue:           make(chan IngestionRequest, config.AsyncQueueSize),
		workerPool:      NewWorkerPool(config.MaxConcurrentHosts, logger),
		batchAggregator: NewBatchAggregator(config, logger),
		metrics:         NewIngestionMetrics(),
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Start background processors
	ap.start()

	return ap
}

// IngestOsquerySoftware queues osquery software for async processing
func (ap *AsyncProcessor) IngestOsquerySoftware(ctx context.Context, hostID uint, host *fleet.Host, softwareRows []map[string]string) error {
	request := IngestionRequest{
		Type:         IngestionTypeOsquery,
		HostID:       hostID,
		Host:         host,
		SoftwareRows: softwareRows,
		Priority:     ap.determinePriority(len(softwareRows)),
		Timestamp:    time.Now(),
	}

	return ap.enqueueRequest(ctx, request)
}

// IngestMDMSoftware queues MDM software for async processing
func (ap *AsyncProcessor) IngestMDMSoftware(ctx context.Context, hostID uint, host *fleet.Host, software []fleet.Software) error {
	request := IngestionRequest{
		Type:     IngestionTypeMDM,
		HostID:   hostID,
		Host:     host,
		Software: software,
		Priority: ap.determinePriority(len(software)),
		Timestamp: time.Now(),
	}

	return ap.enqueueRequest(ctx, request)
}

// IngestOsquerySoftwareSync processes osquery software synchronously when needed
func (ap *AsyncProcessor) IngestOsquerySoftwareSync(ctx context.Context, hostID uint, host *fleet.Host, softwareRows []map[string]string) error {
	request := IngestionRequest{
		Type:         IngestionTypeOsquery,
		HostID:       hostID,
		Host:         host,
		SoftwareRows: softwareRows,
		ResultChan:   make(chan error, 1),
		Priority:     PriorityHigh, // Sync requests get high priority
		Timestamp:    time.Now(),
	}

	if err := ap.enqueueRequest(ctx, request); err != nil {
		return err
	}

	// Wait for result
	select {
	case err := <-request.ResultChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (ap *AsyncProcessor) enqueueRequest(ctx context.Context, request IngestionRequest) error {
	select {
	case ap.queue <- request:
		ap.metrics.RecordQueued()
		return nil
	case <-ctx.Done():
		ap.metrics.RecordRejected()
		return ctx.Err()
	default:
		// Queue is full
		ap.metrics.RecordRejected()
		level.Warn(ap.logger).Log(
			"msg", "software ingestion queue full, dropping request",
			"host_id", request.HostID,
			"type", request.Type,
		)
		if request.ResultChan != nil {
			request.ResultChan <- fleet.NewQueueFullError("software ingestion queue full")
		}
		return fleet.NewQueueFullError("software ingestion queue full")
	}
}

func (ap *AsyncProcessor) determinePriority(softwareCount int) Priority {
	if softwareCount > 50 {
		return PriorityLow // Large updates get lower priority
	} else if softwareCount > 10 {
		return PriorityNormal
	}
	return PriorityHigh // Small updates get processed quickly
}

func (ap *AsyncProcessor) start() {
	ap.wg.Add(1)
	go ap.processingLoop()
}

func (ap *AsyncProcessor) processingLoop() {
	defer ap.wg.Done()

	// Priority queues for different priority levels
	highPriorityQueue := make([]IngestionRequest, 0)
	normalPriorityQueue := make([]IngestionRequest, 0)
	lowPriorityQueue := make([]IngestionRequest, 0)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ap.ctx.Done():
			return

		case request := <-ap.queue:
			// Sort into priority queues
			switch request.Priority {
			case PriorityHigh:
				highPriorityQueue = append(highPriorityQueue, request)
			case PriorityNormal:
				normalPriorityQueue = append(normalPriorityQueue, request)
			case PriorityLow:
				lowPriorityQueue = append(lowPriorityQueue, request)
			}

		case <-ticker.C:
			// Process queues in priority order
			ap.processQueue(&highPriorityQueue)
			ap.processQueue(&normalPriorityQueue)
			ap.processQueue(&lowPriorityQueue)
		}
	}
}

func (ap *AsyncProcessor) processQueue(queue *[]IngestionRequest) {
	if len(*queue) == 0 {
		return
	}

	// Take up to batch size for processing
	batchSize := 5 // Process 5 at a time
	if len(*queue) < batchSize {
		batchSize = len(*queue)
	}

	batch := (*queue)[:batchSize]
	*queue = (*queue)[batchSize:]

	// Process batch
	for _, request := range batch {
		ap.workerPool.Submit(func() {
			ap.processRequest(request)
		})
	}
}

func (ap *AsyncProcessor) processRequest(request IngestionRequest) {
	start := time.Now()
	var err error

	switch request.Type {
	case IngestionTypeOsquery:
		err = ap.inner.IngestOsquerySoftware(ap.ctx, request.HostID, request.Host, request.SoftwareRows)
		ap.metrics.RecordIngestion(time.Since(start), len(request.SoftwareRows), err)
	case IngestionTypeMDM:
		err = ap.inner.IngestMDMSoftware(ap.ctx, request.HostID, request.Host, request.Software)
		ap.metrics.RecordIngestion(time.Since(start), len(request.Software), err)
	}

	// Send result if synchronous request
	if request.ResultChan != nil {
		request.ResultChan <- err
	}

	if err != nil {
		level.Warn(ap.logger).Log(
			"msg", "async software ingestion failed",
			"host_id", request.HostID,
			"type", request.Type,
			"duration", time.Since(start),
			"err", err,
		)
	}
}

// Stop gracefully shuts down the async processor
func (ap *AsyncProcessor) Stop() {
	ap.cancel()
	ap.wg.Wait()
	ap.workerPool.Stop()
}

// WorkerPool manages a pool of workers for parallel processing
type WorkerPool struct {
	workers chan func()
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewWorkerPool(size int, logger log.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	wp := &WorkerPool{
		workers: make(chan func(), size*2), // Buffer for smooth operation
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start workers
	for i := 0; i < size; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for {
		select {
		case <-wp.ctx.Done():
			return
		case fn := <-wp.workers:
			fn()
		}
	}
}

func (wp *WorkerPool) Submit(fn func()) {
	select {
	case wp.workers <- fn:
		// Submitted successfully
	case <-wp.ctx.Done():
		// Worker pool is shutting down
	default:
		// Workers are busy, execute synchronously to avoid blocking
		fn()
	}
}

func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
}

// BatchAggregator groups similar requests to reduce database load
type BatchAggregator struct {
	config LoadManagementConfig
	logger log.Logger
	// TODO: Implement intelligent batching based on host characteristics
}

func NewBatchAggregator(config LoadManagementConfig, logger log.Logger) *BatchAggregator {
	return &BatchAggregator{
		config: config,
		logger: logger,
	}
}

// Additional metrics for async processing
func (m *IngestionMetrics) RecordQueued() {
	// Implementation for tracking queued requests
}

func (m *IngestionMetrics) GetAsyncStats() AsyncStats {
	// Implementation for getting async-specific stats
	return AsyncStats{}
}

type AsyncStats struct {
	QueueDepth        int           `json:"queue_depth"`
	AverageQueueTime  time.Duration `json:"average_queue_time"`
	WorkerUtilization float64       `json:"worker_utilization"`
}

// Custom error types
func NewQueueFullError(message string) error {
	return &QueueFullError{Message: message}
}

type QueueFullError struct {
	Message string
}

func (e *QueueFullError) Error() string {
	return e.Message
}