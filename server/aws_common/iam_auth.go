package aws_common

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	// Token validity is 15 minutes, but we refresh after 10 minutes plus jitter
	tokenRefreshTime = 10 * time.Minute
	maxJitter        = 30 * time.Second
)

// IAMTokenCache holds a cached token and its generation time
type IAMTokenCache struct {
	token     string
	generated time.Time
}

// TokenGenerator is a function that generates a new IAM authentication token
type TokenGenerator func(ctx context.Context) (string, error)

// IAMAuthTokenManager manages AWS IAM authentication tokens with caching
type IAMAuthTokenManager struct {
	// Token generator function specific to the service (RDS, ElastiCache, etc.)
	generateToken TokenGenerator

	// Token cache with RW mutex
	cacheMu sync.RWMutex
	cache   *IAMTokenCache
}

// NewIAMAuthTokenManager creates a new IAM authentication token manager
func NewIAMAuthTokenManager(tokenGen TokenGenerator) *IAMAuthTokenManager {
	return &IAMAuthTokenManager{
		generateToken: tokenGen,
	}
}

// GetToken retrieves a valid IAM authentication token, using cache when possible
func (m *IAMAuthTokenManager) GetToken(ctx context.Context) (string, error) {
	// Calculate expiry time with jitter
	jitter := time.Duration(rand.Int63n(int64(maxJitter))) //nolint:gosec // jitter doesn't need cryptographic randomness
	expiryTime := tokenRefreshTime + jitter

	// Check if we have a valid cached token
	m.cacheMu.RLock()
	if m.cache != nil && time.Since(m.cache.generated) < expiryTime {
		token := m.cache.token
		m.cacheMu.RUnlock()
		return token, nil
	}
	m.cacheMu.RUnlock()

	// Need to generate a new token
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	// Double-check in case another goroutine generated a token while we were waiting
	if m.cache != nil && time.Since(m.cache.generated) < expiryTime {
		return m.cache.token, nil
	}

	token, err := m.generateToken(ctx)
	if err != nil {
		return "", err
	}

	m.cache = &IAMTokenCache{
		token:     token,
		generated: time.Now(),
	}

	return token, nil
}

// LoadAWSConfig loads AWS configuration with optional assume role support
func LoadAWSConfig(ctx context.Context, region, assumeRoleArn, stsExternalID string) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{config.WithRegion(region)}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// If assume role ARN is provided, configure it
	if assumeRoleArn != "" {
		cfg, err = ConfigureAssumeRoleProvider(cfg, opts, assumeRoleArn, stsExternalID)
		if err != nil {
			return aws.Config{}, fmt.Errorf("failed to configure assume role provider: %w", err)
		}
	}

	return cfg, nil
}
