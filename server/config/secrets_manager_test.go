package config

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockSecretsManagerClient is a mock implementation of the SecretsManagerClient interface
type mockSecretsManagerClient struct {
	mock.Mock
}

// GetSecretValue mocks the AWS Secrets Manager GetSecretValue operation
func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*secretsmanager.GetSecretValueOutput), args.Error(1)
}

func TestRetrieveSecretWithRetry_Success(t *testing.T) {
	mockClient := &mockSecretsManagerClient{}
	expectedKey := "test-32-byte-key-for-aes-encryption"
	secretArn := "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret" // #nosec G101 - test data

	mockClient.On("GetSecretValue", mock.Anything, mock.MatchedBy(func(input *secretsmanager.GetSecretValueInput) bool {
		return input.SecretId != nil && *input.SecretId == secretArn
	})).Return(&secretsmanager.GetSecretValueOutput{
		SecretString: &expectedKey,
	}, nil)

	key, err := retrieveSecretWithRetry(context.Background(), mockClient, secretArn)

	require.NoError(t, err)
	assert.Equal(t, expectedKey, key)
	mockClient.AssertExpectations(t)
}

func TestRetrieveSecretWithRetry_BinarySecret(t *testing.T) {
	mockClient := &mockSecretsManagerClient{}
	secretArn := "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret" // #nosec G101 - test data
	binaryData := []byte("binary-data")

	mockClient.On("GetSecretValue", mock.Anything, mock.MatchedBy(func(input *secretsmanager.GetSecretValueInput) bool {
		return input.SecretId != nil && *input.SecretId == secretArn
	})).Return(&secretsmanager.GetSecretValueOutput{
		SecretBinary: binaryData,
	}, nil)

	_, err := retrieveSecretWithRetry(context.Background(), mockClient, secretArn)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains binary data, expected string")
	mockClient.AssertExpectations(t)
}

func TestRetrieveSecretWithRetry_EmptySecret(t *testing.T) {
	mockClient := &mockSecretsManagerClient{}
	secretArn := "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret" // #nosec G101 - test data

	mockClient.On("GetSecretValue", mock.Anything, mock.MatchedBy(func(input *secretsmanager.GetSecretValueInput) bool {
		return input.SecretId != nil && *input.SecretId == secretArn
	})).Return(&secretsmanager.GetSecretValueOutput{
		// Both SecretString and SecretBinary are nil
	}, nil)

	_, err := retrieveSecretWithRetry(context.Background(), mockClient, secretArn)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "contains no data")
	mockClient.AssertExpectations(t)
}

func TestRetrieveSecretWithRetry_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name        string
		mockError   error
		expectedErr string
		shouldRetry bool
	}{
		{
			name:        "ResourceNotFound",
			mockError:   &types.ResourceNotFoundException{},
			expectedErr: "secret not found",
			shouldRetry: false,
		},
		{
			name:        "InvalidRequest",
			mockError:   &types.InvalidRequestException{},
			expectedErr: "access denied",
			shouldRetry: false,
		},
		{
			name:        "InvalidParameter",
			mockError:   &types.InvalidParameterException{},
			expectedErr: "invalid secret ARN",
			shouldRetry: false,
		},
		{
			name:        "NetworkError",
			mockError:   errors.New("network timeout"),
			expectedErr: "failed to retrieve secret after 3 attempts",
			shouldRetry: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &mockSecretsManagerClient{}
			secretArn := "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret" // #nosec G101 - test data, not real credentials

			if tc.shouldRetry {
				// Should be called 3 times for retryable errors
				mockClient.On("GetSecretValue", mock.Anything, mock.Anything).Return(
					(*secretsmanager.GetSecretValueOutput)(nil), tc.mockError).Times(3)
			} else {
				// Should only be called once for non-retryable errors
				mockClient.On("GetSecretValue", mock.Anything, mock.Anything).Return(
					(*secretsmanager.GetSecretValueOutput)(nil), tc.mockError).Once()
			}

			_, err := retrieveSecretWithRetry(context.Background(), mockClient, secretArn)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
			mockClient.AssertExpectations(t)
		})
	}
}

func TestRetrieveSecretWithRetry_ContextCancellation(t *testing.T) {
	mockClient := &mockSecretsManagerClient{}
	secretArn := "arn:aws:secretsmanager:us-west-2:123456789012:secret:test-secret" // #nosec G101 - test data, not real credentials

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Mock the first call to return a retryable error
	mockClient.On("GetSecretValue", mock.Anything, mock.Anything).Return(
		(*secretsmanager.GetSecretValueOutput)(nil), errors.New("network error")).Once()

	_, err := retrieveSecretWithRetry(ctx, mockClient, secretArn)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	mockClient.AssertExpectations(t)
}

func TestRetrieveSecretsManagerSecret_LocalStackDefaultRegion(t *testing.T) {
	awsEndpointURL := os.Getenv("AWS_ENDPOINT_URL")
	if awsEndpointURL == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping LocalStack integration test")
	}

	ctx := context.Background()

	localStackOpts := []func(*aws_config.LoadOptions) error{
		aws_config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	}

	// Configure LocalStack client
	cfg, err := aws_config.LoadDefaultConfig(ctx, localStackOpts...)
	require.NoError(t, err)
	client := secretsmanager.NewFromConfig(cfg)

	secretName := "fleet-test-private-key-localstack"
	privateKey := "test-key-exactly-32-bytes-long!"

	// Clean up any existing secret
	_, _ = client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   &secretName,
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})

	output, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         &secretName,
		SecretString: &privateKey,
		Description:  aws.String("password"),
	})
	require.NoError(t, err, "Failed to create secret in LocalStack")
	secretArn := *output.ARN

	// Clean up after test
	defer func() {
		_, _ = client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			SecretId:                   &secretName,
			ForceDeleteWithoutRecovery: aws.Bool(true),
		})
	}()
	retrievedKey, err := RetrieveSecretsManagerSecretWithOptions(ctx, secretArn, "", "", "", localStackOpts...)
	require.NoError(t, err)
	assert.Equal(t, privateKey, retrievedKey)

	// Test with invalid ARN
	invalidArn := "arn:aws:secretsmanager:us-east-1:000000000000:secret:nonexistent-secret"
	_, err = RetrieveSecretsManagerSecretWithOptions(ctx, invalidArn, "", "", "", localStackOpts...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestRetrieveSecretsManagerSecret_LocalStackDifferentRegion(t *testing.T) {
	awsEndpointURL := os.Getenv("AWS_ENDPOINT_URL")
	if awsEndpointURL == "" {
		t.Skip("AWS_ENDPOINT_URL not set, skipping LocalStack integration test")
	}

	ctx := context.Background()

	localStackOpts := []func(*aws_config.LoadOptions) error{
		aws_config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	}

	// Configure LocalStack client
	cfg, err := aws_config.LoadDefaultConfig(ctx, append(localStackOpts, aws_config.WithRegion("us-east-2"))...)
	require.NoError(t, err)
	client := secretsmanager.NewFromConfig(cfg)

	secretName := "fleet-test-private-key-localstack"
	privateKey := "test-key-exactly-32-bytes-long!"

	// Clean up any existing secret
	_, _ = client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   &secretName,
		ForceDeleteWithoutRecovery: aws.Bool(true),
	})

	output, err := client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         &secretName,
		SecretString: &privateKey,
		Description:  aws.String("password"),
	})
	require.NoError(t, err, "Failed to create secret in LocalStack")
	secretArn := *output.ARN

	// Clean up after test
	defer func() {
		_, _ = client.DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
			SecretId:                   &secretName,
			ForceDeleteWithoutRecovery: aws.Bool(true),
		})
	}()
	retrievedKey, err := RetrieveSecretsManagerSecretWithOptions(ctx, secretArn, "us-east-2", "", "", localStackOpts...)
	require.NoError(t, err)
	assert.Equal(t, privateKey, retrievedKey)

	// Test with invalid ARN
	invalidArn := "arn:aws:secretsmanager:us-east-1:000000000000:secret:nonexistent-secret"
	_, err = RetrieveSecretsManagerSecretWithOptions(ctx, invalidArn, "us-east-2", "", "", localStackOpts...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret not found")
}
