package service

import (
	"context"
	"runtime"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupURL(t *testing.T) {
	tests := []struct {
		in       string
		expected string
		name     string
	}{
		{"  http://foo.bar.com  ", "http://foo.bar.com", "leading and trailing whitespace"},
		{"\n http://foo.com \t", "http://foo.com", "whitespace"},
		{"http://foo.com", "http://foo.com", "noop"},
		{"http://foo.com/", "http://foo.com", "trailing slash"},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actual := cleanupURL(test.in)
			assert.Equal(tt, test.expected, actual)
		})
	}

}

func TestCreateAppConfig(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	var appConfigTests = []struct {
		configPayload fleet.AppConfig
	}{
		{
			configPayload: fleet.AppConfig{
				OrgInfo: fleet.OrgInfo{
					OrgLogoURL: "acme.co/images/logo.png",
					OrgName:    "Acme",
				},
				ServerSettings: fleet.ServerSettings{
					ServerURL:         "https://acme.co:8080/",
					LiveQueryDisabled: true,
				},
			},
		},
	}

	for _, tt := range appConfigTests {
		var result *fleet.AppConfig
		ds.NewAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) (*fleet.AppConfig, error) {
			result = config
			return config, nil
		}

		var gotSecrets []*fleet.EnrollSecret
		ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
			gotSecrets = secrets
			return nil
		}

		ctx := test.UserContext(test.UserAdmin)
		_, err := svc.NewAppConfig(ctx, tt.configPayload)
		require.Nil(t, err)

		payload := tt.configPayload
		assert.Equal(t, payload.OrgInfo.OrgLogoURL, result.OrgInfo.OrgLogoURL)
		assert.Equal(t, payload.OrgInfo.OrgName, result.OrgInfo.OrgName)
		assert.Equal(t, "https://acme.co:8080/", result.ServerSettings.ServerURL)
		assert.Equal(t, payload.ServerSettings.LiveQueryDisabled, result.ServerSettings.LiveQueryDisabled)

		// Ensure enroll secret was set
		require.NotNil(t, gotSecrets)
		require.Len(t, gotSecrets, 1)
		assert.Len(t, gotSecrets[0].Secret, 32)
	}
}

func TestEmptyEnrollSecret(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	err := svc.ApplyEnrollSecretSpec(
		test.UserContext(test.UserAdmin),
		&fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{}},
		},
	)
	require.Error(t, err)

	err = svc.ApplyEnrollSecretSpec(
		test.UserContext(test.UserAdmin),
		&fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: ""}}},
	)
	require.Error(t, err, "empty secret should be disallowed")

	err = svc.ApplyEnrollSecretSpec(
		test.UserContext(test.UserAdmin),
		&fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: "foo"}},
		},
	)
	require.NoError(t, err)
}

func TestService_LoggingConfig(t *testing.T) {
	logFile := "/dev/null"
	if runtime.GOOS == "windows" {
		logFile = "NUL"
	}

	fileSystemConfig := fleet.FilesystemConfig{FilesystemConfig: config.FilesystemConfig{
		StatusLogFile:        logFile,
		ResultLogFile:        logFile,
		EnableLogRotation:    false,
		EnableLogCompression: false,
	}}

	firehoseConfig := fleet.FirehoseConfig{
		Region:       testFirehosePluginConfig().Firehose.Region,
		StatusStream: testFirehosePluginConfig().Firehose.StatusStream,
		ResultStream: testFirehosePluginConfig().Firehose.ResultStream,
	}

	kinesisConfig := fleet.KinesisConfig{
		Region:       testKinesisPluginConfig().Kinesis.Region,
		StatusStream: testKinesisPluginConfig().Kinesis.StatusStream,
		ResultStream: testKinesisPluginConfig().Kinesis.ResultStream,
	}

	lambdaConfig := fleet.LambdaConfig{
		Region:         testLambdaPluginConfig().Lambda.Region,
		StatusFunction: testLambdaPluginConfig().Lambda.StatusFunction,
		ResultFunction: testLambdaPluginConfig().Lambda.ResultFunction,
	}

	pubsubConfig := fleet.PubSubConfig{
		PubSubConfig: config.PubSubConfig{
			Project:       testPubSubPluginConfig().PubSub.Project,
			StatusTopic:   testPubSubPluginConfig().PubSub.StatusTopic,
			ResultTopic:   testPubSubPluginConfig().PubSub.ResultTopic,
			AddAttributes: false,
		},
	}

	type fields struct {
		config config.FleetConfig
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *fleet.Logging
		wantErr bool
	}{
		{
			name:   "test default test config (aka filesystem)",
			fields: fields{config: config.TestConfig()},
			args:   args{ctx: test.UserContext(test.UserAdmin)},
			want: &fleet.Logging{
				Debug: true,
				Json:  false,
				Result: fleet.LoggingPlugin{
					Plugin: "filesystem",
					Config: fileSystemConfig,
				},
				Status: fleet.LoggingPlugin{
					Plugin: "filesystem",
					Config: fileSystemConfig,
				},
			},
		},
		{
			name:   "test firehose config",
			fields: fields{config: testFirehosePluginConfig()},
			args:   args{ctx: test.UserContext(test.UserAdmin)},
			want: &fleet.Logging{
				Debug: true,
				Json:  false,
				Result: fleet.LoggingPlugin{
					Plugin: "firehose",
					Config: firehoseConfig,
				},
				Status: fleet.LoggingPlugin{
					Plugin: "firehose",
					Config: firehoseConfig,
				},
			},
		},
		{
			name:   "test kinesis config",
			fields: fields{config: testKinesisPluginConfig()},
			args:   args{ctx: test.UserContext(test.UserAdmin)},
			want: &fleet.Logging{
				Debug: true,
				Json:  false,
				Result: fleet.LoggingPlugin{
					Plugin: "kinesis",
					Config: kinesisConfig,
				},
				Status: fleet.LoggingPlugin{
					Plugin: "kinesis",
					Config: kinesisConfig,
				},
			},
		},
		{
			name:   "test lambda config",
			fields: fields{config: testLambdaPluginConfig()},
			args:   args{ctx: test.UserContext(test.UserAdmin)},
			want: &fleet.Logging{
				Debug: true,
				Json:  false,
				Result: fleet.LoggingPlugin{
					Plugin: "lambda",
					Config: lambdaConfig,
				},
				Status: fleet.LoggingPlugin{
					Plugin: "lambda",
					Config: lambdaConfig,
				},
			},
		},
		{
			name:   "test pubsub config",
			fields: fields{config: testPubSubPluginConfig()},
			args:   args{ctx: test.UserContext(test.UserAdmin)},
			want: &fleet.Logging{
				Debug: true,
				Json:  false,
				Result: fleet.LoggingPlugin{
					Plugin: "pubsub",
					Config: pubsubConfig,
				},
				Status: fleet.LoggingPlugin{
					Plugin: "pubsub",
					Config: pubsubConfig,
				},
			},
		},
		{
			name:   "test stdout config",
			fields: fields{config: testStdoutPluginConfig()},
			args:   args{ctx: test.UserContext(test.UserAdmin)},
			want: &fleet.Logging{
				Debug: true,
				Json:  false,
				Result: fleet.LoggingPlugin{
					Plugin: "stdout",
					Config: nil,
				},
				Status: fleet.LoggingPlugin{
					Plugin: "stdout",
					Config: nil,
				},
			},
		},
		{
			name:    "test unrecognized config",
			fields:  fields{config: testUnrecognizedPluginConfig()},
			args:    args{ctx: test.UserContext(test.UserAdmin)},
			wantErr: true,
			want:    nil,
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := new(mock.Store)
			svc := newTestServiceWithConfig(t, ds, tt.fields.config, nil, nil)
			got, err := svc.LoggingConfig(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoggingConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.Equal(t, tt.want, got) {
				t.Errorf("LoggingConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModifyAppConfigPatches(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	storedConfig := &fleet.AppConfig{}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return storedConfig, nil
	}

	ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error {
		storedConfig = info
		return nil
	}

	configJSON := []byte(`{"org_info": { "org_name": "Acme", "org_logo_url": "somelogo.jpg" }}`)

	ctx := test.UserContext(test.UserAdmin)
	_, err := svc.ModifyAppConfig(ctx, configJSON)
	require.NoError(t, err)

	assert.Equal(t, "Acme", storedConfig.OrgInfo.OrgName)

	configJSON = []byte(`{"server_settings": { "server_url": "http://someurl" }}`)

	_, err = svc.ModifyAppConfig(ctx, configJSON)
	require.NoError(t, err)

	assert.Equal(t, "Acme", storedConfig.OrgInfo.OrgName)
	assert.Equal(t, "http://someurl", storedConfig.ServerSettings.ServerURL)
}
