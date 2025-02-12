package proxy

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/go-json-experiment/json"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/option"
)

var (
	// Required env vars to use the proxy
	androidServiceCredentials = os.Getenv("FLEET_DEV_ANDROID_SERVICE_CREDENTIALS")
	androidPubSubTopic        = os.Getenv("FLEET_DEV_ANDROID_PUBSUB_TOPIC")
	androidProjectID          string
)

type Proxy struct {
	logger kitlog.Logger
	mgmt   *androidmanagement.Service
}

func NewProxy(ctx context.Context, logger kitlog.Logger) *Proxy {
	if androidServiceCredentials == "" || androidPubSubTopic == "" {
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
