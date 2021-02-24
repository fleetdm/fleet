package logging

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

const (
	// See
	// https://docs.aws.amazon.com/lambda/latest/dg/gettingstarted-limits.html
	// for documentation on limits.
	//
	// (Payload size is lower for async requests)
	lambdaMaxSizeOfPayload = 6 * 1000 * 1000 // 6MB
)

type lambdaLogWriter struct {
	client       lambdaiface.LambdaAPI
	functionName string
	logger       log.Logger
}

func NewLambdaLogWriter(region, id, secret, stsAssumeRoleArn, functionName string, logger log.Logger) (*lambdaLogWriter, error) {
	conf := &aws.Config{
		Region: &region,
	}

	// Only provide static credentials if we have them
	// otherwise use the default credentials provider chain
	if id != "" && secret != "" {
		conf.Credentials = credentials.NewStaticCredentials(id, secret, "")
	}

	sess, err := session.NewSession(conf)
	if err != nil {
		return nil, errors.Wrap(err, "create Lambda client")
	}

	if stsAssumeRoleArn != "" {
		creds := stscreds.NewCredentials(sess, stsAssumeRoleArn)
		conf.Credentials = creds

		sess, err = session.NewSession(conf)

		if err != nil {
			return nil, errors.Wrap(err, "create Lambda client")
		}
	}
	client := lambda.New(sess)

	f := &lambdaLogWriter{
		client:       client,
		functionName: functionName,
		logger:       logger,
	}
	if err := f.validateFunction(); err != nil {
		return nil, errors.Wrap(err, "validate lambda")
	}
	return f, nil
}

func (f *lambdaLogWriter) validateFunction() error {
	out, err := f.client.Invoke(
		&lambda.InvokeInput{
			FunctionName:   &f.functionName,
			InvocationType: aws.String("DryRun"),
		},
	)
	if err != nil {
		return errors.Wrapf(err, "dry run %s", f.functionName)
	}
	if out.FunctionError != nil {
		return errors.Errorf(
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

		out, err := f.client.Invoke(
			&lambda.InvokeInput{
				FunctionName: &f.functionName,
				Payload:      []byte(log),
			},
		)
		if err != nil {
			return errors.Wrapf(err, "run %s", f.functionName)
		}
		if out.FunctionError != nil {
			return errors.Errorf(
				"run %s function error: %s",
				f.functionName,
				*out.FunctionError,
			)
		}
	}

	return nil
}
