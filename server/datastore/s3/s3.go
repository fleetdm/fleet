package s3

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

const awsRegionHint = "us-east-1"

// Datastore is a type implementing the CarveStore interface
// relying on AWS S3 storage
type Datastore struct {
	metadatadb fleet.CarveStore
	s3client   *s3.S3
	bucket     string
	prefix     string
}

// New initializes an S3 Datastore
func New(config config.S3Config, metadatadb fleet.CarveStore) (*Datastore, error) {
	conf := &aws.Config{}

	// Use default auth provire if no static credentials were provided
	if config.AccessKeyID != "" && config.SecretAccessKey != "" {
		conf.Credentials = credentials.NewStaticCredentials(
			config.AccessKeyID,
			config.SecretAccessKey,
			"",
		)
	}

	sess, err := session.NewSession(conf)
	if err != nil {
		return nil, errors.Wrap(err, "create S3 client")
	}

	// Assume role if configured
	if config.StsAssumeRoleArn != "" {
		stscreds.NewCredentials(sess, config.StsAssumeRoleArn)
		creds := stscreds.NewCredentials(sess, config.StsAssumeRoleArn)
		conf.Credentials = creds
		sess, err = session.NewSession(conf)
		if err != nil {
			return nil, errors.Wrap(err, "create S3 client")
		}
	}

	region, err := s3manager.GetBucketRegion(context.TODO(), sess, config.Bucket, awsRegionHint)
	if err != nil {
		return nil, errors.Wrap(err, "create S3 client")
	}

	return &Datastore{
		metadatadb: metadatadb,
		s3client:   s3.New(sess, &aws.Config{Region: &region}),
		bucket:     config.Bucket,
		prefix:     config.Prefix,
	}, nil
}
