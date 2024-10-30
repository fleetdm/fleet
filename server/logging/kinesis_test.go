package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	"github.com/fleetdm/fleet/v4/server/logging/mock"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
)

func makeKinesisWriterWithMock(client kinesisiface.KinesisAPI, stream string) *kinesisLogWriter {
	return &kinesisLogWriter{
		client: client,
		stream: stream,
		logger: log.NewNopLogger(),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func getLogsFromPutRecordsInput(input *kinesis.PutRecordsInput) []json.RawMessage {
	var logs []json.RawMessage
	for _, record := range input.Records {
		// remove the newline appended to get back the original raw byte input
		logs = append(logs, bytes.Trim(record.Data, "\n"))
	}
	return logs
}

func TestKinesisRetryableFailure(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	putFunc := func(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
		callCount += 1
		assert.Equal(t, logs, getLogsFromPutRecordsInput(input))
		assert.Equal(t, "foobar", *input.StreamName)
		if callCount < 3 {
			return nil, awserr.New(kinesis.ErrCodeProvisionedThroughputExceededException, "", nil)
		}
		// Returning a non-retryable error earlier helps keep this test faster
		return nil, errors.New("generic error")
	}
	k := &mock.KinesisMock{PutRecordsFunc: putFunc}
	writer := makeKinesisWriterWithMock(k, "foobar")
	err := writer.Write(ctx, logs)
	assert.Error(t, err)
	assert.Equal(t, 3, callCount)
}

func TestKinesisNormalPut(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	putFunc := func(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
		callCount += 1
		assert.Equal(t, logs, getLogsFromPutRecordsInput(input))
		assert.Equal(t, "foobar", *input.StreamName)
		return &kinesis.PutRecordsOutput{FailedRecordCount: aws.Int64(0)}, nil
	}
	k := &mock.KinesisMock{PutRecordsFunc: putFunc}
	writer := makeKinesisWriterWithMock(k, "foobar")
	err := writer.Write(ctx, logs)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestKinesisSomeFailures(t *testing.T) {
	ctx := context.Background()
	k := &mock.KinesisMock{}
	callCount := 0

	call3 := func(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
		// final invocation
		callCount += 1
		assert.Equal(t, logs[1:2], getLogsFromPutRecordsInput(input))
		return &kinesis.PutRecordsOutput{
			FailedRecordCount: aws.Int64(0),
		}, nil
	}

	call2 := func(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
		// Set to invoke call3 next time
		k.PutRecordsFunc = call3
		callCount += 1
		assert.Equal(t, logs[1:], getLogsFromPutRecordsInput(input))
		return &kinesis.PutRecordsOutput{
			FailedRecordCount: aws.Int64(1),
			Records: []*kinesis.PutRecordsResultEntry{
				&kinesis.PutRecordsResultEntry{
					ErrorCode: aws.String("error"),
				},
				&kinesis.PutRecordsResultEntry{
					SequenceNumber: aws.String("foo"),
				},
			},
		}, nil
	}

	call1 := func(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
		// Use call2 function for next call
		k.PutRecordsFunc = call2
		callCount += 1
		assert.Equal(t, logs, getLogsFromPutRecordsInput(input))
		return &kinesis.PutRecordsOutput{
			FailedRecordCount: aws.Int64(1),
			Records: []*kinesis.PutRecordsResultEntry{
				&kinesis.PutRecordsResultEntry{
					SequenceNumber: aws.String("foo"),
				},
				&kinesis.PutRecordsResultEntry{
					ErrorCode: aws.String("error"),
				},
				&kinesis.PutRecordsResultEntry{
					ErrorCode: aws.String("error"),
				},
			},
		}, nil
	}
	k.PutRecordsFunc = call1
	writer := makeKinesisWriterWithMock(k, "foobar")
	err := writer.Write(ctx, logs)
	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestKinesisFailAllRecords(t *testing.T) {
	ctx := context.Background()
	k := &mock.KinesisMock{}
	callCount := 0

	k.PutRecordsFunc = func(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
		callCount += 1
		assert.Equal(t, logs, getLogsFromPutRecordsInput(input))
		if callCount < 3 {
			return &kinesis.PutRecordsOutput{
				FailedRecordCount: aws.Int64(1),
				Records: []*kinesis.PutRecordsResultEntry{
					{ErrorCode: aws.String("error")},
					{ErrorCode: aws.String("error")},
					{ErrorCode: aws.String("error")},
				},
			}, nil
		}
		// Make test quicker by returning non-retryable error
		// before all retries are exhausted.
		return nil, errors.New("generic error")
	}

	writer := makeKinesisWriterWithMock(k, "foobar")
	err := writer.Write(ctx, logs)
	assert.Error(t, err)
	assert.Equal(t, 3, callCount)
}

func TestKinesisRecordTooBig(t *testing.T) {
	ctx := context.Background()
	newLogs := make([]json.RawMessage, len(logs))
	copy(newLogs, logs)
	newLogs[0] = make(json.RawMessage, kinesisMaxSizeOfRecord+1)
	callCount := 0
	putFunc := func(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
		callCount += 1
		assert.Equal(t, newLogs[1:], getLogsFromPutRecordsInput(input))
		assert.Equal(t, "foobar", *input.StreamName)
		return &kinesis.PutRecordsOutput{FailedRecordCount: aws.Int64(0)}, nil
	}
	k := &mock.KinesisMock{PutRecordsFunc: putFunc}
	writer := makeKinesisWriterWithMock(k, "foobar")
	err := writer.Write(ctx, newLogs)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestKinesisSplitBatchBySize(t *testing.T) {
	ctx := context.Background()
	// Make each record just under 1 MB (accounting for partitionkey) so that it
	// takes 3 total batches of just under 5 MB each
	logs := make([]json.RawMessage, 15)
	for i := 0; i < len(logs); i++ {
		logs[i] = make(json.RawMessage, kinesisMaxSizeOfRecord-1-256)
	}
	callCount := 0
	putFunc := func(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
		callCount += 1
		assert.Len(t, getLogsFromPutRecordsInput(input), 5)
		assert.Equal(t, "foobar", *input.StreamName)
		return &kinesis.PutRecordsOutput{FailedRecordCount: aws.Int64(0)}, nil
	}
	k := &mock.KinesisMock{PutRecordsFunc: putFunc}
	writer := makeKinesisWriterWithMock(k, "foobar")
	err := writer.Write(ctx, logs)
	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestKinesisSplitBatchByCount(t *testing.T) {
	ctx := context.Background()
	logs := make([]json.RawMessage, 2000)
	for i := 0; i < len(logs); i++ {
		logs[i] = json.RawMessage(`{}`)
	}
	callCount := 0
	putFunc := func(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
		callCount += 1
		assert.Len(t, getLogsFromPutRecordsInput(input), kinesisMaxRecordsInBatch)
		assert.Equal(t, "foobar", *input.StreamName)
		return &kinesis.PutRecordsOutput{FailedRecordCount: aws.Int64(0)}, nil
	}
	k := &mock.KinesisMock{PutRecordsFunc: putFunc}
	writer := makeKinesisWriterWithMock(k, "foobar")
	err := writer.Write(ctx, logs)
	assert.NoError(t, err)
	assert.Equal(t, 4, callCount)
}

func TestKinesisValidateStreamActive(t *testing.T) {
	describeFunc := func(input *kinesis.DescribeStreamInput) (*kinesis.DescribeStreamOutput, error) {
		assert.Equal(t, "test", *input.StreamName)
		return &kinesis.DescribeStreamOutput{
			StreamDescription: &kinesis.StreamDescription{
				StreamStatus: aws.String(kinesis.StreamStatusActive),
			},
		}, nil
	}
	k := &mock.KinesisMock{DescribeStreamFunc: describeFunc}
	writer := makeKinesisWriterWithMock(k, "test")
	err := writer.validateStream()
	assert.NoError(t, err)
	assert.True(t, k.DescribeStreamFuncInvoked)
}

func TestKinesisValidateStreamNotActive(t *testing.T) {
	describeFunc := func(input *kinesis.DescribeStreamInput) (*kinesis.DescribeStreamOutput, error) {
		assert.Equal(t, "test", *input.StreamName)
		return &kinesis.DescribeStreamOutput{
			StreamDescription: &kinesis.StreamDescription{
				StreamStatus: aws.String(kinesis.StreamStatusCreating),
			},
		}, nil
	}
	k := &mock.KinesisMock{DescribeStreamFunc: describeFunc}
	writer := makeKinesisWriterWithMock(k, "test")
	err := writer.validateStream()
	assert.Error(t, err)
	assert.True(t, k.DescribeStreamFuncInvoked)
}

func TestKinesisValidateStreamError(t *testing.T) {
	describeFunc := func(input *kinesis.DescribeStreamInput) (*kinesis.DescribeStreamOutput, error) {
		assert.Equal(t, "test", *input.StreamName)
		return nil, errors.New("kaboom!")
	}
	k := &mock.KinesisMock{DescribeStreamFunc: describeFunc}
	writer := makeKinesisWriterWithMock(k, "test")
	err := writer.validateStream()
	assert.Error(t, err)
	assert.True(t, k.DescribeStreamFuncInvoked)
}
