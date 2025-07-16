package s3

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/aws_common"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const awsRegionHint = "us-east-1"

type s3store struct {
	s3Client         *s3.Client
	bucket           string
	prefix           string
	cloudFrontConfig *config.S3CloudFrontConfig
}

type installerNotFoundError struct{}

var _ fleet.NotFoundError = (*installerNotFoundError)(nil)

func (p installerNotFoundError) Error() string {
	return "installer not found"
}

func (p installerNotFoundError) IsNotFound() bool {
	return true
}

// newS3Store initializes an S3 Datastore.
func newS3Store(cfg config.S3ConfigInternal) (*s3store, error) {
	var opts []func(*aws_config.LoadOptions) error

	// The service endpoint is deprecated, but we still set it
	// in case users are using it.
	// It is also used when testing with minio.
	if cfg.EndpointURL != "" {
		opts = append(opts, aws_config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL: cfg.EndpointURL,
				}, nil
			})),
		)
	}

	// DisableSSL is only used for testing.
	if cfg.DisableSSL {
		// Ignoring "G402: TLS InsecureSkipVerify set true", this is only used for automated testing.
		c := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{ //nolint:gosec
			InsecureSkipVerify: false,
		}))
		opts = append(opts, aws_config.WithHTTPClient(c))
	}

	// Use default auth provider if no static credentials were provided.
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, aws_config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)))
	}

	if cfg.Region == "" {
		// Attempt to deduce region from bucket.
		conf, err := aws_config.LoadDefaultConfig(context.Background(),
			append(opts, aws_config.WithRegion(awsRegionHint))...,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create default config to get bucket region: %w", err)
		}
		bucketRegion, err := manager.GetBucketRegion(context.Background(), s3.NewFromConfig(conf), cfg.Bucket)
		if err != nil {
			return nil, fmt.Errorf("get bucket region: %w", err)
		}
		cfg.Region = bucketRegion
	}

	opts = append(opts, aws_config.WithRegion(cfg.Region))
	conf, err := aws_config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create default config: %w", err)
	}

	if cfg.StsAssumeRoleArn != "" {
		conf, err = aws_common.ConfigureAssumeRoleProvider(conf, opts, cfg.StsAssumeRoleArn, cfg.StsExternalID)
		if err != nil {
			return nil, fmt.Errorf("failed to configure assume role provider: %w", err)
		}
	}

	s3Client := s3.NewFromConfig(conf, func(o *s3.Options) {
		o.UsePathStyle = cfg.ForceS3PathStyle
	})

	return &s3store{
		s3Client:         s3Client,
		bucket:           cfg.Bucket,
		prefix:           cfg.Prefix,
		cloudFrontConfig: cfg.CloudFrontConfig,
	}, nil
}

// CreateTestBucket creates a bucket with the provided name and a default
// bucket config. Only recommended for local testing.
func (s *s3store) CreateTestBucket(ctx context.Context, name string) error {
	_, err := s.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket:                    &name,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{},
	})

	// Don't error if the bucket already exists
	var (
		bucketAlreadyExists     *types.BucketAlreadyExists
		bucketAlreadyOwnedByYou *types.BucketAlreadyOwnedByYou
	)
	if errors.As(err, &bucketAlreadyExists) || errors.As(err, &bucketAlreadyOwnedByYou) {
		return nil
	}
	return err
}
