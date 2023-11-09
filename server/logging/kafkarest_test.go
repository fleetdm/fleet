package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKafkaRestWrite(t *testing.T) {
	ctx := context.Background()

	var buf []byte
	var err error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, r.URL.Path, "/topics/foo")
		require.Equal(t, r.Header.Get("Content-Type"), "foobar")
		w.WriteHeader(200)
	}))
	defer server.Close()

	producer := &kafkaRESTProducer{
		client:           server.Client(),
		URL:              fmt.Sprintf(krPublishTopicURL, server.URL, "foo"),
		CheckURL:         fmt.Sprintf(krCheckTopicURL, server.URL, "foo"),
		ContentTypeValue: "foobar",
	}

	err = producer.Write(ctx, logs)
	require.NoError(t, err)

	expected := makeKafkaRecords(logs)
	var actual kafkaRecords
	err = json.Unmarshal(buf, &actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func makeKafkaRecords(messages []json.RawMessage) kafkaRecords {
	data := kafkaRecords{
		Records: make([]kafkaValue, len(messages)),
	}

	for i, log := range messages {
		data.Records[i] = kafkaValue{
			Value: log,
		}
	}
	return data
}
