package fleet

import (
	"github.com/fleetdm/fleet/v4/server/config"
	"reflect"
	"runtime"
	"testing"
)

func TestLoggingFromConfig(t *testing.T) {
	// default log file location depends on OS
	logFile := "/dev/null"
	if runtime.GOOS == "windows" {
		logFile = "NUL"
	}

	type args struct {
		conf config.FleetConfig
	}
	tests := []struct {
		name string
		args args
		want *Logging
	}{
		{
			name: "Test Config Serializes Properly",
			args: args{conf: config.TestConfig()},
			want: &Logging{
				Debug:           true,
				Json:            false,
				ResultLogPlugin: "filesystem",
				StatusLogPlugin: "filesystem",
				FileSystem: &FileSystemConfig{config.FilesystemConfig{
					StatusLogFile:        logFile,
					ResultLogFile:        logFile,
					EnableLogRotation:    false,
					EnableLogCompression: false,
				}},
				Firehose: nil,
				Kinesis:  nil,
				Lambda:   nil,
				PubSub:   nil,
			},
		},
		{
			name: "Kinesis Logging Plugin serializes Properly",
			args: args{conf: config.TestKinesisPluginConfig()},
			want: &Logging{
				Debug:           true,
				Json:            false,
				ResultLogPlugin: "kinesis",
				StatusLogPlugin: "kinesis",
				FileSystem:      nil,
				Firehose:        nil,
				Kinesis: &KinesisConfig{
					Region:       config.TestKinesisPluginConfig().Kinesis.Region,
					StatusStream: config.TestKinesisPluginConfig().Kinesis.StatusStream,
					ResultStream: config.TestKinesisPluginConfig().Kinesis.ResultStream,
				},
				Lambda: nil,
				PubSub: nil,
			},
		},
		{
			name: "Firehose Logging Plugin serializes Properly",
			args: args{conf: config.TestFirehosePluginConfig()},
			want: &Logging{
				Debug:           true,
				Json:            false,
				ResultLogPlugin: "firehose",
				StatusLogPlugin: "firehose",
				FileSystem:      nil,
				Firehose: &FirehoseConfig{
					Region:       config.TestFirehosePluginConfig().Firehose.Region,
					StatusStream: config.TestFirehosePluginConfig().Firehose.StatusStream,
					ResultStream: config.TestFirehosePluginConfig().Firehose.ResultStream,
				},
				Kinesis: nil,
				Lambda:  nil,
				PubSub:  nil,
			},
		},
		{
			name: "Lambda Logging Plugin serializes Properly",
			args: args{conf: config.TestLambdaPluginConfig()},
			want: &Logging{
				Debug:           true,
				Json:            false,
				ResultLogPlugin: "lambda",
				StatusLogPlugin: "lambda",
				FileSystem:      nil,
				Firehose:        nil,
				Kinesis:         nil,
				Lambda: &LambdaConfig{
					Region:         config.TestLambdaPluginConfig().Lambda.Region,
					StatusFunction: config.TestLambdaPluginConfig().Lambda.StatusFunction,
					ResultFunction: config.TestLambdaPluginConfig().Lambda.ResultFunction,
				},
				PubSub: nil,
			},
		},
		{
			name: "PubSub Logging Plugin serializes Properly",
			args: args{conf: config.TestPubSubPluginConfig()},
			want: &Logging{
				Debug:           true,
				Json:            false,
				ResultLogPlugin: "pubsub",
				StatusLogPlugin: "pubsub",
				FileSystem:      nil,
				Firehose:        nil,
				Kinesis:         nil,
				Lambda:          nil,
				PubSub: &PubSubConfig{config.PubSubConfig{
					Project:       config.TestPubSubPluginConfig().PubSub.Project,
					StatusTopic:   config.TestPubSubPluginConfig().PubSub.StatusTopic,
					ResultTopic:   config.TestPubSubPluginConfig().PubSub.ResultTopic,
					AddAttributes: config.TestPubSubPluginConfig().PubSub.AddAttributes,
				}},
			},
		},
		{
			name: "Stdout Logging Plugin serializes Properly",
			args: args{conf: config.FleetConfig{
				Osquery:    config.OsqueryConfig{ResultLogPlugin: "stdout", StatusLogPlugin: "stdout"},
			}},
			want: &Logging{
				Debug:           false,
				Json:            false,
				ResultLogPlugin: "stdout",
				StatusLogPlugin: "stdout",
				FileSystem:      nil,
				Firehose:        nil,
				Kinesis:         nil,
				Lambda:          nil,
				PubSub:          nil,
			},
		},
		{
			name: "Empty Logging Plugin serializes Properly",
			args: args{conf: config.FleetConfig{
				Osquery:    config.OsqueryConfig{ResultLogPlugin: "", StatusLogPlugin: ""},
			}},
			want: &Logging{
				Debug:           false,
				Json:            false,
				ResultLogPlugin: "filesystem",
				StatusLogPlugin: "filesystem",
				FileSystem:      &FileSystemConfig{config.FilesystemConfig{
					StatusLogFile:        "",
					ResultLogFile:        "",
					EnableLogRotation:    false,
					EnableLogCompression: false,
				}},
				Firehose:        nil,
				Kinesis:         nil,
				Lambda:          nil,
				PubSub:          nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LoggingFromConfig(tt.args.conf); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoggingFromConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
