package logging

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
)

const (
	natsTestLogCount = 1000
	natsTestSubject  = "test.logs"
	natsTestTimeout  = 5 * time.Second
)

// makeNatsClient creates a new NATS client.
func makeNatsClient(t *testing.T, url string) *nats.Conn {
	// Connect to the NATS server, in order to receive logs.
	nc, err := nats.Connect(url)

	// Ensure the NATS connection was created successfully.
	require.NoError(t, err)

	// Return the NATS connection.
	return nc
}

// makeNatsServer creates a new NATS server.
func makeNatsServer(t *testing.T) *server.Server {
	// Define the NATS server options, allowing the server to use a random port.
	opts := &server.Options{
		Host: "127.0.0.1",
		Port: -1,
	}

	// Create the NATS server.
	ns, err := server.NewServer(opts)

	// Ensure the NATS server was created successfully.
	require.NoError(t, err)

	// Start the NATS server.
	go ns.Start()

	// Ensure the NATS server is ready for connections within 5 seconds.
	require.True(t, ns.ReadyForConnections(natsTestTimeout))

	// Return the NATS server.
	return ns
}

func TestNatsLogWriter(t *testing.T) {
	// Create the NATS server and connection.
	ns := makeNatsServer(t)
	nc := makeNatsClient(t, ns.ClientURL())

	// Ensure the NATS server is shutdown when the test is done.
	defer ns.Shutdown()

	// Ensure the NATS connection is closed when the test is done.
	defer nc.Close()

	var expected, received []json.RawMessage

	for n := range natsTestLogCount {
		expected = append(expected, json.RawMessage(fmt.Sprintf(`{"foo":"bar %d"}`, n)))
	}

	// Create a wait group to wait for the logs to be received.
	var wg sync.WaitGroup

	// Add the number of expected logs to the wait group.
	wg.Add(len(expected))

	// Subscribe to the NATS subject.
	sub, err := nc.Subscribe(natsTestSubject, func(m *nats.Msg) {
		received = append(received, m.Data)

		wg.Done()
	})

	// Ensure the subscription was created successfully.
	require.NoError(t, err)

	// Ensure the subscription is unsubscribed when the test is done.
	defer sub.Unsubscribe()

	// Create the NATS log writer.
	writer, err := NewNatsLogWriter(
		ns.ClientURL(),
		natsTestSubject,
		"",
		"",
		"",
		"",
		"",
		false,
		natsTestTimeout,
		log.NewNopLogger(),
	)

	require.NoError(t, err)

	// Write the expected logs to the NATS log writer, and ensure it succeeds.
	require.NoError(t, writer.Write(t.Context(), expected))

	// Wait for all logs to be received.
	wg.Wait()

	// Ensure the received logs are equal to the expected logs.
	require.Equal(t, expected, received)
}
