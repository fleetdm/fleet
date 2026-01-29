// Package logging provides the ClickHouse log destination for Fleet.
package logging

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// CompressionMethod represents the compression algorithm to use.
type CompressionMethod string

const (
	CompressionNone CompressionMethod = "none"
	CompressionLZ4  CompressionMethod = "lz4"
	CompressionZSTD CompressionMethod = "zstd"
)

// ClickHouseConfig holds configuration for the ClickHouse log destination.
type ClickHouseConfig struct {
	// Address is the ClickHouse server address (host:port). Multiple addresses can be comma-separated.
	Address string

	// Database is the ClickHouse database name.
	Database string

	// Username for authentication.
	Username string

	// Password for authentication.
	Password string

	// TableName is the target table for logs (used if per-type tables not set).
	TableName string

	// Per-log-type table names (optional, overrides TableName)
	StatusTableName string
	ResultTableName string
	AuditTableName  string

	// Compression method: "none", "lz4" (default), "zstd"
	Compression string

	// TLS configuration
	TLSEnabled        bool
	TLSSkipVerify     bool
	TLSCAFile         string
	TLSClientCertFile string
	TLSClientKeyFile  string

	// Batching configuration
	BatchSize     int
	FlushInterval time.Duration
	MaxQueueSize  int

	// Retry configuration
	MaxRetries   int
	RetryBackoff time.Duration
}

// clickHouseLogWriter implements fleet.JSONLogger for ClickHouse.
type clickHouseLogWriter struct {
	conn   driver.Conn
	config ClickHouseConfig
	logger log.Logger
	name   string // "status", "result", or "audit"

	// Batching
	queue         chan logEntry
	batchSize     int
	flushInterval time.Duration

	// Metrics
	insertCount atomic.Uint64
	dropCount   atomic.Uint64
	failCount   atomic.Uint64

	// Health tracking
	healthy atomic.Bool

	// Lifecycle
	wg     sync.WaitGroup
	stopCh chan struct{}
	once   sync.Once
}

// logEntry represents a single log entry in the queue.
type logEntry struct {
	eventTime      time.Time
	logType        string
	hostIdentifier string
	teamID         uint32
	data           string
}

// NewClickHouseLogWriter creates a new ClickHouse log writer.
func NewClickHouseLogWriter(
	address string,
	database string,
	username string,
	password string,
	tableName string,
	statusTableName string,
	resultTableName string,
	auditTableName string,
	compression string,
	tlsEnabled bool,
	tlsSkipVerify bool,
	tlsCAFile string,
	tlsClientCertFile string,
	tlsClientKeyFile string,
	batchSize int,
	flushInterval time.Duration,
	maxQueueSize int,
	maxRetries int,
	retryBackoff time.Duration,
	name string,
	logger log.Logger,
) (*clickHouseLogWriter, error) {
	// Validate required parameters
	if address == "" {
		return nil, errors.New("ClickHouse address is required")
	}

	// Parse comma-separated addresses
	addresses := strings.Split(address, ",")
	if database == "" {
		return nil, errors.New("ClickHouse database is required")
	}

	// Resolve table name based on log type and per-type configuration
	resolvedTableName := tableName
	switch name {
	case "status":
		if statusTableName != "" {
			resolvedTableName = statusTableName
		}
	case "result":
		if resultTableName != "" {
			resolvedTableName = resultTableName
		}
	case "audit":
		if auditTableName != "" {
			resolvedTableName = auditTableName
		}
	}
	if resolvedTableName == "" {
		resolvedTableName = "fleet_logs"
	}

	// Parse compression method (default to LZ4)
	var compressionMethod clickhouse.CompressionMethod
	switch strings.ToLower(compression) {
	case "none", "":
		compressionMethod = clickhouse.CompressionNone
	case "lz4":
		compressionMethod = clickhouse.CompressionLZ4
	case "zstd":
		compressionMethod = clickhouse.CompressionZSTD
	default:
		compressionMethod = clickhouse.CompressionLZ4 // default to LZ4
	}

	// Apply defaults
	if batchSize <= 0 {
		batchSize = 5000
	}
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}
	if maxQueueSize <= 0 {
		maxQueueSize = 50000
	}
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if retryBackoff <= 0 {
		retryBackoff = 100 * time.Millisecond
	}

	// Build connection options
	opts := &clickhouse.Options{
		Addr: addresses,
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: compressionMethod,
		},
		DialTimeout:          10 * time.Second,
		MaxOpenConns:         10,
		MaxIdleConns:         5,
		ConnMaxLifetime:      time.Hour,
		ConnOpenStrategy:     clickhouse.ConnOpenInOrder,
		BlockBufferSize:      10,
		MaxCompressionBuffer: 10240,
	}

	// Configure TLS if enabled
	if tlsEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: tlsSkipVerify, //nolint:gosec // G402: User-configurable option for testing/dev environments
		}

		// Load CA certificate for server verification
		if tlsCAFile != "" {
			caCert, err := os.ReadFile(tlsCAFile)
			if err != nil {
				return nil, fmt.Errorf("read TLS CA file: %w", err)
			}

			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, errors.New("failed to parse TLS CA certificate")
			}
			tlsConfig.RootCAs = caCertPool
		}

		// Load client certificate for mTLS (must have both cert and key)
		if tlsClientCertFile != "" && tlsClientKeyFile != "" {
			clientCert, err := tls.LoadX509KeyPair(tlsClientCertFile, tlsClientKeyFile)
			if err != nil {
				return nil, fmt.Errorf("load TLS client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{clientCert}
		}

		opts.TLS = tlsConfig
	}

	// Open connection
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open ClickHouse connection: %w", err)
	}

	w := &clickHouseLogWriter{
		conn: conn,
		config: ClickHouseConfig{
			Address:           address,
			Database:          database,
			Username:          username,
			Password:          password,
			TableName:         resolvedTableName,
			StatusTableName:   statusTableName,
			ResultTableName:   resultTableName,
			AuditTableName:    auditTableName,
			Compression:       compression,
			TLSEnabled:        tlsEnabled,
			TLSSkipVerify:     tlsSkipVerify,
			TLSCAFile:         tlsCAFile,
			TLSClientCertFile: tlsClientCertFile,
			TLSClientKeyFile:  tlsClientKeyFile,
			BatchSize:         batchSize,
			FlushInterval:     flushInterval,
			MaxQueueSize:      maxQueueSize,
			MaxRetries:        maxRetries,
			RetryBackoff:      retryBackoff,
		},
		logger:        logger,
		name:          name,
		queue:         make(chan logEntry, maxQueueSize),
		batchSize:     batchSize,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}

	// Initial health check (non-fatal)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		level.Warn(logger).Log(
			"msg", "ClickHouse not reachable on startup, will retry",
			"err", err,
			"addresses", fmt.Sprintf("%v", addresses),
		)
		w.healthy.Store(false)
	} else {
		w.healthy.Store(true)

		// Ensure schema exists
		if err := w.ensureSchema(ctx); err != nil {
			level.Warn(logger).Log(
				"msg", "failed to ensure ClickHouse schema",
				"err", err,
			)
		}
	}

	// Start background flush goroutine
	w.wg.Add(1)
	go w.runBatcher()

	level.Info(logger).Log(
		"msg", "ClickHouse log writer initialized",
		"name", name,
		"addresses", fmt.Sprintf("%v", addresses),
		"database", database,
		"table", resolvedTableName,
		"compression", compression,
		"batch_size", batchSize,
		"flush_interval", flushInterval,
	)

	return w, nil
}

// Write implements fleet.JSONLogger.
// It enqueues log entries for async batch writing to ClickHouse.
func (w *clickHouseLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	for _, rawLog := range logs {
		entry := logEntry{
			eventTime: time.Now().UTC(),
			logType:   w.name,
			data:      string(rawLog),
		}

		// Try to extract host_identifier and team_id from the log
		var logData map[string]interface{}
		if err := json.Unmarshal(rawLog, &logData); err == nil {
			if hostID, ok := logData["hostIdentifier"].(string); ok {
				entry.hostIdentifier = hostID
			} else if hostID, ok := logData["host_identifier"].(string); ok {
				entry.hostIdentifier = hostID
			}

			if teamID, ok := logData["team_id"].(float64); ok {
				entry.teamID = uint32(teamID)
			}
		}

		// Non-blocking enqueue
		select {
		case w.queue <- entry:
			// Successfully enqueued
		default:
			// Queue full, drop entry
			w.dropCount.Add(1)
			level.Warn(w.logger).Log(
				"msg", "ClickHouse queue full, dropping log entry",
				"name", w.name,
				"queue_size", len(w.queue),
			)
		}
	}

	return nil
}

// runBatcher is the background goroutine that batches and flushes logs.
func (w *clickHouseLogWriter) runBatcher() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	batch := make([]logEntry, 0, w.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		batchToSend := batch
		batch = make([]logEntry, 0, w.batchSize)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := w.writeBatch(ctx, batchToSend); err != nil {
			level.Error(w.logger).Log(
				"msg", "failed to write batch to ClickHouse",
				"err", err,
				"batch_size", len(batchToSend),
				"name", w.name,
			)
			w.failCount.Add(uint64(len(batchToSend)))
		} else {
			w.insertCount.Add(uint64(len(batchToSend)))
		}
	}

	for {
		select {
		case <-w.stopCh:
			// Drain remaining entries
			draining := true
			for draining {
				select {
				case entry := <-w.queue:
					batch = append(batch, entry)
					if len(batch) >= w.batchSize {
						flush()
					}
				default:
					draining = false
				}
			}
			flush()
			return

		case <-ticker.C:
			flush()

		case entry := <-w.queue:
			batch = append(batch, entry)
			if len(batch) >= w.batchSize {
				flush()
			}
		}
	}
}

// writeBatch writes a batch of log entries to ClickHouse with retry.
func (w *clickHouseLogWriter) writeBatch(ctx context.Context, entries []logEntry) error {
	var lastErr error

	for attempt := 1; attempt <= w.config.MaxRetries; attempt++ {
		err := w.doWriteBatch(ctx, entries)
		if err == nil {
			w.healthy.Store(true)
			return nil
		}

		lastErr = err
		w.healthy.Store(false)

		if attempt < w.config.MaxRetries {
			backoff := w.config.RetryBackoff * time.Duration(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	return lastErr
}

// doWriteBatch performs the actual batch insert.
func (w *clickHouseLogWriter) doWriteBatch(ctx context.Context, entries []logEntry) error {
	query := fmt.Sprintf(
		"INSERT INTO %s (event_time, log_type, host_identifier, team_id, data)",
		w.config.TableName,
	)

	batch, err := w.conn.PrepareBatch(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, e := range entries {
		if err := batch.Append(
			e.eventTime,
			e.logType,
			e.hostIdentifier,
			e.teamID,
			e.data,
		); err != nil {
			return fmt.Errorf("append row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send batch: %w", err)
	}

	return nil
}

// ensureSchema creates the log table if it doesn't exist.
func (w *clickHouseLogWriter) ensureSchema(ctx context.Context) error {
	ddl := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			event_id        UUID DEFAULT generateUUIDv4(),
			event_time      DateTime64(3),
			ingest_time     DateTime64(3) DEFAULT now64(3),
			log_type        LowCardinality(String),
			host_identifier String DEFAULT '',
			team_id         UInt32 DEFAULT 0,
			data            String
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(event_time)
		ORDER BY (log_type, event_time, host_identifier)
		TTL toDateTime(event_time) + INTERVAL 90 DAY DELETE
		SETTINGS index_granularity = 8192
	`, w.config.TableName)

	if err := w.conn.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	return nil
}

// Close gracefully shuts down the writer.
func (w *clickHouseLogWriter) Close() error {
	w.once.Do(func() {
		close(w.stopCh)
		w.wg.Wait()
	})
	return w.conn.Close()
}

// Stats returns current writer statistics.
func (w *clickHouseLogWriter) Stats() (inserted, dropped, failed uint64, queueLen int, healthy bool) {
	return w.insertCount.Load(),
		w.dropCount.Load(),
		w.failCount.Load(),
		len(w.queue),
		w.healthy.Load()
}
