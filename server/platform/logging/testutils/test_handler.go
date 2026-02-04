// Package testutils provides testing utilities for the logging package.
package testutils

import (
	"context"
	"log/slog"
	"slices"
	"sync"
)

// TestHandler is a slog.Handler that captures log records for testing.
// It allows tests to verify logging behavior without parsing serialized output.
type TestHandler struct {
	// mu and records are pointers so they're shared across WithAttrs/WithGroup calls.
	// This mirrors how real handlers share their output destination (io.Writer)
	// while maintaining independent configuration (attrs, groups).
	mu      *sync.Mutex
	records *[]slog.Record
	attrs   []slog.Attr
	group   string
	minLevel slog.Level
}

// NewTestHandler creates a new TestHandler that accepts all log levels.
func NewTestHandler() *TestHandler {
	return &TestHandler{
		mu:      &sync.Mutex{},
		records: &[]slog.Record{},
		minLevel: slog.LevelDebug,
	}
}

// NewTestHandlerWithLevel creates a new TestHandler that only accepts
// logs at or above the specified level.
func NewTestHandlerWithLevel(level slog.Level) *TestHandler {
	return &TestHandler{
		mu:       &sync.Mutex{},
		records:  &[]slog.Record{},
		minLevel: level,
	}
}

// Enabled returns true if the level is at or above the handler's minimum level.
func (h *TestHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel
}

// Handle captures the record for later inspection.
func (h *TestHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Clone the record to avoid issues with reused records
	clone := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	// Add pre-set attrs first
	clone.AddAttrs(h.attrs...)

	// Then add record attrs
	r.Attrs(func(a slog.Attr) bool {
		clone.AddAttrs(a)
		return true
	})

	*h.records = append(*h.records, clone)
	return nil
}

// WithAttrs returns a new handler with the given attributes added.
func (h *TestHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	return &TestHandler{
		mu:      h.mu,
		records: h.records,
		attrs:   slices.Concat(h.attrs, attrs),
		group:   h.group,
		minLevel: h.minLevel,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *TestHandler) WithGroup(name string) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()
	return &TestHandler{
		mu:      h.mu,
		records: h.records,
		attrs:   h.attrs,
		group:   name,
		minLevel: h.minLevel,
	}
}

// Records returns all captured log records.
func (h *TestHandler) Records() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(*h.records)
}

// Clear removes all captured records.
func (h *TestHandler) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	*h.records = nil
}

// LastRecord returns the most recently captured record, or nil if none.
func (h *TestHandler) LastRecord() *slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(*h.records) == 0 {
		return nil
	}
	r := (*h.records)[len(*h.records)-1]
	return &r
}

// RecordAttrs extracts all attributes from a record as a map.
func RecordAttrs(r *slog.Record) map[string]any {
	attrs := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	return attrs
}
