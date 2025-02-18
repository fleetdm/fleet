package mail

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type fleetSESSender interface {
	SendRawEmail(input *ses.SendRawEmailInput) (*ses.SendRawEmailOutput, error)
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

func (s *sesSender) SendEmail(e fleet.Email) error {
	if s.client == nil {
		return errors.New("ses sender not configured")
	}
	msg, err := getMessageBody(e, getFromSES)
	if err != nil {
		return err
	}
	return s.sendMail(e, msg)
}

func (s *sesSender) CanSendEmail(smtpSettings fleet.SMTPSettings) bool {
	return s.client != nil
}

func NewSESSender(region, endpointURL, id, secret, stsAssumeRoleArn, stsExternalID, sourceArn string) (*sesSender, error) {
	conf := &aws.Config{
		Region:   &region,
		Endpoint: &endpointURL, // empty string or nil will use default values
	}

	// Only provide static credentials if we have them
	// otherwise use the default credentials provider chain
	if id != "" && secret != "" {
		conf.Credentials = credentials.NewStaticCredentials(id, secret, "")
	}

	sess, err := session.NewSession(conf)
	if err != nil {
		return nil, fmt.Errorf("create SES client: %w", err)
	}

	if stsAssumeRoleArn != "" {
		creds := stscreds.NewCredentials(sess, stsAssumeRoleArn, func(provider *stscreds.AssumeRoleProvider) {
			if stsExternalID != "" {
				provider.ExternalID = &stsExternalID
			}
		})
		conf.Credentials = creds

		sess, err = session.NewSession(conf)

		if err != nil {
			return nil, fmt.Errorf("create SES client: %w", err)
		}
	}
	return &sesSender{client: ses.New(sess), sourceArn: sourceArn}, nil
}

func (s *sesSender) sendMail(e fleet.Email, msg []byte) error {
	toAddresses := make([]*string, len(e.To))
	for i := range e.To {
		t := e.To[i]
		toAddresses[i] = &t
	}

	_, err := s.client.SendRawEmail(&ses.SendRawEmailInput{
		Destinations: toAddresses,
		FromArn:      &s.sourceArn,
		RawMessage:   &ses.RawMessage{Data: msg},
		SourceArn:    &s.sourceArn,
	})
	if err != nil {
		return err
	}
	return nil
}
