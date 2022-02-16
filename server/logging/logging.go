// Package logging provides logger "plugins" for writing osquery status and
// result logs to various destinations.
package logging

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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
			return nil, fmt.Errorf("create filesystem status logger: %w", err)
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
			return nil, fmt.Errorf("create firehose status logger: %w", err)
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
			return nil, fmt.Errorf("create kinesis status logger: %w", err)
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
			return nil, fmt.Errorf("create lambda status logger: %w", err)
		}
	case "pubsub":
		status, err = NewPubSubLogWriter(
			config.PubSub.Project,
			config.PubSub.StatusTopic,
			false,
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf("create pubsub status logger: %w", err)
		}
	case "stdout":
		status, err = NewStdoutLogWriter()
		if err != nil {
			return nil, fmt.Errorf("create stdout status logger: %w", err)
		}
	case "kafkarest":
		status, err = NewKafkaRESTWriter(&KafkaRESTParams{
			KafkaProxyHost: config.KafkaREST.ProxyHost,
			KafkaTopic:     config.KafkaREST.StatusTopic,
			KafkaTimeout:   config.KafkaREST.Timeout,
		})
		if err != nil {
			return nil, fmt.Errorf("create kafka rest status logger: %w", err)
		}
	default:
		return nil, fmt.Errorf(
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
			return nil, fmt.Errorf("create filesystem result logger: %w", err)
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
			return nil, fmt.Errorf("create firehose result logger: %w", err)
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
			return nil, fmt.Errorf("create kinesis result logger: %w", err)
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
			return nil, fmt.Errorf("create lambda result logger: %w", err)
		}
	case "pubsub":
		result, err = NewPubSubLogWriter(
			config.PubSub.Project,
			config.PubSub.ResultTopic,
			config.PubSub.AddAttributes,
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf("create pubsub result logger: %w", err)
		}
	case "stdout":
		result, err = NewStdoutLogWriter()
		if err != nil {
			return nil, fmt.Errorf("create stdout result logger: %w", err)
		}
	case "kafkarest":
		result, err = NewKafkaRESTWriter(&KafkaRESTParams{
			KafkaProxyHost: config.KafkaREST.ProxyHost,
			KafkaTopic:     config.KafkaREST.ResultTopic,
			KafkaTimeout:   config.KafkaREST.Timeout,
		})
		if err != nil {
			return nil, fmt.Errorf("create kafka rest result logger: %w", err)
		}
	default:
		return nil, fmt.Errorf(
			"unknown result log plugin: %s", config.Osquery.StatusLogPlugin,
		)
	}
	return &OsqueryLogger{Status: status, Result: result}, nil
}
