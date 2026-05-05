// Package aws_common contains common functionality used
// by packages that use AWS features (kinesis, firehose, ses, lambda, s3)
package aws_common

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// ConfigureAssumeRoleProvider configures the credential provider with a "Assume Role"
// provider and returns a new aws.Config.
//
// It overrides any aws_config.WithCredentialsProvider set in opts.
func ConfigureAssumeRoleProvider(
	conf aws.Config,
	opts []func(*aws_config.LoadOptions) error,
	stsAssumeRoleARN,
	stsExternalID string,
) (aws.Config, error) {
	stsClient := sts.NewFromConfig(conf)
	credsProvider := stscreds.NewAssumeRoleProvider(stsClient, stsAssumeRoleARN, func(r *stscreds.AssumeRoleOptions) {
		if stsExternalID != "" {
			r.ExternalID = &stsExternalID
		}
	})
	// Overrides any previous aws_config.WithCredentialsProvider set in opts.
	opts = append(opts,
		aws_config.WithCredentialsProvider(credsProvider),
	)
	conf, err := aws_config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to create default config with sts assume role: %w", err)
	}
	return conf, nil
}
