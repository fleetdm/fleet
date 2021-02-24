package mock

import (
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/stretchr/testify/mock"
)

type LambdaMock struct {
	mock.Mock
	lambdaiface.LambdaAPI
}

func (l *LambdaMock) Invoke(input *lambda.InvokeInput) (*lambda.InvokeOutput, error) {
	args := l.Called(input)
	out, err := args.Get(0), args.Error(1)
	if out == nil {
		return nil, err
	}
	return out.(*lambda.InvokeOutput), err
}
