package mail

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type fleetSESSender interface {
	SendRawEmail(ctx context.Context, input *ses.SendRawEmailInput, optFns ...func(*ses.Options)) (*ses.SendRawEmailOutput, error)
}

type sesSender struct {
	client    fleetSESSender
	sourceArn string
}

func getFromSES(e fleet.Email) (string, error) {
	serverURL, err := url.Parse(e.ServerURL)
	if err != nil || len(serverURL.Host) == 0 {
		return "", fmt.Errorf("failed to parse server url %s err: %w", e.ServerURL, err)
	}
	return fmt.Sprintf("From: %s\r\n", fmt.Sprintf("do-not-reply@%s", serverURL.Host)), nil
}

func (s *sesSender) SendEmail(ctx context.Context, e fleet.Email) error {
	if s.client == nil {
		return errors.New("ses sender not configured")
	}
	msg, err := getMessageBody(e, getFromSES)
	if err != nil {
		return err
	}
	return s.sendMail(ctx, e, msg)
}

func (s *sesSender) CanSendEmail(smtpSettings fleet.SMTPSettings) bool {
	return s.client != nil
}

func NewSESSender(region, endpointURL, id, secret, stsAssumeRoleArn, stsExternalID, sourceArn string) (*sesSender, error) {
	var opts []func(*aws_config.LoadOptions) error

	// The service endpoint is deprecated, but we still set it
	// in case users are using it.
	if endpointURL != "" {
		opts = append(opts, aws_config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL: endpointURL,
				}, nil
			})),
		)
	}

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

	sesClient := ses.NewFromConfig(conf)

	return &sesSender{
		client:    sesClient,
		sourceArn: sourceArn,
	}, nil
}

func (s *sesSender) sendMail(ctx context.Context, e fleet.Email, msg []byte) error {
	_, err := s.client.SendRawEmail(ctx, &ses.SendRawEmailInput{
		Destinations: e.To,
		FromArn:      &s.sourceArn,
		RawMessage:   &types.RawMessage{Data: msg},
		SourceArn:    &s.sourceArn,
	})
	if err != nil {
		return err
	}
	return nil
}
