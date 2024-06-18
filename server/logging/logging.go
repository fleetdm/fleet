// Package logging provides logger "plugins" for various destinations.
package logging

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type FilesystemConfig struct {
	LogFile string

	EnableLogRotation    bool
	EnableLogCompression bool
	MaxSize              int
	MaxAge               int
	MaxBackups           int
}

type FirehoseConfig struct {
	StreamName string

	Region           string
	EndpointURL      string
	AccessKeyID      string
	SecretAccessKey  string
	StsAssumeRoleArn string
	StsExternalID    string
}

type KinesisConfig struct {
	StreamName string

	Region           string
	EndpointURL      string
	AccessKeyID      string
	SecretAccessKey  string
	StsAssumeRoleArn string
	StsExternalID    string
}

type LambdaConfig struct {
	Function string

	Region           string
	AccessKeyID      string
	SecretAccessKey  string
	StsAssumeRoleArn string
	StsExternalID    string
}

type PubSubConfig struct {
	Topic string

	Project       string
	AddAttributes bool
}

type KafkaRESTConfig struct {
	Topic string

	ProxyHost        string
	ContentTypeValue string
	Timeout          int
}

type Config struct {
	Plugin string

	Filesystem FilesystemConfig
	Firehose   FirehoseConfig
	Kinesis    KinesisConfig
	Lambda     LambdaConfig
	PubSub     PubSubConfig
	KafkaREST  KafkaRESTConfig
}

func NewJSONLogger(name string, config Config, logger log.Logger) (fleet.JSONLogger, error) {
	switch config.Plugin {
	case "":
		// Allow "" to mean filesystem for backwards compatibility
		level.Info(logger).Log(
			"msg",
			fmt.Sprintf("plugin for %s not explicitly specified. Assuming 'filesystem'", name),
		)
		fallthrough
	case "filesystem":
		writer, err := NewFilesystemLogWriter(
			config.Filesystem.LogFile,
			logger,
			config.Filesystem.EnableLogRotation,
			config.Filesystem.EnableLogCompression,
			config.Filesystem.MaxSize,
			config.Filesystem.MaxAge,
			config.Filesystem.MaxBackups,
		)
		if err != nil {
			return nil, fmt.Errorf("create filesystem %s logger: %w", name, err)
		}
		return fleet.JSONLogger(writer), nil
	case "firehose":
		writer, err := NewFirehoseLogWriter(
			config.Firehose.Region,
			config.Firehose.EndpointURL,
			config.Firehose.AccessKeyID,
			config.Firehose.SecretAccessKey,
			config.Firehose.StsAssumeRoleArn,
			config.Firehose.StsExternalID,
			config.Firehose.StreamName,
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf("create firehose %s logger: %w", name, err)
		}
		return fleet.JSONLogger(writer), nil
	case "kinesis":
		writer, err := NewKinesisLogWriter(
			config.Kinesis.Region,
			config.Kinesis.EndpointURL,
			config.Kinesis.AccessKeyID,
			config.Kinesis.SecretAccessKey,
			config.Kinesis.StsAssumeRoleArn,
			config.Kinesis.StsExternalID,
			config.Kinesis.StreamName,
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf("create kinesis %s logger: %w", name, err)
		}
		return fleet.JSONLogger(writer), nil
	case "lambda":
		writer, err := NewLambdaLogWriter(
			config.Lambda.Region,
			config.Lambda.AccessKeyID,
			config.Lambda.SecretAccessKey,
			config.Lambda.StsAssumeRoleArn,
			config.Lambda.StsExternalID,
			config.Lambda.Function,
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf("create lambda %s logger: %w", name, err)
		}
		return fleet.JSONLogger(writer), nil
	case "pubsub":
		writer, err := NewPubSubLogWriter(
			config.PubSub.Project,
			config.PubSub.Topic,
			config.PubSub.AddAttributes,
			logger,
		)
		if err != nil {
			return nil, fmt.Errorf("create pubsub %s logger: %w", name, err)
		}
		return fleet.JSONLogger(writer), nil
	case "stdout":
		writer, err := NewStdoutLogWriter()
		if err != nil {
			return nil, fmt.Errorf("create stdout %s logger: %w", name, err)
		}
		return fleet.JSONLogger(writer), nil
	case "kafkarest":
		writer, err := NewKafkaRESTWriter(&KafkaRESTParams{
			KafkaProxyHost:        config.KafkaREST.ProxyHost,
			KafkaTopic:            config.KafkaREST.Topic,
			KafkaContentTypeValue: config.KafkaREST.ContentTypeValue,
			KafkaTimeout:          config.KafkaREST.Timeout,
		})
		if err != nil {
			return nil, fmt.Errorf("create kafka rest %s logger: %w", name, err)
		}
		return fleet.JSONLogger(writer), nil
	default:
		return nil, fmt.Errorf(
			"unknown %s log plugin: %s", name, config.Plugin,
		)
	}
}
