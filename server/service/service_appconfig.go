package service

import (
	"context"
	"html/template"
	"strings"

	"github.com/fleetdm/fleet/v4/server"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
)

func (svc *Service) NewAppConfig(ctx context.Context, p fleet.AppConfig) (*fleet.AppConfig, error) {
	// skipauth: No user context yet when the app config is first created.
	svc.authz.SkipAuthorization(ctx)

	newConfig, err := svc.ds.NewAppConfig(ctx, &p)
	if err != nil {
		return nil, err
	}

	// Set up a default enroll secret
	secret := svc.config.Packaging.GlobalEnrollSecret
	if secret == "" {
		secret, err = server.GenerateRandomText(fleet.EnrollSecretDefaultLength)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate enroll secret string")
		}
	}
	secrets := []*fleet.EnrollSecret{
		{
			Secret: secret,
		},
	}
	err = svc.ds.ApplyEnrollSecrets(ctx, nil, secrets)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "save enroll secret")
	}

	return newConfig, nil
}

func (svc *Service) sendTestEmail(ctx context.Context, config *fleet.AppConfig) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	var smtpSettings fleet.SMTPSettings
	if config.SMTPSettings != nil {
		smtpSettings = *config.SMTPSettings
	}

	testMail := fleet.Email{
		Subject: "Hello from Fleet",
		To:      []string{vc.User.Email},
		Mailer: &mail.SMTPTestMailer{
			BaseURL:  template.URL(config.ServerSettings.ServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
		},
		SMTPSettings: smtpSettings,
		ServerURL:    config.ServerSettings.ServerURL,
	}

	if err := mail.Test(svc.mailService, testMail); err != nil {
		return endpoint_utils.MailError{Message: err.Error()}
	}
	return nil
}

func cleanupURL(url string) string {
	return strings.TrimRight(strings.Trim(url, " \t\n"), "/")
}

func (svc *Service) License(ctx context.Context) (*fleet.LicenseInfo, error) {
	if !svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceToken) {
		if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	lic, _ := license.FromContext(ctx)
	return lic, nil
}

func (svc *Service) SetupRequired(ctx context.Context) (bool, error) {
	hasUsers, err := svc.ds.HasUsers(ctx)
	if err != nil {
		return false, err
	}
	return !hasUsers, nil
}

func (svc *Service) UpdateIntervalConfig(ctx context.Context) (*fleet.UpdateIntervalConfig, error) {
	return &fleet.UpdateIntervalConfig{
		OSQueryDetail: svc.config.Osquery.DetailUpdateInterval,
		OSQueryPolicy: svc.config.Osquery.PolicyUpdateInterval,
	}, nil
}

func (svc *Service) VulnerabilitiesConfig(ctx context.Context) (*fleet.VulnerabilitiesConfig, error) {
	return &fleet.VulnerabilitiesConfig{
		DatabasesPath:               svc.config.Vulnerabilities.DatabasesPath,
		Periodicity:                 svc.config.Vulnerabilities.Periodicity,
		CPEDatabaseURL:              svc.config.Vulnerabilities.CPEDatabaseURL,
		CPETranslationsURL:          svc.config.Vulnerabilities.CPETranslationsURL,
		CVEFeedPrefixURL:            svc.config.Vulnerabilities.CVEFeedPrefixURL,
		CurrentInstanceChecks:       svc.config.Vulnerabilities.CurrentInstanceChecks,
		DisableDataSync:             svc.config.Vulnerabilities.DisableDataSync,
		RecentVulnerabilityMaxAge:   svc.config.Vulnerabilities.RecentVulnerabilityMaxAge,
		DisableWinOSVulnerabilities: svc.config.Vulnerabilities.DisableWinOSVulnerabilities,
	}, nil
}

func (svc *Service) LoggingConfig(ctx context.Context) (*fleet.Logging, error) {
	conf := svc.config
	logging := &fleet.Logging{
		Debug: conf.Logging.Debug,
		Json:  conf.Logging.JSON,
	}

	loggings := []struct {
		plugin string
		target *fleet.LoggingPlugin
	}{
		{
			plugin: conf.Osquery.StatusLogPlugin,
			target: &logging.Status,
		},
		{
			plugin: conf.Osquery.ResultLogPlugin,
			target: &logging.Result,
		},
	}

	if conf.Activity.EnableAuditLog {
		loggings = append(loggings, struct {
			plugin string
			target *fleet.LoggingPlugin
		}{
			plugin: conf.Activity.AuditLogPlugin,
			target: &logging.Audit,
		})
	}

	for _, lp := range loggings {
		switch lp.plugin {
		case "", "filesystem":
			*lp.target = fleet.LoggingPlugin{
				Plugin: "filesystem",
				Config: fleet.FilesystemConfig{
					FilesystemConfig: conf.Filesystem,
				},
			}
		case "kinesis":
			*lp.target = fleet.LoggingPlugin{
				Plugin: "kinesis",
				Config: fleet.KinesisConfig{
					Region:       conf.Kinesis.Region,
					StatusStream: conf.Kinesis.StatusStream,
					ResultStream: conf.Kinesis.ResultStream,
					AuditStream:  conf.Kinesis.AuditStream,
				},
			}
		case "firehose":
			*lp.target = fleet.LoggingPlugin{
				Plugin: "firehose",
				Config: fleet.FirehoseConfig{
					Region:       conf.Firehose.Region,
					StatusStream: conf.Firehose.StatusStream,
					ResultStream: conf.Firehose.ResultStream,
					AuditStream:  conf.Firehose.AuditStream,
				},
			}
		case "lambda":
			*lp.target = fleet.LoggingPlugin{
				Plugin: "lambda",
				Config: fleet.LambdaConfig{
					Region:         conf.Lambda.Region,
					StatusFunction: conf.Lambda.StatusFunction,
					ResultFunction: conf.Lambda.ResultFunction,
					AuditFunction:  conf.Lambda.AuditFunction,
				},
			}
		case "pubsub":
			*lp.target = fleet.LoggingPlugin{
				Plugin: "pubsub",
				Config: fleet.PubSubConfig{
					PubSubConfig: conf.PubSub,
				},
			}
		case "stdout":
			*lp.target = fleet.LoggingPlugin{Plugin: "stdout"}
		case "kafkarest":
			*lp.target = fleet.LoggingPlugin{
				Plugin: "kafkarest",
				Config: fleet.KafkaRESTConfig{
					StatusTopic: conf.KafkaREST.StatusTopic,
					ResultTopic: conf.KafkaREST.ResultTopic,
					AuditTopic:  conf.KafkaREST.AuditTopic,
					ProxyHost:   conf.KafkaREST.ProxyHost,
				},
			}
		default:
			return nil, ctxerr.Errorf(ctx, "unrecognized logging plugin: %s", lp.plugin)
		}
	}
	return logging, nil
}

func (svc *Service) EmailConfig(ctx context.Context) (*fleet.EmailConfig, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	conf := svc.config
	var email *fleet.EmailConfig
	switch conf.Email.EmailBackend {
	case "ses":
		email = &fleet.EmailConfig{
			Backend: conf.Email.EmailBackend,
			Config: fleet.SESConfig{
				Region:    conf.SES.Region,
				SourceARN: conf.SES.SourceArn,
			},
		}
	default:
		// SES is the only email provider configured as server envs/yaml file, the default implementation, SMTP, is configured via API/UI
		// SMTP config gets its own dedicated section in the AppConfig response
	}

	return email, nil
}
