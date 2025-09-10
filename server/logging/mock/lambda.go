package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/stretchr/testify/mock"
)

type LambdaMock struct {
	mock.Mock
}

func (l *LambdaMock) Invoke(ctx context.Context, input *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
	args := l.Called(input)
	out, err := args.Get(0), args.Error(1)
	if out == nil {
		return nil, err
	}
	return out.(*lambda.InvokeOutput), err
}
