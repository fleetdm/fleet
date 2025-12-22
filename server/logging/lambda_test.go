package logging

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/fleetdm/fleet/v4/server/logging/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	tmock "github.com/stretchr/testify/mock"
)

func makeLambdaWriterWithMock(client LambdaAPI, functionName string) *lambdaLogWriter {
	return &lambdaLogWriter{
		client:       client,
		functionName: functionName,
		logger:       log.NewNopLogger(),
	}
}

func TestLambdaValidateFunctionError(t *testing.T) {
	m := &mock.LambdaMock{}
	ctx := context.Background()
	m.On("Invoke", &lambda.InvokeInput{FunctionName: aws.String("foobar"), InvocationType: types.InvocationTypeDryRun}).
		Return(nil, errors.New("failed"))
	writer := makeLambdaWriterWithMock(m, "foobar")
	err := writer.validateFunction(ctx)
	assert.Error(t, err)
	m.AssertExpectations(test.Quiet(t))
}

func TestLambdaValidateFunctionErrorFunction(t *testing.T) {
	m := &mock.LambdaMock{}
	ctx := context.Background()
	m.On("Invoke", &lambda.InvokeInput{FunctionName: aws.String("foobar"), InvocationType: types.InvocationTypeDryRun}).
		Return(&lambda.InvokeOutput{FunctionError: aws.String("failed")}, nil)
	writer := makeLambdaWriterWithMock(m, "foobar")
	err := writer.validateFunction(ctx)
	assert.Error(t, err)
	m.AssertExpectations(test.Quiet(t))
}

func TestLambdaValidateFunctionSuccess(t *testing.T) {
	m := &mock.LambdaMock{}
	ctx := context.Background()
	m.On("Invoke", &lambda.InvokeInput{FunctionName: aws.String("foobar"), InvocationType: types.InvocationTypeDryRun}).
		Return(&lambda.InvokeOutput{}, nil)
	writer := makeLambdaWriterWithMock(m, "foobar")
	err := writer.validateFunction(ctx)
	assert.NoError(t, err)
	m.AssertExpectations(test.Quiet(t))
}

func TestLambdaError(t *testing.T) {
	m := &mock.LambdaMock{}
	m.On("Invoke", tmock.MatchedBy(
		func(in *lambda.InvokeInput) bool {
			return *in.FunctionName == "foobar" && in.InvocationType == ""
		},
	)).Return(nil, errors.New("failed"))
	writer := makeLambdaWriterWithMock(m, "foobar")
	err := writer.Write(context.Background(), logs)
	assert.Error(t, err)
	m.AssertExpectations(test.Quiet(t))
}

func TestLambdaSuccess(t *testing.T) {
	m := &mock.LambdaMock{}
	m.On("Invoke", tmock.MatchedBy(
		func(in *lambda.InvokeInput) bool {
			return len(in.Payload) > 0 && *in.FunctionName == "foobar" && in.InvocationType == ""
		},
	)).Return(&lambda.InvokeOutput{}, nil).
		Times(len(logs))
	writer := makeLambdaWriterWithMock(m, "foobar")
	err := writer.Write(context.Background(), logs)
	assert.NoError(t, err)
	m.AssertExpectations(test.Quiet(t))
}
