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
