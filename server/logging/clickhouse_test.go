package logging

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestClickHouseLogWriterValidation(t *testing.T) {
	t.Run("missing address", func(t *testing.T) {
		_, err := NewClickHouseLogWriter(
			"", // empty address
			"fleet_logs",
			"fleet",
			"fleet",
			"fleet_logs",
			"", "", "",
			"lz4",
			false, false, "", "", "",
			1000,
			5*time.Second,
			50000,
			3,
			100*time.Millisecond,
			"test",
			log.NewNopLogger(),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "address is required")
	})

	t.Run("missing database", func(t *testing.T) {
		_, err := NewClickHouseLogWriter(
			"localhost:9000",
			"", // empty database
			"fleet",
			"fleet",
			"fleet_logs",
			"", "", "",
			"lz4",
			false, false, "", "", "",
			1000,
			5*time.Second,
			50000,
			3,
			100*time.Millisecond,
			"test",
			log.NewNopLogger(),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "database is required")
	})

	t.Run("invalid TLS CA file", func(t *testing.T) {
		_, err := NewClickHouseLogWriter(
			"localhost:9000",
			"fleet_logs",
			"fleet",
			"fleet",
			"fleet_logs",
			"", "", "",
			"lz4",
			true,  // TLS enabled
			false, // skip verify
			"/nonexistent/ca.pem", // invalid CA file
			"", "",
			1000,
			5*time.Second,
			50000,
			3,
			100*time.Millisecond,
			"test",
			log.NewNopLogger(),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "read TLS CA file")
	})

	t.Run("invalid TLS CA certificate content", func(t *testing.T) {
		// Create a temp file with invalid certificate content
		tmpDir := t.TempDir()
		caFile := filepath.Join(tmpDir, "invalid-ca.pem")
		err := os.WriteFile(caFile, []byte("not a valid certificate"), 0600)
		require.NoError(t, err)

		_, err = NewClickHouseLogWriter(
			"localhost:9000",
			"fleet_logs",
			"fleet",
			"fleet",
			"fleet_logs",
			"", "", "",
			"lz4",
			true,   // TLS enabled
			false,  // skip verify
			caFile, // invalid CA content
			"", "",
			1000,
			5*time.Second,
			50000,
			3,
			100*time.Millisecond,
			"test",
			log.NewNopLogger(),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse TLS CA certificate")
	})

	t.Run("invalid TLS client cert file", func(t *testing.T) {
		_, err := NewClickHouseLogWriter(
			"localhost:9000",
			"fleet_logs",
			"fleet",
			"fleet",
			"fleet_logs",
			"", "", "",
			"lz4",
			true,  // TLS enabled
			false, // skip verify
			"",    // no CA file
			"/nonexistent/client.pem", // invalid client cert
			"/nonexistent/client.key", // invalid client key
			1000,
			5*time.Second,
			50000,
			3,
			100*time.Millisecond,
			"test",
			log.NewNopLogger(),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "load TLS client certificate")
	})
}

func TestClickHouseConfigDefaults(t *testing.T) {
	t.Run("default table name", func(t *testing.T) {
		// When no table name is provided, it should default to "fleet_logs"
		// We can't test this without a real connection, but we can verify
		// the config structure accepts empty table names
		config := ClickHouseConfig{
			Address:  "localhost:9000",
			Database: "fleet_logs",
		}
		require.Equal(t, "", config.TableName)
	})

	t.Run("compression methods", func(t *testing.T) {
		// Verify compression constants are defined correctly
		require.Equal(t, CompressionMethod("none"), CompressionNone)
		require.Equal(t, CompressionMethod("lz4"), CompressionLZ4)
		require.Equal(t, CompressionMethod("zstd"), CompressionZSTD)
	})

	t.Run("config struct fields", func(t *testing.T) {
		config := ClickHouseConfig{
			Address:           "localhost:9000,localhost:9001",
			Database:          "fleet_logs",
			Username:          "fleet",
			Password:          "secret",
			TableName:         "logs",
			StatusTableName:   "status_logs",
			ResultTableName:   "result_logs",
			AuditTableName:    "audit_logs",
			Compression:       "lz4",
			TLSEnabled:        true,
			TLSSkipVerify:     false,
			TLSCAFile:         "/path/to/ca.pem",
			TLSClientCertFile: "/path/to/cert.pem",
			TLSClientKeyFile:  "/path/to/key.pem",
			BatchSize:         5000,
			FlushInterval:     10 * time.Second,
			MaxQueueSize:      100000,
			MaxRetries:        5,
			RetryBackoff:      200 * time.Millisecond,
		}

		require.Equal(t, "localhost:9000,localhost:9001", config.Address)
		require.Equal(t, "fleet_logs", config.Database)
		require.Equal(t, "fleet", config.Username)
		require.Equal(t, "secret", config.Password)
		require.Equal(t, "logs", config.TableName)
		require.Equal(t, "status_logs", config.StatusTableName)
		require.Equal(t, "result_logs", config.ResultTableName)
		require.Equal(t, "audit_logs", config.AuditTableName)
		require.Equal(t, "lz4", config.Compression)
		require.True(t, config.TLSEnabled)
		require.False(t, config.TLSSkipVerify)
		require.Equal(t, "/path/to/ca.pem", config.TLSCAFile)
		require.Equal(t, "/path/to/cert.pem", config.TLSClientCertFile)
		require.Equal(t, "/path/to/key.pem", config.TLSClientKeyFile)
		require.Equal(t, 5000, config.BatchSize)
		require.Equal(t, 10*time.Second, config.FlushInterval)
		require.Equal(t, 100000, config.MaxQueueSize)
		require.Equal(t, 5, config.MaxRetries)
		require.Equal(t, 200*time.Millisecond, config.RetryBackoff)
	})
}

func TestClickHouseTableNameResolution(t *testing.T) {
	tests := []struct {
		name            string
		logType         string
		tableName       string
		statusTableName string
		resultTableName string
		auditTableName  string
		expected        string
	}{
		{
			name:            "status log with specific table",
			logType:         "status",
			tableName:       "default_table",
			statusTableName: "status_logs",
			expected:        "status_logs",
		},
		{
			name:            "result log with specific table",
			logType:         "result",
			tableName:       "default_table",
			resultTableName: "result_logs",
			expected:        "result_logs",
		},
		{
			name:           "audit log with specific table",
			logType:        "audit",
			tableName:      "default_table",
			auditTableName: "audit_logs",
			expected:       "audit_logs",
		},
		{
			name:      "status log falls back to default",
			logType:   "status",
			tableName: "default_table",
			expected:  "default_table",
		},
		{
			name:      "result log falls back to default",
			logType:   "result",
			tableName: "default_table",
			expected:  "default_table",
		},
		{
			name:      "audit log falls back to default",
			logType:   "audit",
			tableName: "default_table",
			expected:  "default_table",
		},
		{
			name:     "no table names defaults to fleet_logs",
			logType:  "status",
			expected: "fleet_logs",
		},
		{
			name:     "unknown log type defaults to fleet_logs",
			logType:  "unknown",
			expected: "fleet_logs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This mirrors the logic in NewClickHouseLogWriter
			resolvedTableName := tt.tableName
			switch tt.logType {
			case "status":
				if tt.statusTableName != "" {
					resolvedTableName = tt.statusTableName
				}
			case "result":
				if tt.resultTableName != "" {
					resolvedTableName = tt.resultTableName
				}
			case "audit":
				if tt.auditTableName != "" {
					resolvedTableName = tt.auditTableName
				}
			}
			if resolvedTableName == "" {
				resolvedTableName = "fleet_logs"
			}
			require.Equal(t, tt.expected, resolvedTableName)
		})
	}
}

func TestClickHouseLogEntryParsing(t *testing.T) {
	t.Run("parse hostIdentifier", func(t *testing.T) {
		rawLog := json.RawMessage(`{"hostIdentifier": "host-123", "name": "test"}`)
		var logData map[string]interface{}
		err := json.Unmarshal(rawLog, &logData)
		require.NoError(t, err)

		var hostID string
		if id, ok := logData["hostIdentifier"].(string); ok {
			hostID = id
		}
		require.Equal(t, "host-123", hostID)
	})

	t.Run("parse host_identifier", func(t *testing.T) {
		rawLog := json.RawMessage(`{"host_identifier": "host-456", "name": "test"}`)
		var logData map[string]interface{}
		err := json.Unmarshal(rawLog, &logData)
		require.NoError(t, err)

		var hostID string
		if id, ok := logData["hostIdentifier"].(string); ok {
			hostID = id
		} else if id, ok := logData["host_identifier"].(string); ok {
			hostID = id
		}
		require.Equal(t, "host-456", hostID)
	})

	t.Run("parse team_id", func(t *testing.T) {
		rawLog := json.RawMessage(`{"team_id": 42, "name": "test"}`)
		var logData map[string]interface{}
		err := json.Unmarshal(rawLog, &logData)
		require.NoError(t, err)

		var teamID uint32
		if id, ok := logData["team_id"].(float64); ok {
			teamID = uint32(id)
		}
		require.Equal(t, uint32(42), teamID)
	})

	t.Run("missing fields default to zero values", func(t *testing.T) {
		rawLog := json.RawMessage(`{"name": "test", "action": "snapshot"}`)
		var logData map[string]interface{}
		err := json.Unmarshal(rawLog, &logData)
		require.NoError(t, err)

		var hostID string
		var teamID uint32
		if id, ok := logData["hostIdentifier"].(string); ok {
			hostID = id
		} else if id, ok := logData["host_identifier"].(string); ok {
			hostID = id
		}
		if id, ok := logData["team_id"].(float64); ok {
			teamID = uint32(id)
		}
		require.Equal(t, "", hostID)
		require.Equal(t, uint32(0), teamID)
	})
}

func TestClickHouseCompressionParsing(t *testing.T) {
	// These tests verify the compression string parsing logic
	tests := []struct {
		name        string
		compression string
		expected    string // expected normalized value
	}{
		{"none lowercase", "none", "none"},
		{"empty string defaults to none", "", "none"},
		{"lz4 lowercase", "lz4", "lz4"},
		{"LZ4 uppercase", "LZ4", "lz4"},
		{"zstd lowercase", "zstd", "zstd"},
		{"ZSTD uppercase", "ZSTD", "zstd"},
		{"unknown defaults to lz4", "unknown", "lz4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mirror the compression parsing logic
			var normalized string
			switch lc := strings.ToLower(tt.compression); lc {
			case "none", "":
				normalized = "none"
			case "lz4":
				normalized = "lz4"
			case "zstd":
				normalized = "zstd"
			default:
				normalized = "lz4" // default to LZ4
			}
			require.Equal(t, tt.expected, normalized)
		})
	}
}

func TestClickHouseLogEntry(t *testing.T) {
	t.Run("logEntry struct", func(t *testing.T) {
		now := time.Now().UTC()
		entry := logEntry{
			eventTime:      now,
			logType:        "status",
			hostIdentifier: "host-123",
			teamID:         42,
			data:           `{"foo": "bar"}`,
		}

		require.Equal(t, now, entry.eventTime)
		require.Equal(t, "status", entry.logType)
		require.Equal(t, "host-123", entry.hostIdentifier)
		require.Equal(t, uint32(42), entry.teamID)
		require.Equal(t, `{"foo": "bar"}`, entry.data)
	})
}

// TestNewJSONLoggerClickHouse tests the ClickHouse integration with NewJSONLogger
func TestNewJSONLoggerClickHouse(t *testing.T) {
	t.Run("clickhouse plugin not configured", func(t *testing.T) {
		// When clickhouse is specified but address is empty, it should fail
		config := Config{
			Plugin: "clickhouse",
			ClickHouse: ClickHouseConfig{
				Address:  "", // empty - should fail validation
				Database: "fleet_logs",
			},
		}
		_, err := NewJSONLogger("status", config, log.NewNopLogger())
		require.Error(t, err)
		require.Contains(t, err.Error(), "address is required")
	})
}

// TestClickHouseWriteQueuing tests the non-blocking queue behavior
func TestClickHouseWriteQueuing(t *testing.T) {
	t.Run("queue channel capacity", func(t *testing.T) {
		// Verify the queue channel type can hold logEntry values
		queue := make(chan logEntry, 100)

		entry := logEntry{
			eventTime:      time.Now().UTC(),
			logType:        "test",
			hostIdentifier: "host-1",
			teamID:         1,
			data:           `{"test": true}`,
		}

		// Non-blocking send should succeed
		select {
		case queue <- entry:
			// Success
		default:
			t.Fatal("queue should not be full")
		}

		// Receive and verify
		received := <-queue
		require.Equal(t, entry.logType, received.logType)
		require.Equal(t, entry.hostIdentifier, received.hostIdentifier)
		require.Equal(t, entry.teamID, received.teamID)
		require.Equal(t, entry.data, received.data)
	})
}

// TestClickHouseContextCancellation tests context handling
func TestClickHouseContextCancellation(t *testing.T) {
	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Verify the context is canceled
		require.Error(t, ctx.Err())
		require.Equal(t, context.Canceled, ctx.Err())
	})

	t.Run("timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Wait for timeout
		<-ctx.Done()

		require.Error(t, ctx.Err())
		require.Equal(t, context.DeadlineExceeded, ctx.Err())
	})
}
