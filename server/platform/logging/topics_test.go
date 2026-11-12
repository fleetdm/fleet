package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopicEnabledByDefault(t *testing.T) {
	t.Cleanup(ResetTopics)
	assert.True(t, TopicEnabled("unknown-topic"))
}

func TestDisableTopic(t *testing.T) {
	t.Cleanup(ResetTopics)
	DisableTopic("my-topic")
	assert.False(t, TopicEnabled("my-topic"))
}

func TestEnableTopicReenables(t *testing.T) {
	t.Cleanup(ResetTopics)
	DisableTopic("my-topic")
	assert.False(t, TopicEnabled("my-topic"))
	EnableTopic("my-topic")
	assert.True(t, TopicEnabled("my-topic"))
}

func TestResetTopics(t *testing.T) {
	t.Cleanup(ResetTopics)
	DisableTopic("a")
	DisableTopic("b")
	ResetTopics()
	assert.True(t, TopicEnabled("a"))
	assert.True(t, TopicEnabled("b"))
}
