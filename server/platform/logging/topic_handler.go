package logging

import (
	"context"
	"log/slog"
)

const logTopicAttrKey = "log_topic"

// TopicFilterHandler is a slog.Handler that filters log records based on
// the log_topic found in the log record's attributes, or on the handler
// itself if set using `WithAttrs`.
type TopicFilterHandler struct {
	base  slog.Handler
	topic string
}

// NewTopicFilterHandler wraps base with topic-aware filtering.
func NewTopicFilterHandler(base slog.Handler) *TopicFilterHandler {
	return &TopicFilterHandler{base: base}
}

// Enabled reports whether the handler handles records at the given level.
func (h *TopicFilterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

// Handle processes the log record. It performs a defensive re-check of the
// topic before delegating to the base handler.
func (h *TopicFilterHandler) Handle(ctx context.Context, r slog.Record) error {
	topic := h.topic
	// Override any log_topic attribute on the handler with one from the record's attributes, if present.
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == logTopicAttrKey {
			topic = a.Value.String()
			return false
		}
		return true
	})
	// Don't log if there's a disabled topic either on the handler or the record.
	if topic != "" && !TopicEnabled(topic) {
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
	// Check if the new attributes include a log_topic that should override the handler's current topic.
	for _, a := range attrs {
		if a.Key == logTopicAttrKey {
			return &TopicFilterHandler{base: h.base.WithAttrs(attrs), topic: a.Value.String()}
		}
	}
	return &TopicFilterHandler{base: h.base.WithAttrs(attrs), topic: h.topic}
}

// WithGroup returns a new TopicFilterHandler wrapping the base with the given group.
func (h *TopicFilterHandler) WithGroup(name string) slog.Handler {
	return &TopicFilterHandler{base: h.base.WithGroup(name), topic: h.topic}
}

// Ensure TopicFilterHandler implements slog.Handler at compile time.
var _ slog.Handler = (*TopicFilterHandler)(nil)
