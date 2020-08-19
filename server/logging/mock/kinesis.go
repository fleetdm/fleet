package mock

import (
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
)

var _ kinesisiface.KinesisAPI = (*KinesisMock)(nil)

type PutRecordsFunc func(*kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error)
type DescribeStreamFunc func(input *kinesis.DescribeStreamInput) (*kinesis.DescribeStreamOutput, error)
type KinesisMock struct {
	kinesisiface.KinesisAPI

	PutRecordsFunc            PutRecordsFunc
	PutRecordsFuncInvoked     bool
	DescribeStreamFunc        DescribeStreamFunc
	DescribeStreamFuncInvoked bool
}

func (k *KinesisMock) PutRecords(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
	k.PutRecordsFuncInvoked = true
	return k.PutRecordsFunc(input)
}

func (k *KinesisMock) DescribeStream(input *kinesis.DescribeStreamInput) (*kinesis.DescribeStreamOutput, error) {
	k.DescribeStreamFuncInvoked = true
	return k.DescribeStreamFunc(input)
}
