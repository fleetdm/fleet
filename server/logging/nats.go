package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/nats-io/nats.go"
)

type natsPublisher interface {
	Publish(ctx context.Context, logs []json.RawMessage) error
}

type natsLogWriter struct {
	logger    log.Logger
	publisher natsPublisher
	timeout   time.Duration
}

// NewNatsLogWriter creates a new NATS log writer.
func NewNatsLogWriter(server, subject, credFile, nkeyFile, tlsClientCrtFile, tlsClientKeyFile, tlsCACrtFile string, jetstream bool, timeout time.Duration, logger log.Logger) (*natsLogWriter, error) {
	// Ensure the NATS server URL is set.
	if server == "" {
		return nil, errors.New("nats server URL missing")
	}

	// Ensure the NATS subject is set.
	if subject == "" {
		return nil, errors.New("nats subject missing")
	}

	// Ensure credentials file and nkey file are not used together.
	if credFile != "" && nkeyFile != "" {
		return nil, errors.New("nats credentials file and nkey file cannot be used together")
	}

	// Create the NATS connection options.
	opts := []nats.Option{nats.Name("NATS Fleet Publisher")}

	// Is a credentials file set?
	if credFile != "" {
		level.Debug(logger).Log(
			"msg", "using credentials file",
			"file", credFile,
		)

		opts = append(opts, nats.UserCredentials(credFile))
	}

	// Is a NKey seed file set?
	if nkeyFile != "" {
		level.Debug(logger).Log(
			"msg", "using NKey file",
			"file", nkeyFile,
		)

		opt, err := nats.NkeyOptionFromSeed(nkeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create nats nkey option: %w", err)
		}

		opts = append(opts, opt)
	}

	// Is a TLS client certificate and key set?
	if tlsClientCrtFile != "" && tlsClientKeyFile != "" {
		level.Debug(logger).Log(
			"msg", "using TLS client certificate and key files",
			"crt", tlsClientCrtFile,
			"key", tlsClientKeyFile,
		)

		opts = append(opts, nats.ClientCert(tlsClientCrtFile, tlsClientKeyFile))
	}

	// Is a CA certificate set?
	if tlsCACrtFile != "" {
		level.Debug(logger).Log(
			"msg", "using CA certificate file",
			"file", tlsCACrtFile,
		)

		opts = append(opts, nats.RootCAs(tlsCACrtFile))
	}

	level.Debug(logger).Log(
		"msg", "connecting to NATS server",
		"server", server,
	)

	// Connect to the NATS server.
	nc, err := nats.Connect(server, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats server: %w", err)
	}

	level.Debug(logger).Log(
		"msg", "connected to NATS server",
		"server", server,
	)

	// Create the NATS log writer.
	writer := &natsLogWriter{
		logger:  logger,
		timeout: timeout,
	}

	// Create the NATS publisher.
	if jetstream {
		writer.publisher, err = newNatsStreamPublisher(nc, subject)
	} else {
		writer.publisher, err = newNatsDirectPublisher(nc, subject)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create NATS publisher: %w", err)
	}

	return writer, nil
}

// Write publishes the logs to the NATS server by calling the publisher.
func (w *natsLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	// Create a context with the configured timeout.
	ctx, cancel := context.WithTimeout(ctx, w.timeout)

	defer cancel()

	return w.publisher.Publish(ctx, logs)
}

// natsDirectPublisher represents a NATS direct publisher.
type natsDirectPublisher struct {
	client  *nats.Conn
	subject string
}

// newNatsDirectPublisher creates a new direct publisher.
func newNatsDirectPublisher(client *nats.Conn, subject string) (*natsDirectPublisher, error) {
	return &natsDirectPublisher{client, subject}, nil
}

// Publish publishes the logs synchronously to the NATS server.
func (p *natsDirectPublisher) Publish(ctx context.Context, logs []json.RawMessage) error {
	for _, log := range logs {
		if err := p.client.Publish(p.subject, log); err != nil {
			return fmt.Errorf("failed to publish log: %w", err)
		}
	}

	return p.client.FlushWithContext(ctx)
}

// natsStreamPublisher represents a JetStream publisher.
type natsStreamPublisher struct {
	jetstream nats.JetStreamContext
	subject   string
}

// newNatsStreamPublisher creates a new JetStream publisher.
func newNatsStreamPublisher(client *nats.Conn, subject string) (*natsStreamPublisher, error) {
	js, err := client.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	return &natsStreamPublisher{
		jetstream: js,
		subject:   subject,
	}, nil
}

// Publish publishes the logs asynchronously using the JetStream API.
func (p *natsStreamPublisher) Publish(ctx context.Context, logs []json.RawMessage) error {
	for _, log := range logs {
		if _, err := p.jetstream.PublishAsync(p.subject, log); err != nil {
			return fmt.Errorf("failed to publish log: %w", err)
		}
	}

	// Wait for either the context to be done or the publish to be complete.
	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-p.jetstream.PublishAsyncComplete():
		return nil
	}
}
