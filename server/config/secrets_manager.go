package config

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/fleetdm/fleet/v4/server/aws_common"
)

// SecretsManagerClient interface for dependency injection and testing
type SecretsManagerClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput,
		optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// retrieveSecretWithRetry retrieves the secret from AWS with retry logic
func retrieveSecretWithRetry(ctx context.Context, client SecretsManagerClient, secretArn string) (string, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter: base 500ms with ±50% randomization
			baseBackoff := time.Duration(500*(1<<uint(attempt-1))) * time.Millisecond    // #nosec G115 - attempt is bounded by maxRetries
			jitter := time.Duration(rand.Float64()*float64(baseBackoff)) - baseBackoff/2 // #nosec G404 - not security sensitive
			backoff := baseBackoff + jitter
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		input := &secretsmanager.GetSecretValueInput{
			SecretId: &secretArn,
		}

		output, err := client.GetSecretValue(ctx, input)
		if err != nil {
			lastErr = err

			// Don't retry certain errors
			var notFoundErr *types.ResourceNotFoundException
			var unauthorizedErr *types.InvalidRequestException
			var invalidParamErr *types.InvalidParameterException

			if errors.As(err, &notFoundErr) {
				return "", fmt.Errorf("secret not found: %s", secretArn)
			}
			if errors.As(err, &unauthorizedErr) {
				return "", fmt.Errorf("access denied to secret: %s", secretArn)
			}
			if errors.As(err, &invalidParamErr) {
				return "", fmt.Errorf("invalid secret ARN: %s", secretArn)
			}

			// Retry for other errors (network issues, throttling, etc.)
			continue
		}

		// Extract secret value
		if output.SecretString != nil {
			return *output.SecretString, nil
		}

		if output.SecretBinary != nil {
			return "", fmt.Errorf("secret %s contains binary data, expected string", secretArn)
		}

		return "", fmt.Errorf("secret %s contains no data", secretArn)
	}

	return "", fmt.Errorf("failed to retrieve secret after %d attempts: %w", maxRetries, lastErr)
}

// RetrieveSecretsManagerSecret retrieves a secret from AWS Secrets Manager
// with support for STS assume role authentication
func RetrieveSecretsManagerSecret(ctx context.Context, secretArn, region, assumeRoleArn, externalID string) (string, error) {
	return RetrieveSecretsManagerSecretWithOptions(ctx, secretArn, region, assumeRoleArn, externalID)
}

// RetrieveSecretsManagerSecretWithOptions retrieves a secret from AWS Secrets Manager
// with custom AWS config options (useful for testing with LocalStack)
func RetrieveSecretsManagerSecretWithOptions(ctx context.Context, secretArn, region, assumeRoleArn, externalID string, opts ...func(*aws_config.LoadOptions) error) (string, error) {
	configOpts := []func(*aws_config.LoadOptions) error{aws_config.WithRegion(region)}
	configOpts = append(configOpts, opts...)
	cfg, err := aws_config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	if assumeRoleArn != "" {
		cfg, err = aws_common.ConfigureAssumeRoleProvider(cfg, nil, assumeRoleArn, externalID)
		if err != nil {
			return "", fmt.Errorf("failed to configure assume role: %w", err)
		}
	}

	client := secretsmanager.NewFromConfig(cfg)

	return retrieveSecretWithRetry(ctx, client, secretArn)
}
