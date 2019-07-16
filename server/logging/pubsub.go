package logging

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

type pubSubLogWriter struct {
	topic  *pubsub.Topic
	logger log.Logger
}

func NewPubSubLogWriter(projectId string, topicName string, logger log.Logger) (*pubSubLogWriter, error) {
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
	)

	return &pubSubLogWriter{
		topic:  topic,
		logger: logger,
	}, nil
}

func (w *pubSubLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	results := make([]*pubsub.PublishResult, len(logs))

	// Add all of the messages to the global pubsub queue
	for i, log := range logs {
		data, err := log.MarshalJSON()
		if err != nil {
			return errors.Wrap(err, "marshal message into JSON")
		}

		if len(data) > pubsub.MaxPublishRequestBytes {
			level.Info(w.logger).Log(
				"msg", "dropping log over 10MB PubSub limit",
				"size", len(data),
				"log", string(log[:100])+"...",
			)
			continue
		}

		message := &pubsub.Message{
			Data: data,
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
