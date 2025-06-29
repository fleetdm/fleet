package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/firehose"
	"github.com/aws/aws-sdk-go-v2/service/firehose/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	firehoseMaxRetries = 8

	// See
	// https://docs.aws.amazon.com/sdk-for-go/api/service/firehose/#Firehose.PutRecordBatch
	// for documentation on limits.
	firehoseMaxRecordsInBatch = 500
	firehoseMaxSizeOfRecord   = 1000 * 1000     // 1,000 KB
	firehoseMaxSizeOfBatch    = 4 * 1000 * 1000 // 4 MB
)

type FirehoseAPI interface {
	PutRecordBatch(ctx context.Context, input *firehose.PutRecordBatchInput, optFns ...func(*firehose.Options)) (*firehose.PutRecordBatchOutput, error)
	DescribeDeliveryStream(ctx context.Context, input *firehose.DescribeDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.DescribeDeliveryStreamOutput, error)
}

type firehoseLogWriter struct {
	client FirehoseAPI
	stream string
	logger log.Logger
}

func NewFirehoseLogWriter(region, endpointURL, id, secret, stsAssumeRoleArn, stsExternalID, stream string, logger log.Logger) (*firehoseLogWriter, error) {
	var opts []func(*aws_config.LoadOptions) error

	// The service endpoint is deprecated, but we still set it
	// in case users are using it.
	if endpointURL != "" {
		opts = append(opts, aws_config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL: endpointURL,
				}, nil
			})),
		)
	}

	// Only provide static credentials if we have them
	// otherwise use the default credentials provider chain.
	if id != "" && secret != "" {
		opts = append(opts,
			aws_config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(id, secret, "")),
		)
	}

	if stsAssumeRoleArn != "" {
		opts = append(opts, aws_config.WithAssumeRoleCredentialOptions(func(r *stscreds.AssumeRoleOptions) {
			r.RoleARN = stsAssumeRoleArn
			if stsExternalID != "" {
				r.ExternalID = &stsExternalID
			}
		}))
	}

	opts = append(opts, aws_config.WithRegion(region))
	conf, err := aws_config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create default config: %w", err)
	}

	firehoseClient := firehose.NewFromConfig(conf)

	f := &firehoseLogWriter{
		client: firehoseClient,
		stream: stream,
		logger: logger,
	}
	if err := f.validateStream(context.Background()); err != nil {
		return nil, fmt.Errorf("validate firehose: %w", err)
	}
	return f, nil
}

func (f *firehoseLogWriter) validateStream(ctx context.Context) error {
	out, err := f.client.DescribeDeliveryStream(ctx,
		&firehose.DescribeDeliveryStreamInput{
			DeliveryStreamName: &f.stream,
		},
	)
	if err != nil {
		return fmt.Errorf("describe stream %s: %w", f.stream, err)
	}

	if out.DeliveryStreamDescription.DeliveryStreamStatus != types.DeliveryStreamStatusActive {
		return fmt.Errorf("delivery stream %s not active", f.stream)
	}

	return nil
}

func (f *firehoseLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	var records []types.Record
	totalBytes := 0
	for _, log := range logs {
		// Add newline because Firehose does not output each record on
		// a separate line.
		log = append(log, '\n')

		// We don't really have a good option for what to do with logs
		// that are too big for Firehose. This behavior is consistent
		// with osquery's behavior in the Firehose logger plugin, and
		// the beginning bytes of the log should help the Fleet admin
		// diagnose the query generating huge results.
		if len(log) > firehoseMaxSizeOfRecord {
			level.Info(f.logger).Log(
				"msg", "dropping log over 1MB Firehose limit",
				"size", len(log),
				"log", string(log[:100])+"...",
			)
			continue
		}

		// If adding this log will exceed the limit on number of
		// records in the batch, or the limit on total size of the
		// records in the batch, we need to push this batch before
		// adding any more.
		if len(records) >= firehoseMaxRecordsInBatch ||
			totalBytes+len(log) > firehoseMaxSizeOfBatch {
			if err := f.putRecordBatch(ctx, 0, records); err != nil {
				return ctxerr.Wrap(ctx, err, "put records")
			}
			totalBytes = 0
			records = nil
		}

		records = append(records, types.Record{
			Data: []byte(log),
		})
		totalBytes += len(log)
	}

	// Push the final batch
	if len(records) > 0 {
		if err := f.putRecordBatch(ctx, 0, records); err != nil {
			return ctxerr.Wrap(ctx, err, "put records")
		}
	}

	return nil
}

func (f *firehoseLogWriter) putRecordBatch(ctx context.Context, try int, records []types.Record) error {
	if try > 0 {
		time.Sleep(100 * time.Millisecond * time.Duration(math.Pow(2.0, float64(try))))
	}
	input := &firehose.PutRecordBatchInput{
		DeliveryStreamName: &f.stream,
		Records:            records,
	}

	output, err := f.client.PutRecordBatch(ctx, input)
	if err != nil {
		var serviceUnavailableException *types.ServiceUnavailableException
		if errors.As(err, &serviceUnavailableException) {
			if try < firehoseMaxRetries {
				// Retry with backoff
				return f.putRecordBatch(ctx, try+1, records)
			}
		}

		// Not retryable or retries expired
		return err
	}

	// Check errors on individual records
	if output.FailedPutCount != nil && *output.FailedPutCount > 0 {
		if try >= firehoseMaxRetries {
			// Retrieve first error message to provide to user.
			// There could be up to firehoseMaxRecordsInBatch
			// errors here and we don't want to flood that.
			var errMsg string
			for _, record := range output.RequestResponses {
				if record.ErrorCode != nil && record.ErrorMessage != nil {
					errMsg = *record.ErrorMessage
					break
				}
			}

			return fmt.Errorf(
				"failed to put %d records, retries exhausted. First error: %s",
				*output.FailedPutCount, errMsg,
			)
		}

		var failedRecords []types.Record
		// Collect failed records for retry
		for i, record := range output.RequestResponses {
			if record.ErrorCode != nil {
				failedRecords = append(failedRecords, records[i])
			}
		}

		return f.putRecordBatch(ctx, try+1, failedRecords)
	}

	return nil
}
