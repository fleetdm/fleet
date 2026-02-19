package logging

import (
	"context"
	"sync"
)

// topicKeyType is an unexported type for the context key, ensuring no collisions.
type topicKeyType struct{}

// topicKey is the context key for storing a log topic.
var topicKey topicKeyType

// disabledTopics tracks which topics have been explicitly disabled.
// Topics are enabled by default — only topics in this map are disabled.
var (
	disabledTopics   = make(map[string]bool)
	disabledTopicsMu sync.RWMutex
)

// EnableTopic marks a topic as enabled (removes it from the disabled set).
func EnableTopic(name string) {
	disabledTopicsMu.Lock()
	delete(disabledTopics, name)
	disabledTopicsMu.Unlock()
}

// DisableTopic marks a topic as disabled.
func DisableTopic(name string) {
	disabledTopicsMu.Lock()
	disabledTopics[name] = true
	disabledTopicsMu.Unlock()
}

// SetTopicEnabled enables or disables a topic based on the enabled parameter.
func SetTopicEnabled(name string, enabled bool) {
	if enabled {
		EnableTopic(name)
	} else {
		DisableTopic(name)
	}
}

// TopicEnabled returns true unless the topic has been explicitly disabled.
func TopicEnabled(name string) bool {
	disabledTopicsMu.RLock()
	disabled := disabledTopics[name]
	disabledTopicsMu.RUnlock()
	return !disabled
}

// ResetTopics clears all disabled topics, re-enabling everything.
// This is intended for use in tests to ensure isolation.
func ResetTopics() {
	disabledTopicsMu.Lock()
	disabledTopics = make(map[string]bool)
	disabledTopicsMu.Unlock()
}

// ContextWithTopic returns a new context with the given log topic attached.
func ContextWithTopic(ctx context.Context, topic string) context.Context {
	return context.WithValue(ctx, topicKey, topic)
}

// TopicFromContext retrieves the log topic from the context.
// Returns an empty string if no topic is set.
func TopicFromContext(ctx context.Context) string {
	if topic, ok := ctx.Value(topicKey).(string); ok {
		return topic
	}
	return ""
}
