package logging

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTopicTestLogger(buf *bytes.Buffer) *slog.Logger {
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler2 := NewTopicFilterHandler(handler)
	return slog.New(handler2)
}

func TestTopicHandler_NoTopic(t *testing.T) {
	t.Cleanup(ResetTopics)
	var buf bytes.Buffer
	logger := newTopicTestLogger(&buf)

	logger.InfoContext(context.Background(), "hello")
	assert.Contains(t, buf.String(), "hello")
}

func TestTopicHandler_EnabledTopic(t *testing.T) {
	t.Cleanup(ResetTopics)
	var buf bytes.Buffer
	logger := newTopicTestLogger(&buf)

	ctx := ContextWithTopic(context.Background(), "my-topic")
	logger.InfoContext(ctx, "hello from enabled topic")
	assert.Contains(t, buf.String(), "hello from enabled topic")
}

func TestTopicHandler_DisabledTopic(t *testing.T) {
	t.Cleanup(ResetTopics)
	var buf bytes.Buffer
	logger := newTopicTestLogger(&buf)

	DisableTopic("my-topic")
	ctx := ContextWithTopic(context.Background(), "my-topic")
	logger.InfoContext(ctx, "should not appear")
	assert.Empty(t, buf.String())
}

func TestTopicHandler_RespectsBaseLevel(t *testing.T) {
	t.Cleanup(ResetTopics)
	var buf bytes.Buffer
	// Base handler at Info level — Debug messages should be dropped regardless of topic.
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(NewTopicFilterHandler(handler))

	logger.DebugContext(context.Background(), "debug message")
	assert.Empty(t, buf.String())
}

func TestTopicHandler_WithAttrsPassesThrough(t *testing.T) {
	t.Cleanup(ResetTopics)
	var buf bytes.Buffer
	logger := newTopicTestLogger(&buf)

	logger = logger.With("key", "value")
	logger.InfoContext(context.Background(), "with attrs")
	output := buf.String()
	assert.Contains(t, output, "with attrs")
	assert.Contains(t, output, "key=value")
}

func TestTopicHandler_WithGroupPassesThrough(t *testing.T) {
	t.Cleanup(ResetTopics)
	var buf bytes.Buffer
	logger := newTopicTestLogger(&buf)

	logger = logger.WithGroup("grp")
	logger.InfoContext(context.Background(), "with group", "k", "v")
	output := buf.String()
	require.Contains(t, output, "grp.k=v")
}
