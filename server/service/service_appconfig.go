package service

import (
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
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

	newConfig, err := svc.ds.NewAppConfig(ctx, &p)
	if err != nil {
		return nil, err
	}

	// Set up a default enroll secret
	secret, err := server.GenerateRandomText(fleet.EnrollSecretDefaultLength)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate enroll secret string")
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

	testMail := fleet.Email{
		Subject: "Hello from Fleet",
		To:      []string{vc.User.Email},
		Mailer: &mail.SMTPTestMailer{
			BaseURL:  template.URL(config.ServerSettings.ServerURL + svc.config.Server.URLPrefix),
			AssetURL: getAssetURL(),
		},
		Config: config,
	}

	if err := mail.Test(svc.mailService, testMail); err != nil {
		return mailError{message: err.Error()}
	}
	return nil
}

func (svc *Service) makeTestJiraRequest(ctx context.Context, jiraSettings *fleet.JiraIntegration) error {
	client, err := externalsvc.NewJiraClient(&externalsvc.JiraOptions{
		BaseURL:           jiraSettings.URL,
		BasicAuthUsername: jiraSettings.Username,
		BasicAuthPassword: jiraSettings.APIToken,
		ProjectKey:        jiraSettings.ProjectKey,
	})
	if err != nil {
		return &badRequestError{message: fmt.Sprintf("jira integration request failed: %s", err.Error())}
	}
	if _, err := client.GetProject(ctx); err != nil {
		return &badRequestError{message: fmt.Sprintf("jira integration request failed: %s", err.Error())}
	}
	return nil
}

func cleanupURL(url string) string {
	return strings.TrimRight(strings.Trim(url, " \t\n"), "/")
}

func (svc *Service) License(ctx context.Context) (*fleet.LicenseInfo, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return &svc.license, nil
}

func (svc *Service) SetupRequired(ctx context.Context) (bool, error) {
	users, err := svc.ds.ListUsers(ctx, fleet.UserListOptions{ListOptions: fleet.ListOptions{Page: 0, PerPage: 1}})
	if err != nil {
		return false, err
	}
	if len(users) == 0 {
		return true, nil
	}
	return false, nil
}

func (svc *Service) UpdateIntervalConfig(ctx context.Context) (*fleet.UpdateIntervalConfig, error) {
	return &fleet.UpdateIntervalConfig{
		OSQueryDetail: svc.config.Osquery.DetailUpdateInterval,
		OSQueryPolicy: svc.config.Osquery.PolicyUpdateInterval,
	}, nil
}

func (svc *Service) VulnerabilitiesConfig(ctx context.Context) (*fleet.VulnerabilitiesConfig, error) {
	return &fleet.VulnerabilitiesConfig{
		DatabasesPath:             svc.config.Vulnerabilities.DatabasesPath,
		Periodicity:               svc.config.Vulnerabilities.Periodicity,
		CPEDatabaseURL:            svc.config.Vulnerabilities.CPEDatabaseURL,
		CVEFeedPrefixURL:          svc.config.Vulnerabilities.CVEFeedPrefixURL,
		CurrentInstanceChecks:     svc.config.Vulnerabilities.CurrentInstanceChecks,
		DisableDataSync:           svc.config.Vulnerabilities.DisableDataSync,
		RecentVulnerabilityMaxAge: svc.config.Vulnerabilities.RecentVulnerabilityMaxAge,
	}, nil
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
	case "kafkarest":
		logging.Status = fleet.LoggingPlugin{
			Plugin: "kafkarest",
			Config: fleet.KafkaRESTConfig{
				StatusTopic: conf.KafkaREST.StatusTopic,
				ProxyHost:   conf.KafkaREST.ProxyHost,
			},
		}
	default:
		return nil, ctxerr.Errorf(ctx, "unrecognized logging plugin: %s", conf.Osquery.StatusLogPlugin)
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
	case "kafkarest":
		logging.Result = fleet.LoggingPlugin{
			Plugin: "kafkarest",
			Config: fleet.KafkaRESTConfig{
				ResultTopic: conf.KafkaREST.ResultTopic,
				ProxyHost:   conf.KafkaREST.ProxyHost,
			},
		}
	default:
		return nil, ctxerr.Errorf(ctx, "unrecognized logging plugin: %s", conf.Osquery.ResultLogPlugin)

	}
	return logging, nil
}
