package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kinesis"
)

type (
	PutRecordsFunc     func(context.Context, *kinesis.PutRecordsInput, ...func(*kinesis.Options)) (*kinesis.PutRecordsOutput, error)
	DescribeStreamFunc func(context.Context, *kinesis.DescribeStreamInput, ...func(*kinesis.Options)) (*kinesis.DescribeStreamOutput, error)
	KinesisMock        struct {
		PutRecordsFunc            PutRecordsFunc
		PutRecordsFuncInvoked     bool
		DescribeStreamFunc        DescribeStreamFunc
		DescribeStreamFuncInvoked bool
	}
)

func (k *KinesisMock) PutRecords(ctx context.Context, input *kinesis.PutRecordsInput, optFns ...func(*kinesis.Options)) (*kinesis.PutRecordsOutput, error) {
	k.PutRecordsFuncInvoked = true
	return k.PutRecordsFunc(ctx, input, optFns...)
}

func (k *KinesisMock) DescribeStream(ctx context.Context, input *kinesis.DescribeStreamInput, optFns ...func(*kinesis.Options)) (*kinesis.DescribeStreamOutput, error) {
	k.DescribeStreamFuncInvoked = true
	return k.DescribeStreamFunc(ctx, input, optFns...)
}
