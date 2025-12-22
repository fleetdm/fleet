// Package rdsauth provides AWS IAM authentication for RDS MySQL connections.
package rdsauth

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/fleetdm/fleet/v4/server/aws_common"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/go-kit/log"
	"github.com/go-sql-driver/mysql"
	// Blank import registers the "rdsmysql" TLS config with pre-loaded AWS RDS CA certificates
	_ "github.com/shogo82148/rdsmysql/v2"
)

// iamAuthTokenGenerator generates AWS IAM authentication tokens for RDS MySQL
type iamAuthTokenGenerator struct {
	dbEndpoint   string // Full endpoint with port
	dbUsername   string
	region       string
	credentials  aws.CredentialsProvider
	tokenManager *aws_common.IAMAuthTokenManager
}

// newIAMAuthTokenGenerator creates a new IAM authentication token generator
func newIAMAuthTokenGenerator(dbEndpoint, dbUsername, dbPort, region, assumeRoleArn, stsExternalID string) (*iamAuthTokenGenerator, error) {
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

	g := &iamAuthTokenGenerator{
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
func (g *iamAuthTokenGenerator) getAuthToken(ctx context.Context) (string, error) {
	return g.tokenManager.GetToken(ctx)
}

// newToken creates a new IAM authentication token
func (g *iamAuthTokenGenerator) newToken(ctx context.Context) (string, error) {
	authToken, err := auth.BuildAuthToken(ctx, g.dbEndpoint, g.region, g.dbUsername, g.credentials)
	if err != nil {
		return "", fmt.Errorf("failed to build auth token: %w", err)
	}

	return authToken, nil
}

// Connector implements driver.Connector for IAM authentication
type Connector struct {
	baseDSN  string
	tokenGen *iamAuthTokenGenerator
	logger   log.Logger
}

// Connect implements driver.Connector
func (c *Connector) Connect(ctx context.Context) (driver.Conn, error) {
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
func (c *Connector) Driver() driver.Driver {
	return mysql.MySQLDriver{}
}

// NewConnectorFactory returns a factory function that creates IAM-authenticated
// database connectors. This factory can be injected into common_mysql.NewDB
// to enable IAM authentication without adding AWS dependencies to common_mysql.
func NewConnectorFactory(conf *config.MysqlConfig, host, port string) (func(dsn string, logger log.Logger) (driver.Connector, error), error) {
	tokenGen, err := newIAMAuthTokenGenerator(
		host,
		conf.Username,
		port,
		conf.Region,
		conf.StsAssumeRoleArn,
		conf.StsExternalID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM token generator: %w", err)
	}

	return func(dsn string, logger log.Logger) (driver.Connector, error) {
		return &Connector{
			baseDSN:  dsn,
			tokenGen: tokenGen,
			logger:   logger,
		}, nil
	}, nil
}
