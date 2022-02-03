package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const (
	krContentTypeValue  = "application/vnd.kafka.json.v1+json"
	krContentTypeHeader = "Content-Type"
	krTimestampHeader   = "TimeStamp"
	krPublishTopicURL   = "%s/topics/%s"
	krCheckTopicURL     = "%s/topics/?topic=%s"
)

type KafkaRESTParams struct {
	KafkaProxyHost string
	KafkaTopic     string
	KafkaTimeout   int
}

type kafkaRESTProducer struct {
	client   *http.Client
	URL      string
	CheckURL string
}

type kafkaRecords struct {
	Records []kafkaValue `json:"records"`
}

type kafkaValue struct {
	Value json.RawMessage `json:"value"`
}

func NewKafkaRESTWriter(p *KafkaRESTParams) (*kafkaRESTProducer, error) {
	producer := &kafkaRESTProducer{
		URL:      fmt.Sprintf(krPublishTopicURL, p.KafkaProxyHost, p.KafkaTopic),
		CheckURL: fmt.Sprintf(krCheckTopicURL, p.KafkaProxyHost, p.KafkaTopic),
		client:   fleethttp.NewClient(fleethttp.WithTimeout(time.Duration(p.KafkaTimeout) * time.Second)),
	}

	return producer, producer.checkTopic()
}

func (l *kafkaRESTProducer) Write(ctx context.Context, logs []json.RawMessage) error {
	data := kafkaRecords{
		Records: make([]kafkaValue, len(logs)),
	}

	for i, log := range logs {
		data.Records[i] = kafkaValue{
			Value: log,
		}
	}

	output, err := json.Marshal(data)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "kafka rest marshal")
	}

	resp, err := l.post(l.URL, bytes.NewBuffer(output))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "kafka rest post")
	}
	defer resp.Body.Close()

	return checkResponse(resp)
}

func checkResponse(resp *http.Response) (err error) {
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Error: %d. %s", resp.StatusCode, string(body))
	}

	return nil
}

func (l *kafkaRESTProducer) checkTopic() (err error) {
	resp, err := l.client.Get(l.CheckURL)
	if err != nil {
		return fmt.Errorf("kafka rest topic check: %w", err)
	}
	defer resp.Body.Close()

	return checkResponse(resp)
}

func (l *kafkaRESTProducer) post(url string, buf *bytes.Buffer) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, buf)
	if err != nil {
		return nil, fmt.Errorf("kafka rest new request: %w", err)
	}

	now := float64(time.Now().UnixNano()) / float64(time.Second)
	req.Header.Set(krContentTypeHeader, krContentTypeValue)
	req.Header.Set(krTimestampHeader, fmt.Sprintf("%f", now))

	return l.client.Do(req)
}
