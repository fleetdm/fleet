package logging

import (
	"sync"
)

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
