package logging

import (
	"context"
	"encoding/json"
	"fmt"

	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	// See
	// https://docs.aws.amazon.com/lambda/latest/dg/gettingstarted-limits.html
	// for documentation on limits.
	//
	// (Payload size is lower for async requests)
	lambdaMaxSizeOfPayload = 6 * 1000 * 1000 // 6MB
)

type LambdaAPI interface {
	Invoke(ctx context.Context, params *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error)
}

type lambdaLogWriter struct {
	client       LambdaAPI
	functionName string
	logger       log.Logger
}

func NewLambdaLogWriter(region, id, secret, stsAssumeRoleArn, stsExternalID, functionName string, logger log.Logger) (*lambdaLogWriter, error) {
	var opts []func(*aws_config.LoadOptions) error

	// Only provide static credentials if we have them
	// otherwise use the default credentials provider chain.
	if id != "" && secret != "" {
		opts = append(opts,
			aws_config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(id, secret, "")),
		)
	}

	// cfg.StsAssumeRoleArn has been marked as deprecated, but we still set it in case users are using it.
	if stsAssumeRoleArn != "" {
		opts = append(opts, aws_config.WithAssumeRoleCredentialOptions(func(r *stscreds.AssumeRoleOptions) {
			r.RoleARN = stsAssumeRoleArn
			if stsExternalID != "" {
				r.ExternalID = &stsExternalID
			}
		}))
	}

	opts = append(opts, aws_config.WithRegion(region))
	conf, err := aws_config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create default config: %w", err)
	}
	lambdaClient := lambda.NewFromConfig(conf)

	f := &lambdaLogWriter{
		client:       lambdaClient,
		functionName: functionName,
		logger:       logger,
	}
	if err := f.validateFunction(context.Background()); err != nil {
		return nil, fmt.Errorf("validate lambda: %w", err)
	}
	return f, nil
}

func (f *lambdaLogWriter) validateFunction(ctx context.Context) error {
	out, err := f.client.Invoke(ctx,
		&lambda.InvokeInput{
			FunctionName:   &f.functionName,
			InvocationType: types.InvocationTypeDryRun,
		},
	)
	if err != nil {
		return fmt.Errorf("dry run %s: %w", f.functionName, err)
	}
	if out.FunctionError != nil {
		return fmt.Errorf(
			"dry run %s function error: %s",
			f.functionName,
			*out.FunctionError,
		)
	}

	return nil
}

func (f *lambdaLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	for _, log := range logs {
		// We don't really have a good option for what to do with logs
		// that are too big for Lambda. This behavior is consistent
		// with other logging plugins.
		if len(log) > lambdaMaxSizeOfPayload {
			level.Info(f.logger).Log(
				"msg", "dropping log over 6MB Lambda limit",
				"size", len(log),
				"log", string(log[:100])+"...",
			)
			continue
		}

		out, err := f.client.Invoke(ctx,
			&lambda.InvokeInput{
				FunctionName: &f.functionName,
				Payload:      []byte(log),
			},
		)
		if err != nil {
			return fmt.Errorf("run %s: %w", f.functionName, err)
		}
		if out.FunctionError != nil {
			return fmt.Errorf(
				"run %s function error: %s",
				f.functionName,
				*out.FunctionError,
			)
		}
	}

	return nil
}
