package mock

import (
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
)

var _ firehoseiface.FirehoseAPI = (*FirehoseMock)(nil)

type PutRecordBatchFunc func(*firehose.PutRecordBatchInput) (*firehose.PutRecordBatchOutput, error)
type DescribeDeliveryStreamFunc func(input *firehose.DescribeDeliveryStreamInput) (*firehose.DescribeDeliveryStreamOutput, error)
type FirehoseMock struct {
	firehoseiface.FirehoseAPI

	PutRecordBatchFunc                PutRecordBatchFunc
	PutRecordBatchFuncInvoked         bool
	DescribeDeliveryStreamFunc        DescribeDeliveryStreamFunc
	DescribeDeliveryStreamFuncInvoked bool
}

func (f *FirehoseMock) PutRecordBatch(input *firehose.PutRecordBatchInput) (*firehose.PutRecordBatchOutput, error) {
	f.PutRecordBatchFuncInvoked = true
	return f.PutRecordBatchFunc(input)
}

func (f *FirehoseMock) DescribeDeliveryStream(input *firehose.DescribeDeliveryStreamInput) (*firehose.DescribeDeliveryStreamOutput, error) {
	f.DescribeDeliveryStreamFuncInvoked = true
	return f.DescribeDeliveryStreamFunc(input)
}
