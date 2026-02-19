package logging

import (
	"context"
	"log/slog"
)

const logTopicAttrKey = "log_topic"

// TopicFilterHandler is a slog.Handler that filters log records based on
// the topic attached to the context. If a context carries a disabled topic,
// the record is silently dropped.
type TopicFilterHandler struct {
	base slog.Handler
}

// NewTopicFilterHandler wraps base with topic-aware filtering.
func NewTopicFilterHandler(base slog.Handler) *TopicFilterHandler {
	return &TopicFilterHandler{base: base}
}

// Enabled reports whether the handler handles records at the given level.
// Returns false early if the base handler wouldn't handle this level,
// or if the context carries a disabled topic.
func (h *TopicFilterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if !h.base.Enabled(ctx, level) {
		return false
	}
	if topic := TopicFromContext(ctx); topic != "" && !TopicEnabled(topic) {
		return false
	}
	return true
}

// Handle processes the log record. It performs a defensive re-check of the
// topic before delegating to the base handler.
func (h *TopicFilterHandler) Handle(ctx context.Context, r slog.Record) error {
	if topic := TopicFromContext(ctx); topic != "" && !TopicEnabled(topic) {
		return nil
	}
	var topic string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == logTopicAttrKey {
			topic = a.Value.String()
			return false
		}
		return true
	})
	if topic != "" && !TopicEnabled(topic) {
		return nil
	}
	return h.base.Handle(ctx, r)
}

// WithAttrs returns a new TopicFilterHandler wrapping the base with added attributes.
func (h *TopicFilterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TopicFilterHandler{base: h.base.WithAttrs(attrs)}
}

// WithGroup returns a new TopicFilterHandler wrapping the base with the given group.
func (h *TopicFilterHandler) WithGroup(name string) slog.Handler {
	return &TopicFilterHandler{base: h.base.WithGroup(name)}
}

// Ensure TopicFilterHandler implements slog.Handler at compile time.
var _ slog.Handler = (*TopicFilterHandler)(nil)
