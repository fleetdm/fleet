package service

import (
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/ptr"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/kolide/kit/version"
	"github.com/pkg/errors"
)

// mailError is set when an error performing mail operations
type mailError struct {
	message string
}

func (e mailError) Error() string {
	return fmt.Sprintf("a mail error occurred: %s", e.message)
}

func (e mailError) MailError() []map[string]string {
	return []map[string]string{
		{
			"name":   "base",
			"reason": e.message,
		},
	}
}

func (svc *Service) NewAppConfig(ctx context.Context, p fleet.AppConfig) (*fleet.AppConfig, error) {
	// skipauth: No user context yet when the app config is first created.
	svc.authz.SkipAuthorization(ctx)

	if p.ServerSettings == nil {
		p.ServerSettings = &fleet.ServerSettings{}
	}
	// New installations start with analytics enabled by default
	if p.ServerSettings.EnableAnalytics == nil {
		p.ServerSettings.EnableAnalytics = ptr.Bool(true)
	}
	newConfig, err := svc.ds.NewAppConfig(&p)
	if err != nil {
		return nil, err
	}

	// Set up a default enroll secret
	secret, err := server.GenerateRandomText(fleet.EnrollSecretDefaultLength)
	if err != nil {
		return nil, errors.Wrap(err, "generate enroll secret string")
	}
	secrets := []*fleet.EnrollSecret{
		{
			Secret: secret,
		},
	}
	err = svc.ds.ApplyEnrollSecrets(nil, secrets)
	if err != nil {
		return nil, errors.Wrap(err, "save enroll secret")
	}

	return newConfig, nil
}

func (svc *Service) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.AppConfig()
}

func (svc *Service) sendTestEmail(ctx context.Context, config *fleet.AppConfig) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	testMail := fleet.Email{
		Subject: "Hello from Fleet",
		To:      []string{vc.User.Email},
		Mailer: &mail.SMTPTestMailer{
			BaseURL:  template.URL(config.GetString("server_settings.server_url") + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
		},
		Config: config,
	}

	if err := mail.Test(svc.mailService, testMail); err != nil {
		return mailError{message: err.Error()}
	}
	return nil

}

func (svc *Service) ModifyAppConfig(ctx context.Context, p fleet.AppConfig) (*fleet.AppConfig, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	oldAppConfig, err := svc.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	if p.SMTPSettings != nil {
		enabled := p.SMTPSettings.SMTPEnabled
		if enabled == nil && oldAppConfig.GetBool("smtp_settings.configured") || (enabled != nil && *enabled) {
			if err = svc.sendTestEmail(ctx, &p); err != nil {
				return nil, err
			}
			p.SMTPSettings.SMTPConfigured = ptr.Bool(true)
		} else if enabled != nil && !*enabled {
			p.SMTPSettings.SMTPConfigured = ptr.Bool(false)
		}
	}

	if err := svc.ds.SaveAppConfig(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func cleanupURL(url string) string {
	return strings.TrimRight(strings.Trim(url, " \t\n"), "/")
}

func (svc *Service) ApplyEnrollSecretSpec(ctx context.Context, spec *fleet.EnrollSecretSpec) error {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionWrite); err != nil {
		return err
	}

	for _, s := range spec.Secrets {
		if s.Secret == "" {
			return errors.New("enroll secret must not be empty")
		}
	}

	return svc.ds.ApplyEnrollSecrets(nil, spec.Secrets)
}

func (svc *Service) GetEnrollSecretSpec(ctx context.Context) (*fleet.EnrollSecretSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	secrets, err := svc.ds.GetEnrollSecrets(nil)
	if err != nil {
		return nil, err
	}
	return &fleet.EnrollSecretSpec{Secrets: secrets}, nil
}

func (svc *Service) Version(ctx context.Context) (*version.Info, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	info := version.Version()
	return &info, nil
}

func (svc *Service) License(ctx context.Context) (*fleet.LicenseInfo, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return &svc.license, nil
}

func (svc *Service) SetupRequired(ctx context.Context) (bool, error) {
	users, err := svc.ds.ListUsers(fleet.UserListOptions{ListOptions: fleet.ListOptions{Page: 0, PerPage: 1}})
	if err != nil {
		return false, err
	}
	if len(users) == 0 {
		return true, nil
	}
	return false, nil
}

func (svc *Service) LoggingConfig(ctx context.Context) (*fleet.Logging, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	conf := svc.config
	logging := &fleet.Logging{
		Debug: conf.Logging.Debug,
		Json:  conf.Logging.JSON,
	}

	switch conf.Osquery.StatusLogPlugin {
	case "", "filesystem":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "filesystem",
			Config: fleet.FilesystemConfig{FilesystemConfig: conf.Filesystem},
		}
	case "kinesis":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "kinesis",
			Config: fleet.KinesisConfig{
				Region:       conf.Kinesis.Region,
				StatusStream: conf.Kinesis.StatusStream,
				ResultStream: conf.Kinesis.ResultStream,
			},
		}
	case "firehose":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "firehose",
			Config: fleet.FirehoseConfig{
				Region:       conf.Firehose.Region,
				StatusStream: conf.Firehose.StatusStream,
				ResultStream: conf.Firehose.ResultStream,
			},
		}
	case "lambda":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "lambda",
			Config: fleet.LambdaConfig{
				Region:         conf.Lambda.Region,
				StatusFunction: conf.Lambda.StatusFunction,
				ResultFunction: conf.Lambda.ResultFunction,
			},
		}
	case "pubsub":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "pubsub",
			Config: fleet.PubSubConfig{PubSubConfig: conf.PubSub},
		}
	case "stdout":
		logging.Status = fleet.LoggingPlugin{Plugin: "stdout"}
	default:
		return nil, errors.Errorf("unrecognized logging plugin: %s", conf.Osquery.StatusLogPlugin)
	}

	switch conf.Osquery.ResultLogPlugin {
	case "", "filesystem":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "filesystem",
			Config: fleet.FilesystemConfig{FilesystemConfig: conf.Filesystem},
		}
	case "kinesis":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "kinesis",
			Config: fleet.KinesisConfig{
				Region:       conf.Kinesis.Region,
				StatusStream: conf.Kinesis.StatusStream,
				ResultStream: conf.Kinesis.ResultStream,
			},
		}
	case "firehose":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "firehose",
			Config: fleet.FirehoseConfig{
				Region:       conf.Firehose.Region,
				StatusStream: conf.Firehose.StatusStream,
				ResultStream: conf.Firehose.ResultStream,
			},
		}
	case "lambda":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "lambda",
			Config: fleet.LambdaConfig{
				Region:         conf.Lambda.Region,
				StatusFunction: conf.Lambda.StatusFunction,
				ResultFunction: conf.Lambda.ResultFunction,
			},
		}
	case "pubsub":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "pubsub",
			Config: fleet.PubSubConfig{PubSubConfig: conf.PubSub},
		}
	case "stdout":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "stdout",
		}
	default:
		return nil, errors.Errorf("unrecognized logging plugin: %s", conf.Osquery.ResultLogPlugin)

	}
	return logging, nil
}
