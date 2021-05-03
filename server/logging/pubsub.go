package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

const decorationPrefix = "d_"

type pubSubLogWriter struct {
	topic                *pubsub.Topic
	logger               log.Logger
	includeAttributes    []string
	decorationAttributes []string
}

func NewPubSubLogWriter(projectId string, topicName string, includeAttributes string, decorationAttributes string, logger log.Logger) (*pubSubLogWriter, error) {
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, projectId)
	if err != nil {
		return nil, errors.Wrap(err, "create pubsub client")
	}

	topic := client.Topic(topicName)

	level.Info(logger).Log(
		"msg", "GCP PubSub writer configured",
		"project", projectId,
		"topic", topicName,
		"include_attributes", includeAttributes,
		"decoration_attributes", decorationAttributes,
	)

	return &pubSubLogWriter{
		topic:                topic,
		logger:               logger,
		includeAttributes:    strings.Split(includeAttributes, ","),
		decorationAttributes: strings.Split(decorationAttributes, ","),
	}, nil
}

func asString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case float64, bool:
		return fmt.Sprint(v), true
	default:
		return "", false
	}
}

func extractAttributes(dest map[string]string, source map[string]interface{}, attributes []string, logger log.Logger) int {
	var attributeSize int

	for _, key := range attributes {
		value, ok := source[key]
		if !ok {
			continue
		}

		stringVal, ok := asString(value)
		if !ok {
			level.Warn(logger).Log(
				"msg", "not including pubsub attribute with composite typed value",
				"key", key,
				"type", fmt.Sprintf("%T", value),
			)
			continue
		}
		dest[key] = stringVal
		attributeSize += len(key) + len(stringVal) + 2
	}
	return attributeSize
}

func extractDecorations(dest map[string]string, source interface{}, attributes []string, logger log.Logger) int {
	var attributeSize int

	if source == nil || len(attributes) == 0 {
		return 0
	}

	decorations, ok := source.(map[string]interface{})
	if !ok {
		level.Warn(logger).Log(
			"msg", "decorations is not an object type",
			"type", fmt.Sprintf("%T", source),
		)
		return 0
	}

	for _, key := range attributes {
		value, ok := decorations[key]
		if !ok {
			continue
		}

		stringVal, ok := asString(value)
		if !ok {
			level.Warn(logger).Log(
				"msg", "not including pubsub attribute with composite typed value",
				"key", key,
				"type", fmt.Sprintf("%T", value),
			)
			continue
		}
		dest[decorationPrefix+key] = stringVal
		attributeSize += len(decorationPrefix) + len(key) + len(stringVal) + 2
	}
	return attributeSize
}

func (w *pubSubLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	results := make([]*pubsub.PublishResult, len(logs))

	// Add all of the messages to the global pubsub queue
	for i, log := range logs {
		data, err := log.MarshalJSON()
		if err != nil {
			return errors.Wrap(err, "marshal message into JSON")
		}

		attributes := make(map[string]string)
		var attributeSize int

		if len(w.includeAttributes) > 0 || len(w.decorationAttributes) > 0 {
			unmarshaled := make(map[string]interface{})
			if err := json.Unmarshal(log, &unmarshaled); err != nil {
				return errors.Wrap(err, "unmarshalling log message JSON")
			}

			attributeSize = extractAttributes(attributes, unmarshaled, w.includeAttributes, w.logger)
			attributeSize += extractDecorations(attributes, unmarshaled["decorations"], w.decorationAttributes, w.logger)
		}

		if len(data)+attributeSize > pubsub.MaxPublishRequestBytes {
			level.Info(w.logger).Log(
				"msg", "dropping log over 10MB PubSub limit",
				"size", len(data),
				"log", string(log[:100])+"...",
			)
			continue
		}

		message := &pubsub.Message{
			Data:       data,
			Attributes: attributes,
		}

		results[i] = w.topic.Publish(ctx, message)
	}

	// Wait for each message to be pushed to the server
	for _, result := range results {
		_, err := result.Get(ctx)
		if err != nil {
			return errors.Wrap(err, "pubsub publish")
		}
	}

	return nil
}
