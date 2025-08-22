package redis

//go:generate go run gen_aws_region_map.go

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/fleetdm/fleet/v4/server/aws_common"
)

const (
	// emptySHA256 is the SHA256 hash of an empty payload (for GET requests)
	emptySHA256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	elastiCacheServiceName = "elasticache"
)

// awsIAMAuthTokenGenerator generates AWS IAM authentication tokens for ElastiCache
type awsIAMAuthTokenGenerator struct {
	credentials  aws.CredentialsProvider
	signer       *v4.Signer
	region       string
	clusterName  string
	userName     string
	tokenManager *aws_common.IAMAuthTokenManager
}

// newAWSIAMAuthTokenGenerator creates a new AWS IAM authentication token generator
func newAWSIAMAuthTokenGenerator(clusterName, userName, region, assumeRoleArn, stsExternalID string) (*awsIAMAuthTokenGenerator, error) {
	// Load AWS configuration
	cfg, err := aws_common.LoadAWSConfig(context.Background(), region, assumeRoleArn, stsExternalID)
	if err != nil {
		return nil, err
	}

	g := &awsIAMAuthTokenGenerator{
		credentials: cfg.Credentials,
		signer:      v4.NewSigner(),
		region:      region,
		clusterName: clusterName,
		userName:    userName,
	}

	// Create token manager with the generator's token generation function
	g.tokenManager = aws_common.NewIAMAuthTokenManager(g.generateNewToken)

	return g, nil
}

// generateAuthToken generates an IAM authentication token for ElastiCache
// It uses a cache to avoid generating new tokens for every connection
func (g *awsIAMAuthTokenGenerator) generateAuthToken(ctx context.Context) (string, error) {
	return g.tokenManager.GetToken(ctx)
}

// generateNewToken creates a new IAM authentication token
func (g *awsIAMAuthTokenGenerator) generateNewToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://%s/", g.clusterName), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	values := req.URL.Query()
	values.Set("Action", "connect")
	values.Set("User", g.userName)
	req.URL.RawQuery = values.Encode()

	creds, err := g.credentials.Retrieve(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	// Set expiry time (15 minutes)
	query := req.URL.Query()
	query.Set("X-Amz-Expires", "900")
	req.URL.RawQuery = query.Encode()

	presignedURL, _, err := g.signer.PresignHTTP(ctx, creds, req, emptySHA256, elastiCacheServiceName, g.region, time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("failed to presign request: %w", err)
	}

	authToken := strings.TrimPrefix(presignedURL, "https://")

	return authToken, nil
}

// isNumericSuffix checks if a string contains only digits
func isNumericSuffix(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isElastiCacheEndpoint checks if the given endpoint is an ElastiCache endpoint
func isElastiCacheEndpoint(endpoint string) bool {
	// Check for ElastiCache endpoint patterns
	return strings.Contains(endpoint, ".cache.amazonaws.")
}

// parseElastiCacheEndpoint extracts the region and cache name from an ElastiCache endpoint
func parseElastiCacheEndpoint(endpoint string) (region, cacheName string, err error) {
	// Remove port if present
	hostname := endpoint
	if idx := strings.LastIndex(hostname, ":"); idx != -1 {
		hostname = hostname[:idx]
	}

	// Extract region from the ElastiCache endpoint
	// Format: name.region.cache.amazonaws.com or name.serverless.region.cache.amazonaws.com
	parts := strings.Split(hostname, ".")
	if len(parts) < 4 {
		return "", "", fmt.Errorf("invalid ElastiCache endpoint format for IAM auth: %s", endpoint)
	}

	// Find the region code - it's always before "cache.amazonaws.com"
	var regionCode string
	for i := 0; i < len(parts)-2; i++ {
		if parts[i+1] == "cache" && parts[i+2] == "amazonaws" {
			regionCode = parts[i]
			break
		}
	}

	if regionCode == "" {
		return "", "", fmt.Errorf("could not extract region from ElastiCache endpoint: %s", endpoint)
	}

	// Map region code to full region name
	region, ok := awsRegionMap[regionCode]
	if !ok {
		// If not found in map, assume it's already a full region name
		region = regionCode
	}

	// Extract cache name based on endpoint type
	cacheName = extractElastiCacheName(hostname)

	return region, cacheName, nil
}

// extractElastiCacheName extracts the cache name or replication group ID from a hostname
func extractElastiCacheName(hostname string) string {
	if !strings.Contains(hostname, ".cache.amazonaws.") {
		return hostname
	}

	parts := strings.Split(hostname, ".")
	if len(parts) == 0 {
		return hostname
	}

	if strings.Contains(hostname, ".serverless.") {
		// Serverless format: cache-name-xxxxx.serverless.region.cache.amazonaws.com
		// Extract cache name without the random suffix
		namePart := parts[0]
		if idx := strings.LastIndex(namePart, "-"); idx > 0 {
			// Check if what follows the dash looks like a random suffix (6 chars)
			if len(namePart) > idx+6 && len(namePart[idx+1:]) == 6 {
				return namePart[:idx]
			}
		}
		return namePart
	}

	// Standalone format: master.replication-group-id.xxxxx.region.cache.amazonaws.com
	// or: replication-group-id.xxxxx.region.cache.amazonaws.com
	// or: replication-group-id-001.replication-group-id.xxxxx.region.cache.amazonaws.com (cluster mode)
	startIdx := 0
	if parts[0] == "master" && len(parts) > 1 {
		startIdx = 1
	}

	if startIdx < len(parts) {
		cacheName := parts[startIdx]

		// Check if this looks like a cluster node with the pattern:
		// replication-group-id-NNN.replication-group-id.xxx...
		if idx := strings.LastIndex(cacheName, "-"); idx > 0 {
			suffix := cacheName[idx+1:]
			if len(suffix) == 3 && isNumericSuffix(suffix) {
				// This might be a node suffix, check if the next part is the replication group ID
				if startIdx+1 < len(parts) && parts[startIdx+1] == cacheName[:idx] {
					// Yes, it's the cluster node format, return the replication group ID
					return cacheName[:idx]
				}
				// Otherwise just remove the numeric suffix
				return cacheName[:idx]
			}
		}
		return cacheName
	}

	return hostname
}
