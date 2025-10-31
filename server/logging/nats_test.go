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
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"
)

const (
	natsTestLogCount      = 1000
	natsTestDirectSubject = "test.logs.direct"
	natsTestStreamSubject = "test.logs.stream"
	natsTestStreamName    = "test-logs-stream"
	natsTestTimeout       = 5 * time.Second
)

// makeNatsClient creates a new NATS client.
func makeNatsClient(t *testing.T, url string) *nats.Conn {
	t.Helper()

	// Connect to the NATS server, in order to receive logs.
	nc, err := nats.Connect(url)

	// Ensure the NATS connection was created successfully.
	require.NoError(t, err)

	// Return the NATS connection.
	return nc
}

// makeNatsServer creates a new NATS server.
func makeNatsServer(t *testing.T) *server.Server {
	t.Helper()

	// Define the NATS server options, allowing the server to use a random port.
	opts := &server.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		JetStream: true,
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

// makeNatsLogs creates a number of test logs.
func makeNatsLogs(t *testing.T) []json.RawMessage {
	t.Helper()

	var logs []json.RawMessage

	for n := range natsTestLogCount {
		logs = append(logs,
			json.RawMessage(fmt.Sprintf(`{"foo":"bar %d"}`, n)),
		)
	}

	return logs
}

func TestNatsLogRouter(t *testing.T) {
	// Define an abbreviated test log.
	testLog := json.RawMessage(`{
		"action": "snapshot",
		"decorations": {
		  "host_uuid": "85c1244f-9176-2445-8ceb-d6569dc1b417",
		  "hostname": "testhostname"
		},
		"hostIdentifier": "2d3b4dfc-9c1b-4617-ab07-c04dd3a754f0",
		"name": "pack/Global/testname",
		"numerics": false,
		"snapshot": []
	}`)

	t.Run("Constant", func(t *testing.T) {
		router := newNatsConstantRouter("test.logs")

		subject, err := router.Route(testLog)

		require.NoError(t, err)
		require.Equal(t, "test.logs", subject)
	})

	t.Run("Template", func(t *testing.T) {
		router := newNatsTemplateRouter("test.logs.{ $.name | split('/') | last() }.{ $.decorations.hostname }")

		subject, err := router.Route(testLog)

		require.NoError(t, err)
		require.Equal(t, "test.logs.testname.testhostname", subject)
	})
}

func TestNatsLogWriter(t *testing.T) {
	// Create the NATS server and connection.
	ns := makeNatsServer(t)
	nc := makeNatsClient(t, ns.ClientURL())

	// Ensure the NATS server is shutdown when the test is done.
	defer ns.Shutdown()

	// Ensure the NATS connection is closed when the test is done.
	defer nc.Close()

	t.Run("Direct", func(t *testing.T) {
		// Create a wait group to track outstanding logs.
		var wg sync.WaitGroup

		expected := makeNatsLogs(t)
		received := []json.RawMessage{}

		// Add the number of expected logs to the wait group.
		wg.Add(len(expected))

		// Subscribe to the NATS subject.
		_, err := nc.Subscribe(natsTestDirectSubject, func(m *nats.Msg) {
			received = append(received, m.Data)

			wg.Done()
		})

		// Ensure the subscription was created successfully.
		require.NoError(t, err)

		// Create the NATS log writer, specifying that the logs should be
		// published directly to the NATS subject, without using JetStream.
		writer, err := NewNatsLogWriter(
			ns.ClientURL(),
			natsTestDirectSubject,
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
	})

	t.Run("Stream", func(t *testing.T) {
		ctx := t.Context()

		// Create the JetStream context.
		js, err := jetstream.New(nc)

		require.NoError(t, err)

		// Create the in-memory stream.
		st, err := js.CreateStream(ctx, jetstream.StreamConfig{
			Name:     natsTestStreamName,
			Storage:  jetstream.MemoryStorage,
			Subjects: []string{natsTestStreamSubject},
		})

		require.NoError(t, err)

		// Create the NATS log writer, specifying that the logs should be
		// published to the JetStream stream.
		writer, err := NewNatsLogWriter(
			ns.ClientURL(),
			natsTestStreamSubject,
			"",
			"",
			"",
			"",
			"",
			true,
			natsTestTimeout,
			log.NewNopLogger(),
		)

		require.NoError(t, err)

		expected := makeNatsLogs(t)
		received := []json.RawMessage{}

		// Write the expected logs to the NATS log writer, and ensure it succeeds.
		require.NoError(t, writer.Write(ctx, expected))

		// Get the messages from the JetStream stream.
		for n := range uint64(len(expected)) {
			msg, err := st.GetMsg(ctx, n+1)

			require.NoError(t, err)

			received = append(received, msg.Data)
		}

		// Ensure the received logs are equal to the expected logs.
		require.Equal(t, expected, received)
	})
}
