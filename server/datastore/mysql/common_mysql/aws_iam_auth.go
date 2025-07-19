package common_mysql

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/fleetdm/fleet/v4/server/aws_common"
	"github.com/go-kit/log"
	"github.com/go-sql-driver/mysql"
)

// awsIAMAuthTokenGenerator generates AWS IAM authentication tokens for RDS MySQL
type awsIAMAuthTokenGenerator struct {
	dbEndpoint   string // Full endpoint with port
	dbUsername   string
	region       string
	credentials  aws.CredentialsProvider
	tokenManager *aws_common.IAMAuthTokenManager
}

// newAWSIAMAuthTokenGenerator creates a new IAM authentication token generator
func newAWSIAMAuthTokenGenerator(dbEndpoint, dbUsername, dbPort, region, assumeRoleArn, stsExternalID string) (*awsIAMAuthTokenGenerator, error) {
	// Load AWS configuration
	cfg, err := aws_common.LoadAWSConfig(context.Background(), region, assumeRoleArn, stsExternalID)
	if err != nil {
		return nil, err
	}

	// Format endpoint with port if not already included
	fullEndpoint := dbEndpoint
	if !strings.Contains(dbEndpoint, ":") {
		fullEndpoint = fmt.Sprintf("%s:%s", dbEndpoint, dbPort)
	}

	g := &awsIAMAuthTokenGenerator{
		dbEndpoint:  fullEndpoint,
		dbUsername:  dbUsername,
		region:      region,
		credentials: cfg.Credentials,
	}

	// Create token manager with the generator's token generation function
	g.tokenManager = aws_common.NewIAMAuthTokenManager(g.newToken)

	return g, nil
}

// getAuthToken gets an IAM authentication token for RDS
// It uses a cache to avoid generating new tokens for every connection
func (g *awsIAMAuthTokenGenerator) getAuthToken(ctx context.Context) (string, error) {
	return g.tokenManager.GetToken(ctx)
}

// newToken creates a new IAM authentication token
func (g *awsIAMAuthTokenGenerator) newToken(ctx context.Context) (string, error) {
	authToken, err := auth.BuildAuthToken(ctx, g.dbEndpoint, g.region, g.dbUsername, g.credentials)
	if err != nil {
		return "", fmt.Errorf("failed to build auth token: %w", err)
	}

	return authToken, nil
}

// isRDSEndpoint checks if the given endpoint is an RDS endpoint
func isRDSEndpoint(endpoint string) bool {
	return strings.Contains(endpoint, ".rds.amazonaws.")
}

// extractRDSRegion extracts the AWS region from an RDS endpoint
func extractRDSRegion(endpoint string) (string, error) {
	// RDS endpoint formats:
	// - instance-name.abcdefg.region.rds.amazonaws.com (regular RDS)
	// - cluster-name.cluster-abcdefg.region.rds.amazonaws.com (Aurora cluster endpoint)
	// - cluster-name.cluster-ro-abcdefg.region.rds.amazonaws.com (Aurora read-only endpoint)
	// - proxy-name.proxy-abcdefg.region.rds.amazonaws.com (RDS Proxy endpoint)
	// - instance-name.abcdefg.region.rds.amazonaws.com.cn (China regions)

	parts := strings.Split(endpoint, ".")
	if len(parts) < 5 {
		return "", fmt.Errorf("invalid RDS endpoint format for IAM auth: %s", endpoint)
	}

	// Check if it's an RDS endpoint
	if !strings.Contains(endpoint, ".rds.amazonaws.com") {
		return "", fmt.Errorf("endpoint does not appear to be an RDS endpoint: %s", endpoint)
	}

	// Find the region - it's always before "rds.amazonaws.com"
	for i := 0; i < len(parts)-3; i++ {
		if parts[i+1] == "rds" && parts[i+2] == "amazonaws" && (parts[i+3] == "com" || parts[i+3] == "com.cn") {
			return parts[i], nil
		}
	}

	return "", fmt.Errorf("could not extract region from RDS endpoint: %s", endpoint)
}

// awsIAMAuthConnector implements driver.Connector for IAM authentication
type awsIAMAuthConnector struct {
	driverName string
	baseDSN    string
	tokenGen   *awsIAMAuthTokenGenerator
	logger     log.Logger
}

// Connect implements driver.Connector
func (c *awsIAMAuthConnector) Connect(ctx context.Context) (driver.Conn, error) {
	token, err := c.tokenGen.getAuthToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate IAM auth token: %w", err)
	}

	cfg, err := mysql.ParseDSN(c.baseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	cfg.Passwd = token

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connector: %w", err)
	}

	return connector.Connect(ctx)
}

// Driver implements driver.Connector
func (c *awsIAMAuthConnector) Driver() driver.Driver {
	return mysql.MySQLDriver{}
}

// isRDSProxyEndpoint checks if the given address is an RDS Proxy endpoint
func isRDSProxyEndpoint(address string) bool {
	// RDS Proxy endpoints have the format: proxy-name.proxy-xxxxxxxxx.region.rds.amazonaws.com
	return strings.Contains(address, ".proxy-") && strings.Contains(address, ".rds.amazonaws.")
}
