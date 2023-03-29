package mail

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"net/url"
)

type sesSender struct {
	client sesiface.SESAPI
}

func (s *sesSender) SendEmail(e fleet.Email) error {
	if s.client == nil {
		return errors.New("ses sender not configured")
	}
	if !e.Config.SMTPSettings.SMTPConfigured {
		return errors.New("email not configured")
	}
	msg, err := getMessageBody(e)
	if err != nil {
		return err
	}
	return s.sendMail(e, msg)
}

func NewSESSender(region, endpointURL, id, secret, stsAssumeRoleArn string) (*sesSender, error) {
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
		creds := stscreds.NewCredentials(sess, stsAssumeRoleArn)
		conf.Credentials = creds

		sess, err = session.NewSession(conf)

		if err != nil {
			return nil, fmt.Errorf("create SES client: %w", err)
		}
	}
	return &sesSender{client: ses.New(sess)}, nil
}

func (s *sesSender) sendMail(e fleet.Email, msg []byte) error {
	toAddresses := make([]*string, len(e.To))
	for i := range e.To {
		t := e.To[i]
		toAddresses[i] = &t
	}
	replyToAddresses := make([]*string, 1)
	if len(e.Config.SMTPSettings.SMTPSenderAddress) == 0 {
		serverURL, err := url.Parse(e.Config.ServerSettings.ServerURL)
		if err != nil {
			return err
		}
		reply := fmt.Sprintf("do-not-reply@%s", serverURL.Host)
		replyToAddresses[0] = &reply
	} else {
		replyToAddresses[0] = &e.Config.SMTPSettings.SMTPSenderAddress
	}
	subj := e.Subject
	body := string(msg)
	message := &ses.Message{
		Subject: &ses.Content{Data: &subj},
		Body: &ses.Body{
			Text: &ses.Content{Data: &body},
		},
	}
	destination := &ses.Destination{ToAddresses: toAddresses}
	_, err := s.client.SendEmail(&ses.SendEmailInput{
		Message:          message,
		Destination:      destination,
		ReplyToAddresses: replyToAddresses,
	})
	if err != nil {
		return err
	}
	return nil
}
