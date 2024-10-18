package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	kinesisMaxRetries = 8

	// See
	// https://docs.aws.amazon.com/sdk-for-go/api/service/kinesis/#Kinesis.PutRecords
	// for documentation on limits.
	kinesisMaxRecordsInBatch = 500
	kinesisMaxSizeOfRecord   = 1000 * 1000     // 1,000 KB
	kinesisMaxSizeOfBatch    = 5 * 1000 * 1000 // 5 MB
)

type kinesisLogWriter struct {
	client kinesisiface.KinesisAPI
	stream string
	logger log.Logger
	rand   *rand.Rand
}

func NewKinesisLogWriter(region, endpointURL, id, secret, stsAssumeRoleArn, stsExternalID, stream string, logger log.Logger) (*kinesisLogWriter, error) {
	conf := &aws.Config{
		Region:   &region,
		Endpoint: &endpointURL, // empty string or nil will use default values
	}

	// Only provide static credentials if we have them
	// otherwise use the default credentials provider chain
	if id != "" && secret != "" {
		conf.Credentials = credentials.NewStaticCredentials(id, secret, "")
	}

	sess, err := session.NewSession(conf)
	if err != nil {
		return nil, fmt.Errorf("create Kinesis client: %w", err)
	}

	if stsAssumeRoleArn != "" {
		creds := stscreds.NewCredentials(sess, stsAssumeRoleArn, func(provider *stscreds.AssumeRoleProvider) {
			if stsExternalID != "" {
				provider.ExternalID = &stsExternalID
			}
		})
		conf.Credentials = creds

		sess, err = session.NewSession(conf)

		if err != nil {
			return nil, fmt.Errorf("create Kinesis client: %w", err)
		}
	}
	client := kinesis.New(sess)

	// This will be used to generate random partition keys to balance
	// records across Kinesis shards.
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))

	k := &kinesisLogWriter{
		client: client,
		stream: stream,
		logger: logger,
		rand:   rand,
	}
	if err := k.validateStream(); err != nil {
		return nil, fmt.Errorf("create Kinesis writer: %w", err)
	}
	return k, nil
}

func (k *kinesisLogWriter) validateStream() error {
	out, err := k.client.DescribeStream(
		&kinesis.DescribeStreamInput{
			StreamName: &k.stream,
		},
	)
	if err != nil {
		return fmt.Errorf("describe stream %s: %w", k.stream, err)
	}

	if (*out.StreamDescription.StreamStatus) != kinesis.StreamStatusActive {
		return fmt.Errorf("stream %s not active", k.stream)
	}

	return nil
}

func (k *kinesisLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	var records []*kinesis.PutRecordsRequestEntry
	totalBytes := 0
	for _, log := range logs {
		// so we get nice NDJSON
		log = append(log, '\n')
		// Evenly distribute logs across shards by assigning each
		// kinesis.PutRecordsRequestEntry a random partition key.
		partitionKey := fmt.Sprint(k.rand.Intn(256))

		// We don't really have a good option for what to do with logs
		// that are too big for Kinesis. This behavior is consistent
		// with osquery's behavior in the Kinesis logger plugin, and
		// the beginning bytes of the log should help the Fleet admin
		// diagnose the query generating huge results.
		if len(log)+len(partitionKey) > kinesisMaxSizeOfRecord {
			level.Info(k.logger).Log(
				"msg", "dropping log over 1MB Kinesis limit",
				"size", len(log),
				"log", string(log[:100])+"...",
			)
			continue
		}

		// If adding this log will exceed the limit on number of
		// records in the batch, or the limit on total size of the
		// records in the batch, we need to push this batch before
		// adding any more.
		if len(records) >= kinesisMaxRecordsInBatch ||
			totalBytes+len(log)+len(partitionKey) > kinesisMaxSizeOfBatch {
			if err := k.putRecords(0, records); err != nil {
				return ctxerr.Wrap(ctx, err, "put records")
			}
			totalBytes = 0
			records = nil
		}

		records = append(records, &kinesis.PutRecordsRequestEntry{Data: []byte(log), PartitionKey: aws.String(partitionKey)})
		totalBytes += len(log) + len(partitionKey)
	}

	// Push the final batch
	if len(records) > 0 {
		if err := k.putRecords(0, records); err != nil {
			return ctxerr.Wrap(ctx, err, "put records")
		}
	}

	return nil
}

func (k *kinesisLogWriter) putRecords(try int, records []*kinesis.PutRecordsRequestEntry) error {
	if try > 0 {
		time.Sleep(100 * time.Millisecond * time.Duration(math.Pow(2.0, float64(try))))
	}
	input := &kinesis.PutRecordsInput{
		StreamName: &k.stream,
		Records:    records,
	}

	output, err := k.client.PutRecords(input)
	if err != nil {
		var ae awserr.Error
		if errors.As(err, &ae) {
			if try < kinesisMaxRetries {
				// Retry with backoff
				return k.putRecords(try+1, records)
			}
		}

		// Not retryable or retries expired
		return err
	}

	// Check errors on individual records
	if output.FailedRecordCount != nil && *output.FailedRecordCount > 0 {
		if try >= kinesisMaxRetries {
			// Retrieve first error message to provide to user.
			// There could be up to kinesisMaxRecordsInBatch
			// errors here and we don't want to flood that.
			var errMsg string
			for _, record := range output.Records {
				if record.ErrorCode != nil && record.ErrorMessage != nil {
					errMsg = *record.ErrorMessage
					break
				}
			}

			return fmt.Errorf(
				"failed to put %d records, retries exhausted. First error: %s",
				*output.FailedRecordCount, errMsg,
			)
		}

		var failedRecords []*kinesis.PutRecordsRequestEntry
		// Collect failed records for retry
		for i, record := range output.Records {
			if record.ErrorCode != nil {
				failedRecords = append(failedRecords, records[i])
			}
		}

		return k.putRecords(try+1, failedRecords)
	}

	return nil
}
