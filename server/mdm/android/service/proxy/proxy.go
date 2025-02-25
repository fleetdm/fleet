package proxy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/go-json-experiment/json"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Proxy is a temporary placeholder as an interface to the Google API.
// Once the real proxy is implemented on fleetdm.com, this package will be removed.

var (
	// Required env vars to use the proxy
	androidServiceCredentials = os.Getenv("FLEET_DEV_ANDROID_SERVICE_CREDENTIALS")
	androidProjectID          string
)

type Proxy struct {
	logger kitlog.Logger
	mgmt   *androidmanagement.Service
}

func NewProxy(ctx context.Context, logger kitlog.Logger) *Proxy {
	if androidServiceCredentials == "" {
		return nil
	}

	type credentials struct {
		ProjectID string `json:"project_id"`
	}

	var creds credentials
	err := json.Unmarshal([]byte(androidServiceCredentials), &creds)
	if err != nil {
		level.Error(logger).Log("msg", "unmarshaling android service credentials", "err", err)
		return nil
	}
	androidProjectID = creds.ProjectID

	mgmt, err := androidmanagement.NewService(ctx, option.WithCredentialsJSON([]byte(androidServiceCredentials)))
	if err != nil {
		level.Error(logger).Log("msg", "creating android management service", "err", err)
		return nil
	}
	return &Proxy{
		logger: logger,
		mgmt:   mgmt,
	}
}

func (p *Proxy) SignupURLsCreate(callbackURL string) (*android.SignupDetails, error) {
	if p == nil || p.mgmt == nil {
		return nil, errors.New("android management service not initialized")
	}
	signupURL, err := p.mgmt.SignupUrls.Create().ProjectId(androidProjectID).CallbackUrl(callbackURL).Do()
	if err != nil {
		return nil, fmt.Errorf("creating signup url: %w", err)
	}
	return &android.SignupDetails{
		Url:  signupURL.Url,
		Name: signupURL.Name,
	}, nil
}

func (p *Proxy) EnterprisesCreate(ctx context.Context, enabledNotificationTypes []string, enterpriseToken string,
	signupUrlName string, pushURL string) (string, string, error) {
	if p == nil || p.mgmt == nil {
		return "", "", errors.New("android management service not initialized")
	}

	topicName, err := p.createPubSubTopic(ctx, pushURL)
	if err != nil {
		return "", "", fmt.Errorf("creating PubSub topic: %w", err)
	}

	enterprise, err := p.mgmt.Enterprises.Create(&androidmanagement.Enterprise{
		EnabledNotificationTypes: enabledNotificationTypes,
		PubsubTopic:              topicName,
	}).
		ProjectId(androidProjectID).
		EnterpriseToken(enterpriseToken).
		SignupUrlName(signupUrlName).
		Do()
	switch {
	case googleapi.IsNotModified(err):
		return "", "", fmt.Errorf("android enterprise %s was already created", signupUrlName)
	case err != nil:
		return "", "", fmt.Errorf("creating enterprise: %w", err)
	}
	return enterprise.Name, topicName, nil
}

func (p *Proxy) createPubSubTopic(ctx context.Context, pushURL string) (string, error) {
	pubSubClient, err := pubsub.NewClient(ctx, androidProjectID, option.WithCredentialsJSON([]byte(androidServiceCredentials)))
	if err != nil {
		return "", fmt.Errorf("creating PubSub client: %w", err)
	}
	defer pubSubClient.Close()
	pubSubTopic := "a" + uuid.NewString() // PubSub topic names must start with a letter
	topicConfig := pubsub.TopicConfig{
		// Message retention is free for 1 day, so we default to that.
		// Both the topic and subscription retention durations should be 1 day since Google uses whatever is longer.
		// https://cloud.google.com/pubsub/pricing
		RetentionDuration: 24 * time.Hour,
	}
	topic, err := pubSubClient.CreateTopicWithConfig(ctx, pubSubTopic, &topicConfig)
	if err != nil {
		return "", fmt.Errorf("creating PubSub topic: %w", err)
	}
	policy, err := topic.IAM().Policy(ctx) // Ensure the topic exists before creating the subscription
	if err != nil {
		return "", fmt.Errorf("getting PubSub topic policy: %w", err)
	}
	policy.Add("serviceAccount:android-cloud-policy@system.gserviceaccount.com", "roles/pubsub.publisher")
	if err := topic.IAM().SetPolicy(ctx, policy); err != nil {
		return "", fmt.Errorf("setting PubSub subscription policy: %w", err)
	}
	// TODO(fleetdm.com): Retry SetPolicy since it may fail if IAM policies are being modified concurrently

	// Note: We could add a second level of authentication for the subscription, where, upon receiving a message,
	// Fleet server does an API call to Google to verify the message validity.
	_, err = pubSubClient.CreateSubscription(ctx, pubSubTopic, pubsub.SubscriptionConfig{
		Topic:             topic,
		AckDeadline:       60 * time.Second,
		RetentionDuration: 24 * time.Hour,
		PushConfig: pubsub.PushConfig{
			Endpoint: pushURL,
		},
	})
	if err != nil {
		return "", fmt.Errorf("creating PubSub subscription: %w", err)
	}

	// TODO(fleetdm.com): Cleanup the PubSub topics not associated with enterprises (e.g. if the enterprise creation fails)

	return topic.String(), nil
}

func (p *Proxy) EnterprisesPoliciesPatch(enterpriseID string, policyName string, policy *androidmanagement.Policy) error {
	fullPolicyName := fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, policyName)
	_, err := p.mgmt.Enterprises.Policies.Patch(fullPolicyName, policy).Do()
	switch {
	case googleapi.IsNotModified(err):
		p.logger.Log("msg", "Android policy not modified", "enterprise_id", enterpriseID, "policy_name", policyName)
	case err != nil:
		return fmt.Errorf("patching policy %s: %w", fullPolicyName, err)
	}
	return nil
}

func (p *Proxy) EnterprisesEnrollmentTokensCreate(enterpriseName string, token *androidmanagement.EnrollmentToken,
) (*androidmanagement.EnrollmentToken, error) {
	if p == nil || p.mgmt == nil {
		return nil, errors.New("android management service not initialized")
	}
	token, err := p.mgmt.Enterprises.EnrollmentTokens.Create(enterpriseName, token).Do()
	if err != nil {
		return nil, fmt.Errorf("creating enrollment token: %w", err)
	}
	return token, nil
}

func (p *Proxy) EnterpriseDelete(enterpriseID string) error {
	if p == nil || p.mgmt == nil {
		return errors.New("android management service not initialized")
	}

	_, err := p.mgmt.Enterprises.Delete("enterprises/" + enterpriseID).Do()
	switch {
	case googleapi.IsNotModified(err):
		level.Info(p.logger).Log("msg", "enterprise was already deleted", "enterprise_id", enterpriseID)
		return nil
	case err != nil:
		return fmt.Errorf("deleting enterprise %s: %w", enterpriseID, err)
	}
	return nil
}
