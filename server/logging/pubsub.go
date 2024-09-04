package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type pubSubLogWriter struct {
	topic         *pubsub.Topic
	logger        log.Logger
	addAttributes bool
}

type PubSubAttributes struct {
	Name        string            `json:"name"`
	UnixTime    int64             `json:"unixTime"`
	Decorations map[string]string `json:"decorations"`
}

func NewPubSubLogWriter(projectId string, topicName string, addAttributes bool, logger log.Logger) (*pubSubLogWriter, error) {
	ctx := context.Background()

	client, err := pubsub.NewClient(ctx, projectId)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client: %w", err)
	}

	topic := client.Topic(topicName)

	level.Info(logger).Log(
		"msg", "GCP PubSub writer configured",
		"project", projectId,
		"topic", topicName,
		"add_attributes", addAttributes,
	)

	return &pubSubLogWriter{
		topic:         topic,
		logger:        logger,
		addAttributes: addAttributes,
	}, nil
}

func estimateAttributeSize(attributes map[string]string) int {
	var sz int
	for k, v := range attributes {
		sz += len(k) + len(v) + 2
	}
	return sz
}

func (w *pubSubLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	results := make([]*pubsub.PublishResult, len(logs))

	// Add all of the messages to the global pubsub queue
	for i, log := range logs {
		data, err := log.MarshalJSON()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal message into JSON")
		}

		attributes := make(map[string]string)

		if w.addAttributes {
			var unmarshaled PubSubAttributes

			if err := json.Unmarshal(log, &unmarshaled); err != nil {
				return ctxerr.Wrap(ctx, err, "unmarshalling log message JSON")
			}
			attributes["name"] = unmarshaled.Name
			attributes["timestamp"] = time.Unix(unmarshaled.UnixTime, 0).Format(time.RFC3339)
			for k, v := range unmarshaled.Decorations {
				attributes[k] = v
			}
		}

		if len(data)+estimateAttributeSize(attributes) > pubsub.MaxPublishRequestBytes {
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
			return ctxerr.Wrap(ctx, err, "pubsub publish")
		}
	}

	return nil
}
