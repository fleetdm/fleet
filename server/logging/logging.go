// Package logging provides logger "plugins" for writing osquery status and
// result logs to various destinations.
package logging

import (
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

type OsqueryLogger struct {
	Status fleet.JSONLogger
	Result fleet.JSONLogger
}

func New(config config.FleetConfig, logger log.Logger) (*OsqueryLogger, error) {
	var status, result fleet.JSONLogger
	var err error

	switch config.Osquery.StatusLogPlugin {
	case "":
		// Allow "" to mean filesystem for backwards compatibility
		level.Info(logger).Log("msg", "fleet_status_log_plugin not explicitly specified. Assuming 'filesystem'")
		fallthrough
	case "filesystem":
		status, err = NewFilesystemLogWriter(
			config.Filesystem.StatusLogFile,
			logger,
			config.Filesystem.EnableLogRotation,
			config.Filesystem.EnableLogCompression,
		)
		if err != nil {
			return nil, errors.Wrap(err, "create filesystem status logger")
		}
	case "firehose":
		status, err = NewFirehoseLogWriter(
			config.Firehose.Region,
			config.Firehose.EndpointURL,
			config.Firehose.AccessKeyID,
			config.Firehose.SecretAccessKey,
			config.Firehose.StsAssumeRoleArn,
			config.Firehose.StatusStream,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "create firehose status logger")
		}
	case "kinesis":
		status, err = NewKinesisLogWriter(
			config.Kinesis.Region,
			config.Kinesis.EndpointURL,
			config.Kinesis.AccessKeyID,
			config.Kinesis.SecretAccessKey,
			config.Kinesis.StsAssumeRoleArn,
			config.Kinesis.StatusStream,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "create kinesis status logger")
		}
	case "lambda":
		status, err = NewLambdaLogWriter(
			config.Lambda.Region,
			config.Lambda.AccessKeyID,
			config.Lambda.SecretAccessKey,
			config.Lambda.StsAssumeRoleArn,
			config.Lambda.StatusFunction,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "create lambda status logger")
		}
	case "pubsub":
		status, err = NewPubSubLogWriter(
			config.PubSub.Project,
			config.PubSub.StatusTopic,
			false,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "create pubsub status logger")
		}
	case "stdout":
		status, err = NewStdoutLogWriter()
		if err != nil {
			return nil, errors.Wrap(err, "create stdout status logger")
		}
	default:
		return nil, errors.Errorf(
			"unknown status log plugin: %s", config.Osquery.StatusLogPlugin,
		)
	}

	switch config.Osquery.ResultLogPlugin {
	case "":
		// Allow "" to mean filesystem for backwards compatibility
		level.Info(logger).Log("msg", "fleet_result_log_plugin not explicitly specified. Assuming 'filesystem'")
		fallthrough
	case "filesystem":
		result, err = NewFilesystemLogWriter(
			config.Filesystem.ResultLogFile,
			logger,
			config.Filesystem.EnableLogRotation,
			config.Filesystem.EnableLogCompression,
		)
		if err != nil {
			return nil, errors.Wrap(err, "create filesystem result logger")
		}
	case "firehose":
		result, err = NewFirehoseLogWriter(
			config.Firehose.Region,
			config.Firehose.EndpointURL,
			config.Firehose.AccessKeyID,
			config.Firehose.SecretAccessKey,
			config.Kinesis.StsAssumeRoleArn,
			config.Firehose.ResultStream,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "create firehose result logger")
		}
	case "kinesis":
		result, err = NewKinesisLogWriter(
			config.Kinesis.Region,
			config.Kinesis.EndpointURL,
			config.Kinesis.AccessKeyID,
			config.Kinesis.SecretAccessKey,
			config.Kinesis.StsAssumeRoleArn,
			config.Kinesis.ResultStream,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "create kinesis result logger")
		}
	case "lambda":
		result, err = NewLambdaLogWriter(
			config.Lambda.Region,
			config.Lambda.AccessKeyID,
			config.Lambda.SecretAccessKey,
			config.Lambda.StsAssumeRoleArn,
			config.Lambda.ResultFunction,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "create lambda result logger")
		}
	case "pubsub":
		result, err = NewPubSubLogWriter(
			config.PubSub.Project,
			config.PubSub.ResultTopic,
			config.PubSub.AddAttributes,
			logger,
		)
		if err != nil {
			return nil, errors.Wrap(err, "create pubsub result logger")
		}
	case "stdout":
		result, err = NewStdoutLogWriter()
		if err != nil {
			return nil, errors.Wrap(err, "create stdout result logger")
		}
	default:
		return nil, errors.Errorf(
			"unknown result log plugin: %s", config.Osquery.StatusLogPlugin,
		)
	}
	return &OsqueryLogger{Status: status, Result: result}, nil
}
