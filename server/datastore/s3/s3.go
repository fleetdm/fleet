package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/fleetdm/fleet/v4/server/config"
)

const awsRegionHint = "us-east-1"

type s3store struct {
	s3client         *s3.S3
	bucket           string
	prefix           string
	cloudFrontConfig *config.S3CloudFrontConfig
}

// newS3store initializes an S3 Datastore
func newS3store(config config.S3ConfigInternal) (*s3store, error) {
	conf := &aws.Config{}

	// Use default auth provire if no static credentials were provided
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		conf.Credentials = credentials.NewStaticCredentials(
			config.AccessKeyID,
			config.SecretAccessKey,
			"",
		)
	}

	if config.EndpointURL != "" {
		conf.Endpoint = &config.EndpointURL
	}

	conf.DisableSSL = &config.DisableSSL
	conf.S3ForcePathStyle = &config.ForceS3PathStyle

	sess, err := session.NewSession(conf)
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}

	// Assume role if configured
	if config.StsAssumeRoleArn != "" {
		creds := stscreds.NewCredentials(sess, config.StsAssumeRoleArn, func(provider *stscreds.AssumeRoleProvider) {
			if config.StsExternalID != "" {
				provider.ExternalID = &config.StsExternalID
			}
		})
		conf.Credentials = creds
		sess, err = session.NewSession(conf)
		if err != nil {
			return nil, fmt.Errorf("create S3 client: %w", err)
		}
	}

	if len(config.Region) == 0 {
		region, err := s3manager.GetBucketRegion(context.TODO(), sess, config.Bucket, awsRegionHint)
		if err != nil {
			return nil, fmt.Errorf("create S3 client: %w", err)
		}
		config.Region = region
	}

	return &s3store{
		s3client:         s3.New(sess, &aws.Config{Region: &config.Region}),
		bucket:           config.Bucket,
		prefix:           config.Prefix,
		cloudFrontConfig: config.CloudFrontConfig,
	}, nil
}

// CreateTestBucket creates a bucket with the provided name and a default
// bucket config. Only recommended for local testing.
func (s *s3store) CreateTestBucket(name string) error {
	_, err := s.s3client.CreateBucket(&s3.CreateBucketInput{
		Bucket:                    &name,
		CreateBucketConfiguration: &s3.CreateBucketConfiguration{},
	})

	// Don't error if the bucket already exists
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeBucketAlreadyExists, s3.ErrCodeBucketAlreadyOwnedByYou:
			return nil
		}
	}

	return err
}
