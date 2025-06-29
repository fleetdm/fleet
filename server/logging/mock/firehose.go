package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/firehose"
)

type (
	PutRecordBatchFunc         func(context.Context, *firehose.PutRecordBatchInput, ...func(*firehose.Options)) (*firehose.PutRecordBatchOutput, error)
	DescribeDeliveryStreamFunc func(context.Context, *firehose.DescribeDeliveryStreamInput, ...func(*firehose.Options)) (*firehose.DescribeDeliveryStreamOutput, error)

	FirehoseMock struct {
		PutRecordBatchFunc                PutRecordBatchFunc
		PutRecordBatchFuncInvoked         bool
		DescribeDeliveryStreamFunc        DescribeDeliveryStreamFunc
		DescribeDeliveryStreamFuncInvoked bool
	}
)

func (f *FirehoseMock) PutRecordBatch(ctx context.Context, input *firehose.PutRecordBatchInput, optFns ...func(*firehose.Options)) (*firehose.PutRecordBatchOutput, error) {
	f.PutRecordBatchFuncInvoked = true
	return f.PutRecordBatchFunc(ctx, input, optFns...)
}

func (f *FirehoseMock) DescribeDeliveryStream(ctx context.Context, input *firehose.DescribeDeliveryStreamInput, optFns ...func(*firehose.Options)) (*firehose.DescribeDeliveryStreamOutput, error) {
	f.DescribeDeliveryStreamFuncInvoked = true
	return f.DescribeDeliveryStreamFunc(ctx, input, optFns...)
}
