package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/logging"
)

// buildLoggingConfig maps the Fleet config into the common logging.Config
// shared by the osquery status, result, and audit JSON loggers. The
// topic/stream/plugin fields specific to each logger are applied by the
// caller before constructing it.
func buildLoggingConfig(cfg config.FleetConfig) logging.Config {
	return logging.Config{
		Filesystem: logging.FilesystemConfig{
			EnableLogRotation:    cfg.Filesystem.EnableLogRotation,
			EnableLogCompression: cfg.Filesystem.EnableLogCompression,
			MaxSize:              cfg.Filesystem.MaxSize,
			MaxAge:               cfg.Filesystem.MaxAge,
			MaxBackups:           cfg.Filesystem.MaxBackups,
		},
		Webhook: logging.WebhookConfig{},
		Firehose: logging.FirehoseConfig{
			Region:           cfg.Firehose.Region,
			EndpointURL:      cfg.Firehose.EndpointURL,
			AccessKeyID:      cfg.Firehose.AccessKeyID,
			SecretAccessKey:  cfg.Firehose.SecretAccessKey,
			StsAssumeRoleArn: cfg.Firehose.StsAssumeRoleArn,
			StsExternalID:    cfg.Firehose.StsExternalID,
		},
		Kinesis: logging.KinesisConfig{
			Region:           cfg.Kinesis.Region,
			EndpointURL:      cfg.Kinesis.EndpointURL,
			AccessKeyID:      cfg.Kinesis.AccessKeyID,
			SecretAccessKey:  cfg.Kinesis.SecretAccessKey,
			StsAssumeRoleArn: cfg.Kinesis.StsAssumeRoleArn,
			StsExternalID:    cfg.Kinesis.StsExternalID,
		},
		Lambda: logging.LambdaConfig{
			Region:           cfg.Lambda.Region,
			AccessKeyID:      cfg.Lambda.AccessKeyID,
			SecretAccessKey:  cfg.Lambda.SecretAccessKey,
			StsAssumeRoleArn: cfg.Lambda.StsAssumeRoleArn,
			StsExternalID:    cfg.Lambda.StsExternalID,
		},
		PubSub: logging.PubSubConfig{
			Project: cfg.PubSub.Project,
		},
		KafkaREST: logging.KafkaRESTConfig{
			ProxyHost:        cfg.KafkaREST.ProxyHost,
			ContentTypeValue: cfg.KafkaREST.ContentTypeValue,
			Timeout:          cfg.KafkaREST.Timeout,
		},
		Nats: logging.NatsConfig{
			Server:            cfg.Nats.Server,
			CredFile:          cfg.Nats.CredFile,
			NKeyFile:          cfg.Nats.NKeyFile,
			TLSClientCertFile: cfg.Nats.TLSClientCrtFile,
			TLSClientKeyFile:  cfg.Nats.TLSClientKeyFile,
			CACertFile:        cfg.Nats.CACrtFile,
			Compression:       cfg.Nats.Compression,
			JetStream:         cfg.Nats.JetStream,
			Timeout:           cfg.Nats.Timeout,
		},
		Splunk: logging.SplunkConfig{
			URL:                cfg.Splunk.URL,
			Token:              cfg.Splunk.Token,
			Index:              cfg.Splunk.Index,
			Source:             cfg.Splunk.Source,
			SourceType:         cfg.Splunk.SourceType,
			InsecureSkipVerify: cfg.Splunk.InsecureSkipVerify,
		},
	}
}

// shouldEnableAuditLog reports whether the audit JSON logger should be
// constructed: audit logging is a Fleet Premium feature and must also be
// explicitly enabled in config.
func shouldEnableAuditLog(license *fleet.LicenseInfo, cfg config.FleetConfig) bool {
	return license.IsPremium() && cfg.Activity.EnableAuditLog
}

// initOsqueryLogging constructs the osquery status and result JSON loggers
// and, when enabled for a premium license, the audit logger. Failures go
// through initFatal. Returns nil values on the failure path so the function
// is safe when initFatal does not terminate (e.g., tests using a recorder).
func initOsqueryLogging(
	ctx context.Context,
	cfg config.FleetConfig,
	license *fleet.LicenseInfo,
	logger *slog.Logger,
	initFatal func(err error, msg string),
) (status fleet.JSONLogger, result fleet.JSONLogger, audit fleet.JSONLogger) {
	if license == nil {
		initFatal(errors.New("license was nil"), "initializing osqueryd logging")
		return nil, nil, nil
	}

	loggingConfig := buildLoggingConfig(cfg)

	// Set specific configuration to osqueryd status logs.
	loggingConfig.Plugin = cfg.Osquery.StatusLogPlugin
	loggingConfig.Filesystem.LogFile = cfg.Filesystem.StatusLogFile
	loggingConfig.Webhook.URL = cfg.Webhook.StatusURL
	loggingConfig.Firehose.StreamName = cfg.Firehose.StatusStream
	loggingConfig.Kinesis.StreamName = cfg.Kinesis.StatusStream
	loggingConfig.Lambda.Function = cfg.Lambda.StatusFunction
	loggingConfig.PubSub.Topic = cfg.PubSub.StatusTopic
	loggingConfig.PubSub.AddAttributes = false // only used by result logs
	loggingConfig.KafkaREST.Topic = cfg.KafkaREST.StatusTopic
	loggingConfig.Nats.Subject = cfg.Nats.StatusSubject

	statusLogger, err := logging.NewJSONLogger(ctx, "status", loggingConfig, logger)
	if err != nil {
		initFatal(err, "initializing osqueryd status logging")
		return nil, nil, nil
	}

	// Set specific configuration to osqueryd result logs.
	loggingConfig.Plugin = cfg.Osquery.ResultLogPlugin
	loggingConfig.Filesystem.LogFile = cfg.Filesystem.ResultLogFile
	loggingConfig.Webhook.URL = cfg.Webhook.ResultURL
	loggingConfig.Firehose.StreamName = cfg.Firehose.ResultStream
	loggingConfig.Kinesis.StreamName = cfg.Kinesis.ResultStream
	loggingConfig.Lambda.Function = cfg.Lambda.ResultFunction
	loggingConfig.PubSub.Topic = cfg.PubSub.ResultTopic
	loggingConfig.PubSub.AddAttributes = cfg.PubSub.AddAttributes
	loggingConfig.KafkaREST.Topic = cfg.KafkaREST.ResultTopic
	loggingConfig.Nats.Subject = cfg.Nats.ResultSubject

	resultLogger, err := logging.NewJSONLogger(ctx, "result", loggingConfig, logger)
	if err != nil {
		initFatal(err, "initializing osqueryd result logging")
		return nil, nil, nil
	}

	var auditLogger fleet.JSONLogger
	if shouldEnableAuditLog(license, cfg) {
		// Set specific configuration to audit logs.
		loggingConfig.Plugin = cfg.Activity.AuditLogPlugin
		loggingConfig.Filesystem.LogFile = cfg.Filesystem.AuditLogFile
		loggingConfig.Firehose.StreamName = cfg.Firehose.AuditStream
		loggingConfig.Kinesis.StreamName = cfg.Kinesis.AuditStream
		loggingConfig.Lambda.Function = cfg.Lambda.AuditFunction
		loggingConfig.PubSub.Topic = cfg.PubSub.AuditTopic
		loggingConfig.PubSub.AddAttributes = false // only used by result logs
		loggingConfig.KafkaREST.Topic = cfg.KafkaREST.AuditTopic
		loggingConfig.Nats.Subject = cfg.Nats.AuditSubject

		auditLogger, err = logging.NewJSONLogger(ctx, "audit", loggingConfig, logger)
		if err != nil {
			initFatal(err, "initializing audit logging")
			return nil, nil, nil
		}
	}

	return statusLogger, resultLogger, auditLogger
}
