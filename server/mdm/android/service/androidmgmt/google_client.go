package androidmgmt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/go-json-experiment/json"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// GoogleClient connects directly to Google's Android Management API. It is intended to be used for development/debugging.
type GoogleClient struct {
	logger                    kitlog.Logger
	mgmt                      *androidmanagement.Service
	androidServiceCredentials string
	androidProjectID          string
}

// Compile-time check to ensure that ProxyClient implements Client.
var _ Client = &GoogleClient{}

func NewGoogleClient(ctx context.Context, logger kitlog.Logger, getenv func(string) string) Client {
	androidServiceCredentials := getenv("FLEET_DEV_ANDROID_SERVICE_CREDENTIALS")
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

	mgmt, err := androidmanagement.NewService(ctx, option.WithCredentialsJSON([]byte(androidServiceCredentials)))
	if err != nil {
		level.Error(logger).Log("msg", "creating android management service", "err", err)
		return nil
	}
	return &GoogleClient{
		logger:                    logger,
		mgmt:                      mgmt,
		androidServiceCredentials: androidServiceCredentials,
		androidProjectID:          creds.ProjectID,
	}
}

func (g *GoogleClient) SignupURLsCreate(ctx context.Context, serverURL, callbackURL string) (*android.SignupDetails, error) {
	if g == nil || g.mgmt == nil {
		return nil, errors.New("android management service not initialized")
	}
	signupURL, err := g.mgmt.SignupUrls.Create().ProjectId(g.androidProjectID).CallbackUrl(callbackURL).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating signup url: %w", err)
	}
	return &android.SignupDetails{
		Url:  signupURL.Url,
		Name: signupURL.Name,
	}, nil
}

func (g *GoogleClient) EnterprisesCreate(ctx context.Context, req EnterprisesCreateRequest) (EnterprisesCreateResponse, error) {
	res := EnterprisesCreateResponse{}
	if g == nil || g.mgmt == nil {
		return res, errors.New("android management service not initialized")
	}

	topicName, err := g.createPubSubTopic(ctx, req.PubSubPushURL)
	if err != nil {
		return res, fmt.Errorf("creating PubSub topic: %w", err)
	}

	enterprise, err := g.mgmt.Enterprises.Create(&androidmanagement.Enterprise{
		EnabledNotificationTypes: req.EnabledNotificationTypes,
		PubsubTopic:              topicName,
	}).
		ProjectId(g.androidProjectID).
		EnterpriseToken(req.EnterpriseToken).
		SignupUrlName(req.SignupURLName).
		Context(ctx).
		Do()
	switch {
	case googleapi.IsNotModified(err):
		return res, fmt.Errorf("android enterprise %s was already created", req.SignupURLName)
	case err != nil:
		return res, fmt.Errorf("creating enterprise: %w", err)
	}
	res.EnterpriseName = enterprise.Name
	res.TopicName = topicName
	return res, nil
}

func (g *GoogleClient) createPubSubTopic(ctx context.Context, pushURL string) (string, error) {
	pubSubClient, err := pubsub.NewClient(ctx, g.androidProjectID, option.WithCredentialsJSON([]byte(g.androidServiceCredentials)))
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

func (g *GoogleClient) EnterprisesPoliciesPatch(ctx context.Context, policyName string, policy *androidmanagement.Policy) error {
	_, err := g.mgmt.Enterprises.Policies.Patch(policyName, policy).Context(ctx).Do()
	switch {
	case googleapi.IsNotModified(err):
		g.logger.Log("msg", "Android policy not modified", "policy_name", policyName)
	case err != nil:
		return fmt.Errorf("patching policy %s: %w", policyName, err)
	}
	return nil
}

func (g *GoogleClient) EnterprisesEnrollmentTokensCreate(ctx context.Context, enterpriseName string, token *androidmanagement.EnrollmentToken,
) (*androidmanagement.EnrollmentToken, error) {
	if g == nil || g.mgmt == nil {
		return nil, errors.New("android management service not initialized")
	}
	token, err := g.mgmt.Enterprises.EnrollmentTokens.Create(enterpriseName, token).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("creating enrollment token: %w", err)
	}
	return token, nil
}

func (g *GoogleClient) EnterpriseDelete(ctx context.Context, enterpriseName string) error {
	if g == nil || g.mgmt == nil {
		return errors.New("android management service not initialized")
	}

	// To find out the enterprise's PubSub topic, we need to get the enterprise first
	enterprise, err := g.mgmt.Enterprises.Get(enterpriseName).Context(ctx).Do()
	if err != nil {
		level.Error(g.logger).Log("msg", "getting enterprise; perhaps it was already deleted?", "err", err, "enterprise_name", enterpriseName)
		return nil
	}

	_, err = g.mgmt.Enterprises.Delete(enterpriseName).Do()
	switch {
	case googleapi.IsNotModified(err):
		level.Info(g.logger).Log("msg", "enterprise was already deleted", "enterprise_name", enterpriseName)
		return nil
	case err != nil:
		return fmt.Errorf("deleting enterprise %s: %w", enterpriseName, err)
	}

	// Delete the PubSub topic if it exists
	if enterprise == nil || len(enterprise.PubsubTopic) == 0 {
		return nil
	}
	topicID, err := getLastPart(ctx, enterprise.PubsubTopic)
	if err != nil || len(topicID) == 0 {
		level.Error(g.logger).Log("msg", "getting last part of PubSub topic", "err", err, "topic", enterprise.PubsubTopic)
		return nil
	}

	pubSubClient, err := pubsub.NewClient(ctx, g.androidProjectID, option.WithCredentialsJSON([]byte(g.androidServiceCredentials)))
	if err != nil {
		return fmt.Errorf("creating PubSub client: %w", err)
	}
	defer pubSubClient.Close()
	err = pubSubClient.Topic(topicID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("deleting PubSub topic %s: %w", enterprise.PubsubTopic, err)
	}
	// Delete the subscription, which has the same ID as the topic.
	err = pubSubClient.Subscription(topicID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("deleting PubSub subscription %s: %w", topicID, err)
	}

	return nil
}

func (g *GoogleClient) SetAuthenticationSecret(_ string) error {
	return nil
}

func getLastPart(ctx context.Context, name string) (string, error) {
	nameParts := strings.Split(name, "/")
	if len(nameParts) == 0 {
		return "", ctxerr.Errorf(ctx, "invalid Google resource name: %s", name)
	}
	return nameParts[len(nameParts)-1], nil
}
