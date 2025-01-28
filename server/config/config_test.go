package config

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestConfigRoundtrip(t *testing.T) {
	// This test verifies that a config can be roundtripped through yaml.
	// Doing so ensures that config_dump will provide the correct config.
	// Newly added config values will automatically be tested in this
	// function because of the reflection on the config struct.

	// viper tries to load config from the environment too, clear it in case
	// any config values are set in the environment.
	os.Clearenv()

	cmd := &cobra.Command{}
	// Leaving this flag unset means that no attempt will be made to load
	// the config file
	cmd.PersistentFlags().StringP("config", "c", "", "Path to a configuration file")
	man := NewManager(cmd)

	// Use reflection magic to walk the config struct, setting unique
	// values to be verified on the roundtrip. Note that bools are always
	// set to true, which could false positive if the default value is
	// true.
	original := &FleetConfig{}
	v := reflect.ValueOf(original)
	for conf_index := 0; conf_index < v.Elem().NumField(); conf_index++ {
		conf_v := v.Elem().Field(conf_index)
		conf_t := conf_v.Type()
		for key_index := 0; key_index < conf_v.NumField(); key_index++ {
			// ignore unexported fields
			if !conf_t.Field(key_index).IsExported() {
				continue
			}

			key_v := conf_v.Field(key_index)
			switch key_v.Interface().(type) {
			case string:
				switch conf_v.Type().Field(key_index).Name {
				case "TLSProfile":
					// we have to explicitly set value for this key as it will only
					// accept intermediate or modern
					key_v.SetString(TLSProfileModern)
				case "EnableAsyncHostProcessing":
					// supports a bool or per-task config
					key_v.SetString("true")
				case "AsyncHostCollectInterval", "AsyncHostCollectLockTimeout":
					// supports a duration or per-task config
					key_v.SetString("30s")
				// These are deprecated field names in the S3 config. Set them to zero value, which leads to the new fields being populated instead.
				case "Bucket", "Prefix", "Region", "EndpointURL", "AccessKeyID", "SecretAccessKey", "StsAssumeRoleArn", "StsExternalID":
					key_v.SetString("")
				default:
					key_v.SetString(v.Elem().Type().Field(conf_index).Name + "_" + conf_v.Type().Field(key_index).Name)
				}
			case int:
				key_v.SetInt(int64(conf_index*100 + key_index))
			case bool:
				switch conf_v.Type().Field(key_index).Name {
				// These are deprecated field names in the S3 config. Set them to zero value, which leads to the new fields being populated instead.
				case "DisableSSL", "ForceS3PathStyle":
					key_v.SetBool(false)
				default:
					key_v.SetBool(true)
				}
			case time.Duration:
				d := time.Duration(conf_index*100 + key_index)
				key_v.Set(reflect.ValueOf(d))
			}
		}
	}

	// Marshal the generated config
	buf, err := yaml.Marshal(original)
	require.NoError(t, err)
	t.Log(string(buf))

	// Manually load the serialized config
	man.viper.SetConfigType("yaml")
	err = man.viper.ReadConfig(bytes.NewReader(buf))
	require.Nil(t, err)

	// Ensure the read config is the same as the original
	actual := man.LoadConfig()
	assert.Equal(t, *original, actual)
}

func TestConfigOsqueryAsync(t *testing.T) {
	cases := []struct {
		desc         string
		yaml         string
		envVars      []string
		panics       bool
		wantLabelCfg AsyncProcessingConfig
	}{
		{
			desc: "default",
			wantLabelCfg: AsyncProcessingConfig{
				Enabled:                 false,
				CollectInterval:         30 * time.Second,
				CollectMaxJitterPercent: 10,
				CollectLockTimeout:      1 * time.Minute,
				CollectLogStatsInterval: 1 * time.Minute,
				InsertBatch:             2000,
				DeleteBatch:             2000,
				UpdateBatch:             1000,
				RedisPopCount:           1000,
				RedisScanKeysCount:      1000,
			},
		},
		{
			desc: "yaml set enabled true",
			yaml: `
osquery:
  enable_async_host_processing: true`,
			wantLabelCfg: AsyncProcessingConfig{
				Enabled:                 true,
				CollectInterval:         30 * time.Second,
				CollectMaxJitterPercent: 10,
				CollectLockTimeout:      1 * time.Minute,
				CollectLogStatsInterval: 1 * time.Minute,
				InsertBatch:             2000,
				DeleteBatch:             2000,
				UpdateBatch:             1000,
				RedisPopCount:           1000,
				RedisScanKeysCount:      1000,
			},
		},
		{
			desc: "yaml set enabled yes",
			yaml: `
osquery:
  enable_async_host_processing: yes`,
			wantLabelCfg: AsyncProcessingConfig{
				Enabled:                 true,
				CollectInterval:         30 * time.Second,
				CollectMaxJitterPercent: 10,
				CollectLockTimeout:      1 * time.Minute,
				CollectLogStatsInterval: 1 * time.Minute,
				InsertBatch:             2000,
				DeleteBatch:             2000,
				UpdateBatch:             1000,
				RedisPopCount:           1000,
				RedisScanKeysCount:      1000,
			},
		},
		{
			desc: "yaml set enabled on",
			yaml: `
osquery:
  enable_async_host_processing: on`,
			wantLabelCfg: AsyncProcessingConfig{
				Enabled:                 true,
				CollectInterval:         30 * time.Second,
				CollectMaxJitterPercent: 10,
				CollectLockTimeout:      1 * time.Minute,
				CollectLogStatsInterval: 1 * time.Minute,
				InsertBatch:             2000,
				DeleteBatch:             2000,
				UpdateBatch:             1000,
				RedisPopCount:           1000,
				RedisScanKeysCount:      1000,
			},
		},
		{
			desc: "yaml set enabled invalid",
			yaml: `
osquery:
  enable_async_host_processing: nope`,
			panics: true,
		},
		{
			desc: "yaml set enabled per-task",
			yaml: `
osquery:
  enable_async_host_processing: label_membership=true&policy_membership=false`,
			wantLabelCfg: AsyncProcessingConfig{
				Enabled:                 true,
				CollectInterval:         30 * time.Second,
				CollectMaxJitterPercent: 10,
				CollectLockTimeout:      1 * time.Minute,
				CollectLogStatsInterval: 1 * time.Minute,
				InsertBatch:             2000,
				DeleteBatch:             2000,
				UpdateBatch:             1000,
				RedisPopCount:           1000,
				RedisScanKeysCount:      1000,
			},
		},
		{
			desc: "yaml set invalid per-task",
			yaml: `
osquery:
  enable_async_host_processing: label_membership=nope&policy_membership=false`,
			panics: true,
		},
		{
			desc:    "envvar set enabled",
			envVars: []string{"FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING=true"},
			wantLabelCfg: AsyncProcessingConfig{
				Enabled:                 true,
				CollectInterval:         30 * time.Second,
				CollectMaxJitterPercent: 10,
				CollectLockTimeout:      1 * time.Minute,
				CollectLogStatsInterval: 1 * time.Minute,
				InsertBatch:             2000,
				DeleteBatch:             2000,
				UpdateBatch:             1000,
				RedisPopCount:           1000,
				RedisScanKeysCount:      1000,
			},
		},
		{
			desc:    "envvar set enabled on",
			envVars: []string{"FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING=on"}, // on/off, yes/no is only valid in yaml
			panics:  true,
		},
		{
			desc:    "envvar set enabled per task",
			envVars: []string{"FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING=policy_membership=false&label_membership=true"},
			wantLabelCfg: AsyncProcessingConfig{
				Enabled:                 true,
				CollectInterval:         30 * time.Second,
				CollectMaxJitterPercent: 10,
				CollectLockTimeout:      1 * time.Minute,
				CollectLogStatsInterval: 1 * time.Minute,
				InsertBatch:             2000,
				DeleteBatch:             2000,
				UpdateBatch:             1000,
				RedisPopCount:           1000,
				RedisScanKeysCount:      1000,
			},
		},
		{
			desc: "yaml collect interval lock timeout",
			yaml: `
osquery:
  enable_async_host_processing: true
  async_host_collect_interval: 10s
  async_host_collect_lock_timeout: 20s`,
			wantLabelCfg: AsyncProcessingConfig{
				Enabled:                 true,
				CollectInterval:         10 * time.Second,
				CollectMaxJitterPercent: 10,
				CollectLockTimeout:      20 * time.Second,
				CollectLogStatsInterval: 1 * time.Minute,
				InsertBatch:             2000,
				DeleteBatch:             2000,
				UpdateBatch:             1000,
				RedisPopCount:           1000,
				RedisScanKeysCount:      1000,
			},
		},
		{
			desc: "yaml collect interval lock timeout per task",
			yaml: `
osquery:
  enable_async_host_processing: true
  async_host_collect_interval: label_membership=10s
  async_host_collect_lock_timeout: policy_membership=20s`,
			wantLabelCfg: AsyncProcessingConfig{
				Enabled:                 true,
				CollectInterval:         10 * time.Second,
				CollectMaxJitterPercent: 10,
				CollectLockTimeout:      1 * time.Minute,
				CollectLogStatsInterval: 1 * time.Minute,
				InsertBatch:             2000,
				DeleteBatch:             2000,
				UpdateBatch:             1000,
				RedisPopCount:           1000,
				RedisScanKeysCount:      1000,
			},
		},
		{
			desc: "yaml env var override",
			yaml: `
osquery:
  enable_async_host_processing: false
  async_host_collect_interval: label_membership=10s
  async_host_collect_lock_timeout: policy_membership=20s
  async_host_insert_batch: 10`,
			envVars: []string{
				"FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING=policy_membership=false&label_membership=true",
				"FLEET_OSQUERY_ASYNC_HOST_COLLECT_INTERVAL=policy_membership=30s&label_membership=50s",
				"FLEET_OSQUERY_ASYNC_HOST_COLLECT_LOCK_TIMEOUT=40s",
			},
			wantLabelCfg: AsyncProcessingConfig{
				Enabled:                 true,
				CollectInterval:         50 * time.Second,
				CollectMaxJitterPercent: 10,
				CollectLockTimeout:      40 * time.Second,
				CollectLogStatsInterval: 1 * time.Minute,
				InsertBatch:             10,
				DeleteBatch:             2000,
				UpdateBatch:             1000,
				RedisPopCount:           1000,
				RedisScanKeysCount:      1000,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var cmd cobra.Command
			// Leaving this flag unset means that no attempt will be made to load
			// the config file
			cmd.PersistentFlags().StringP("config", "c", "", "Path to a configuration file")
			man := NewManager(&cmd)

			// load the yaml config
			man.viper.SetConfigType("yaml")
			require.NoError(t, man.viper.ReadConfig(strings.NewReader(c.yaml)))

			// TODO: tried to test command-line flags too by using cmd.SetArgs to
			// test-case values, but that didn't seem to work, not sure how it can
			// be done in our particular setup.

			// set the environment variables
			os.Clearenv()
			for _, env := range c.envVars {
				kv := strings.SplitN(env, "=", 2)
				t.Setenv(kv[0], kv[1])
			}

			var loadedCfg FleetConfig
			if c.panics {
				require.Panics(t, func() {
					loadedCfg = man.LoadConfig()
				})
			} else {
				require.NotPanics(t, func() {
					loadedCfg = man.LoadConfig()
				})
				got := loadedCfg.Osquery.AsyncConfigForTask(AsyncTaskLabelMembership)
				require.Equal(t, c.wantLabelCfg, got)
			}
		})
	}
}

func TestToTLSConfig(t *testing.T) {
	dir := t.TempDir()
	caFile, certFile, keyFile, garbageFile := filepath.Join(dir, "ca"),
		filepath.Join(dir, "cert"),
		filepath.Join(dir, "key"),
		filepath.Join(dir, "garbage")
	require.NoError(t, os.WriteFile(caFile, testCA, 0o600))
	require.NoError(t, os.WriteFile(certFile, testCert, 0o600))
	require.NoError(t, os.WriteFile(keyFile, testKey, 0o600))
	require.NoError(t, os.WriteFile(garbageFile, []byte("zzzz"), 0o600))

	cases := []struct {
		name        string
		in          TLS
		errContains string
	}{
		{"zero", TLS{}, ""},
		{"invalid file", TLS{TLSCA: "/no/such/file"}, "no such file"},
		{"CA", TLS{TLSCA: caFile}, ""},
		{"invalid CA content", TLS{TLSCA: garbageFile}, "failed to append PEM"},
		{"CA invalid cert", TLS{TLSCA: caFile, TLSCert: "/no/such/file"}, "no such file"},
		{"CA invalid key", TLS{TLSCA: caFile, TLSCert: certFile, TLSKey: "/no/such/file"}, "no such file"},
		{"CA cert key", TLS{TLSCA: caFile, TLSCert: certFile, TLSKey: keyFile}, ""},
		{"CA invalid cert content", TLS{TLSCA: caFile, TLSCert: garbageFile, TLSKey: keyFile}, "failed to find any PEM data"},
		{"CA invalid key content", TLS{TLSCA: caFile, TLSCert: certFile, TLSKey: garbageFile}, "failed to find any PEM data"},
		{"CA cert key server", TLS{TLSCA: caFile, TLSCert: certFile, TLSKey: keyFile, TLSServerName: "abc"}, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.in.ToTLSConfig()
			if c.errContains != "" {
				require.Error(t, err)
				require.Nil(t, got)
				require.Contains(t, err.Error(), c.errContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			// root ca is required if TLSCA is set
			if c.in.TLSCA != "" {
				require.NotNil(t, got.RootCAs)
			} else {
				require.Nil(t, got.RootCAs)
			}
			require.Equal(t, got.ServerName, c.in.TLSServerName)
			if c.in.TLSCert != "" {
				require.Len(t, got.Certificates, 1)
			} else {
				require.Nil(t, got.Certificates)
			}
		})
	}
}

func TestAppleAPNSSCEPConfig(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, garbageFile, invalidKeyFile := filepath.Join(dir, "cert"),
		filepath.Join(dir, "key"),
		filepath.Join(dir, "garbage"),
		filepath.Join(dir, "invalid_key")
	require.NoError(t, os.WriteFile(certFile, testCert, 0o600))
	require.NoError(t, os.WriteFile(keyFile, testKey, 0o600))
	require.NoError(t, os.WriteFile(garbageFile, []byte("zzzz"), 0o600))
	require.NoError(t, os.WriteFile(invalidKeyFile, unrelatedTestKey, 0o600))

	cases := []struct {
		name       string
		in         MDMConfig
		errMatches string
	}{
		{"missing cert", MDMConfig{AppleAPNsKey: keyFile, AppleSCEPKey: keyFile}, `Apple MDM (APNs|SCEP) configuration: no certificate provided`},
		{"missing key", MDMConfig{AppleAPNsCert: certFile, AppleSCEPCert: certFile}, "Apple MDM (APNs|SCEP) configuration: no key provided"},
		{"missing cert with raw key", MDMConfig{AppleAPNsKeyBytes: string(testKey), AppleSCEPKeyBytes: string(testKey)}, `Apple MDM (APNs|SCEP) configuration: no certificate provided`},
		{"missing key with raw cert", MDMConfig{AppleAPNsCertBytes: string(testCert), AppleSCEPCertBytes: string(testCert)}, "Apple MDM (APNs|SCEP) configuration: no key provided"},
		{"cert file does not exist", MDMConfig{AppleAPNsCert: "no-such-file", AppleAPNsKey: keyFile, AppleSCEPCert: "no-such-file", AppleSCEPKey: keyFile}, `open no-such-file: no such file or directory`},
		{"key file does not exist", MDMConfig{AppleAPNsKey: "no-such-file", AppleAPNsCert: certFile, AppleSCEPKey: "no-such-file", AppleSCEPCert: certFile}, `open no-such-file: no such file or directory`},
		{"valid file pairs", MDMConfig{AppleAPNsCert: certFile, AppleAPNsKey: keyFile, AppleSCEPCert: certFile, AppleSCEPKey: keyFile}, ""},
		{"valid file/raw pairs", MDMConfig{AppleAPNsCert: certFile, AppleAPNsKeyBytes: string(testKey), AppleSCEPCert: certFile, AppleSCEPKeyBytes: string(testKey)}, ""},
		{"valid raw/file pairs", MDMConfig{AppleAPNsCertBytes: string(testCert), AppleAPNsKey: keyFile, AppleSCEPCertBytes: string(testCert), AppleSCEPKey: keyFile}, ""},
		{"invalid file pairs", MDMConfig{AppleAPNsCert: certFile, AppleAPNsKey: invalidKeyFile, AppleSCEPCert: certFile, AppleSCEPKey: invalidKeyFile}, "tls: private key does not match public key"},
		{"invalid file/raw pairs", MDMConfig{AppleAPNsCert: certFile, AppleAPNsKeyBytes: string(unrelatedTestKey), AppleSCEPCert: certFile, AppleSCEPKeyBytes: string(unrelatedTestKey)}, "tls: private key does not match public key"},
		{"invalid raw/file pairs", MDMConfig{AppleAPNsCertBytes: string(testCert), AppleAPNsKey: invalidKeyFile, AppleSCEPCertBytes: string(testCert), AppleSCEPKey: invalidKeyFile}, "tls: private key does not match public key"},
		{"invalid file key", MDMConfig{AppleAPNsCert: certFile, AppleAPNsKey: garbageFile, AppleSCEPCert: certFile, AppleSCEPKey: garbageFile}, "tls: failed to find any PEM data"},
		{"invalid raw key", MDMConfig{AppleAPNsCert: certFile, AppleAPNsKeyBytes: "zzzz", AppleSCEPCert: certFile, AppleSCEPKeyBytes: "zzzz"}, "tls: failed to find any PEM data"},
		{"invalid raw cert", MDMConfig{AppleAPNsCertBytes: "zzzz", AppleAPNsKey: keyFile, AppleSCEPCertBytes: "zzzz", AppleSCEPKey: keyFile}, "tls: failed to find any PEM data in certificate input"},
		{"duplicate cert", MDMConfig{AppleAPNsCert: certFile, AppleAPNsCertBytes: string(testCert), AppleAPNsKey: keyFile, AppleSCEPCert: certFile, AppleSCEPCertBytes: string(testCert), AppleSCEPKey: keyFile}, `Apple MDM (APNs|SCEP) configuration: only one of the certificate path or bytes must be provided`},
		{"duplicate key", MDMConfig{AppleAPNsCert: certFile, AppleAPNsKey: keyFile, AppleAPNsKeyBytes: string(testKey), AppleSCEPCert: certFile, AppleSCEPKey: keyFile, AppleSCEPKeyBytes: string(testKey)}, `Apple MDM (APNs|SCEP) configuration: only one of the key path or bytes must be provided`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.in.AppleAPNsCert != "" || c.in.AppleAPNsCertBytes != "" || c.in.AppleAPNsKey != "" || c.in.AppleAPNsKeyBytes != "" {
				got, pemCert, pemKey, err := c.in.AppleAPNs()
				if c.errMatches != "" {
					require.Error(t, err)
					require.Nil(t, got)
					require.Regexp(t, c.errMatches, err.Error())
				} else {
					require.NoError(t, err)
					require.NotNil(t, got)
					require.NotNil(t, got.Leaf) // APNs cert is parsed and stored
					require.NotEmpty(t, pemCert)
					require.NotEmpty(t, pemKey)
				}
			}

			if c.in.AppleSCEPCert != "" || c.in.AppleSCEPCertBytes != "" || c.in.AppleSCEPKey != "" || c.in.AppleSCEPKeyBytes != "" {
				got, pemCert, pemKey, err := c.in.AppleSCEP()
				if c.errMatches != "" {
					require.Error(t, err)
					require.Nil(t, got)
					require.Regexp(t, c.errMatches, err.Error())
				} else {
					require.NoError(t, err)
					require.NotNil(t, got)
					require.NotNil(t, got.Leaf) // SCEP cert is not kept, not needed
					require.NotEmpty(t, pemCert)
					require.NotEmpty(t, pemKey)
				}
			}
		})
	}
}

func TestAppleBMConfig(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, garbageFile, invalidKeyFile := filepath.Join(dir, "cert"),
		filepath.Join(dir, "key"),
		filepath.Join(dir, "garbage"),
		filepath.Join(dir, "invalid_key")
	require.NoError(t, os.WriteFile(certFile, testCert, 0o600))
	require.NoError(t, os.WriteFile(keyFile, testKey, 0o600))
	require.NoError(t, os.WriteFile(garbageFile, []byte("zzzz"), 0o600))
	require.NoError(t, os.WriteFile(invalidKeyFile, unrelatedTestKey, 0o600))

	cases := []struct {
		name       string
		in         MDMConfig
		errMatches string
	}{
		{"missing cert", MDMConfig{AppleBMKey: keyFile, AppleBMServerToken: garbageFile}, `Apple BM configuration: no certificate provided`},
		{"missing key", MDMConfig{AppleBMCert: certFile, AppleBMServerToken: garbageFile}, "Apple BM configuration: no key provided"},
		{"missing cert with raw key", MDMConfig{AppleBMKeyBytes: string(testKey), AppleBMServerToken: garbageFile}, `Apple BM configuration: no certificate provided`},
		{"missing key with raw cert", MDMConfig{AppleBMCertBytes: string(testCert), AppleBMServerToken: garbageFile}, "Apple BM configuration: no key provided"},
		{"cert file does not exist", MDMConfig{AppleBMCert: "no-such-file", AppleBMKey: keyFile, AppleBMServerToken: garbageFile}, `open no-such-file: no such file or directory`},
		{"key file does not exist", MDMConfig{AppleBMKey: "no-such-file", AppleBMCert: certFile, AppleBMServerToken: garbageFile}, `open no-such-file: no such file or directory`},
		{"invalid file pairs", MDMConfig{AppleBMCert: certFile, AppleBMKey: invalidKeyFile, AppleBMServerToken: garbageFile}, "tls: private key does not match public key"},
		{"invalid file/raw pairs", MDMConfig{AppleBMCert: certFile, AppleBMKeyBytes: string(unrelatedTestKey), AppleBMServerToken: garbageFile}, "tls: private key does not match public key"},
		{"invalid raw/file pairs", MDMConfig{AppleBMCertBytes: string(testCert), AppleBMKey: invalidKeyFile, AppleBMServerToken: garbageFile}, "tls: private key does not match public key"},
		{"invalid file key", MDMConfig{AppleBMCert: certFile, AppleBMKey: garbageFile, AppleBMServerToken: garbageFile}, "tls: failed to find any PEM data"},
		{"invalid raw key", MDMConfig{AppleBMCert: certFile, AppleBMKeyBytes: "zzzz", AppleBMServerToken: garbageFile}, "tls: failed to find any PEM data"},
		{"invalid raw cert", MDMConfig{AppleBMCertBytes: "zzzz", AppleBMKey: keyFile, AppleBMServerToken: garbageFile}, "tls: failed to find any PEM data in certificate input"},
		{"duplicate cert", MDMConfig{AppleBMCert: certFile, AppleBMCertBytes: string(testCert), AppleBMKey: keyFile, AppleBMServerToken: garbageFile}, `Apple BM configuration: only one of the certificate path or bytes must be provided`},
		{"duplicate key", MDMConfig{AppleBMCert: certFile, AppleBMKey: keyFile, AppleBMKeyBytes: string(testKey), AppleBMServerToken: garbageFile}, `Apple BM configuration: only one of the key path or bytes must be provided`},
		{"token file does not exist", MDMConfig{AppleBMCert: certFile, AppleBMKey: keyFile, AppleBMServerToken: "no-such-file"}, `Apple BM configuration: reading token file: open no-such-file: no such file or directory`},
		{"invalid token file", MDMConfig{AppleBMCert: certFile, AppleBMKey: keyFile, AppleBMServerToken: garbageFile}, `Apple BM configuration: decrypt token: malformed MIME header: missing colon: "zzzz"`},
		{"invalid raw token file", MDMConfig{AppleBMCert: certFile, AppleBMKey: keyFile, AppleBMServerTokenBytes: "zzzz"}, `Apple BM configuration: decrypt token: malformed MIME header: missing colon: "zzzz"`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := c.in.AppleBM()
			require.Error(t, err)
			require.Regexp(t, c.errMatches, err.Error())
		})
	}
}

func TestMicrosoftWSTEPConfig(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile, garbageFile, invalidKeyFile := filepath.Join(dir, "cert"),
		filepath.Join(dir, "key"),
		filepath.Join(dir, "garbage"),
		filepath.Join(dir, "invalid_key")
	require.NoError(t, os.WriteFile(certFile, testCert, 0o600))
	require.NoError(t, os.WriteFile(keyFile, testKey, 0o600))
	require.NoError(t, os.WriteFile(garbageFile, []byte("zzzz"), 0o600))
	require.NoError(t, os.WriteFile(invalidKeyFile, unrelatedTestKey, 0o600))

	cases := []struct {
		name       string
		in         MDMConfig
		errMatches string
	}{
		{"missing cert", MDMConfig{WindowsWSTEPIdentityKey: keyFile}, `Microsoft MDM WSTEP configuration: no certificate provided`},
		{"missing key", MDMConfig{WindowsWSTEPIdentityCert: certFile}, "Microsoft MDM WSTEP configuration: no key provided"},
		{"cert file does not exist", MDMConfig{WindowsWSTEPIdentityCert: "no-such-file", WindowsWSTEPIdentityKey: keyFile}, `open no-such-file: no such file or directory`},
		{"key file does not exist", MDMConfig{WindowsWSTEPIdentityKey: "no-such-file", WindowsWSTEPIdentityCert: certFile}, `open no-such-file: no such file or directory`},
		{"valid file pairs", MDMConfig{WindowsWSTEPIdentityCert: certFile, WindowsWSTEPIdentityKey: keyFile}, ""},
		{"valid file/raw pairs", MDMConfig{WindowsWSTEPIdentityCert: certFile, WindowsWSTEPIdentityKeyBytes: string(testKey)}, ""},
		{"valid raw/file pairs", MDMConfig{WindowsWSTEPIdentityCertBytes: string(testCert), WindowsWSTEPIdentityKey: keyFile}, ""},
		{"invalid file pairs", MDMConfig{WindowsWSTEPIdentityCert: certFile, WindowsWSTEPIdentityKey: invalidKeyFile}, "tls: private key does not match public key"},
		{"invalid file key", MDMConfig{WindowsWSTEPIdentityCert: certFile, WindowsWSTEPIdentityKey: garbageFile}, "tls: failed to find any PEM data"},
		{"invalid file cert", MDMConfig{WindowsWSTEPIdentityCert: garbageFile, WindowsWSTEPIdentityKey: keyFile}, "tls: failed to find any PEM data"},
		{"invalid file/raw pairs", MDMConfig{WindowsWSTEPIdentityCert: certFile, WindowsWSTEPIdentityKeyBytes: string(unrelatedTestKey)}, "tls: private key does not match public key"},
		{"invalid raw/file pairs", MDMConfig{WindowsWSTEPIdentityCertBytes: string(testCert), WindowsWSTEPIdentityKey: invalidKeyFile}, "tls: private key does not match public key"},
		{"invalid file key", MDMConfig{WindowsWSTEPIdentityCert: certFile, WindowsWSTEPIdentityKey: garbageFile}, "tls: failed to find any PEM data"},
		{"invalid raw key", MDMConfig{WindowsWSTEPIdentityCert: certFile, WindowsWSTEPIdentityKeyBytes: "zzzz"}, "tls: failed to find any PEM data"},
		{"invalid raw cert", MDMConfig{WindowsWSTEPIdentityCertBytes: "zzzz", WindowsWSTEPIdentityKey: keyFile}, "tls: failed to find any PEM data in certificate input"},
		{"duplicate cert", MDMConfig{WindowsWSTEPIdentityCert: certFile, WindowsWSTEPIdentityCertBytes: string(testCert), WindowsWSTEPIdentityKey: keyFile}, `Microsoft MDM WSTEP configuration: only one of the certificate path or bytes must be provided`},
		{"duplicate key", MDMConfig{WindowsWSTEPIdentityCert: certFile, WindowsWSTEPIdentityKey: keyFile, WindowsWSTEPIdentityKeyBytes: string(testKey)}, `Microsoft MDM WSTEP configuration: only one of the key path or bytes must be provided`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.in.WindowsWSTEPIdentityCert != "" || c.in.WindowsWSTEPIdentityKey != "" {
				got, pemCert, pemKey, err := c.in.MicrosoftWSTEP()
				if c.errMatches != "" {
					require.Error(t, err)
					require.Nil(t, got)
					require.Regexp(t, c.errMatches, err.Error())
				} else {
					require.NoError(t, err)
					require.NotNil(t, got)
					require.NotNil(t, got.Leaf) // TODO: confirm cert is not kept, not needed?
					require.NotEmpty(t, pemCert)
					require.NotEmpty(t, pemKey)
				}
			}
		})
	}
}

var (
	testCA = []byte(`-----BEGIN CERTIFICATE-----
MIIFSzCCAzOgAwIBAgIUf4lOcb9bkN2+u6FjWL0fSFCjGGgwDQYJKoZIhvcNAQEL
BQAwNTETMBEGA1UECgwKUmVkaXMgVGVzdDEeMBwGA1UEAwwVQ2VydGlmaWNhdGUg
QXV0aG9yaXR5MB4XDTIxMTAxOTEyNTEwNloXDTMxMTAxNzEyNTEwNlowNTETMBEG
A1UECgwKUmVkaXMgVGVzdDEeMBwGA1UEAwwVQ2VydGlmaWNhdGUgQXV0aG9yaXR5
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA02LNfNKjI/PwV4F2CVix
vVfFN41yxMKYkapTrvC1nc7lVmG5oxxgOIUpFT+7xj0+h2bBqR+t3eiFiaudz3Yc
9eG2J7BTtMST9QmQtNEyeC17TZxf4XB2EA68dYC24XaHBnSFsPg8/axlIVi1Hz7b
QmDRNY/X3cc3nzGxuuk3NnSN7s1UlKnZ1v0YZGwWhYD3iAv7kQcI3WYF0TF0nc2a
OXb68/AOghq9Z9zLk1ULIfTmT0fcJRsFssWClF7E378PSk0qjB6NEKADVyWq3d2g
8ValKmbKvAacsGxb2EXAPCJsBil0Sv7jAsl1hVfMCBwj6LfPKvn7/K8vbKz7Gtrw
COWVJtzaBrKzpjOTXQp9RnuqlDUZackTmn9hlCMLgapEC+j7PNvS8cyAbOz9bpEk
wdF/wrvUVsJc74+MXzEK7DWBKD2lP9nvY+0DrYJ/55KH1wbIH1RncLm6s6M4Zc9L
YfaeTuklimAOlx8WvuYQUJpxTh6gT4xWqZG2p8IcjxVp2Sl7eYtlaE/u7Ixc+Bfd
QpTaBXrtcQzttPNiSZM8b+nNL05p+LxtSVAYUu1Yc0hWBHJBb/dkDibOU3Mi8Aio
bvpsBp1RLfXSrRMOpXS3w4G1THrhC4IC1KkUbZ8EQaBlwa7mlwV8hxZOjJQ7Mf4D
Z8WEh1j/XH/zlKVJon2aUWUCAwEAAaNTMFEwHQYDVR0OBBYEFIDJJVTvQCl1vMIi
246T25FZVBsWMB8GA1UdIwQYMBaAFIDJJVTvQCl1vMIi246T25FZVBsWMA8GA1Ud
EwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggIBAGZqxsaleZqmljrqrpL5JxoQ
G9/9tvfw5WYqeJ6r8s86HfxaqsEUemzSBb7HFJS42Ik6ghd32d62wp7xLxtQY8As
jvU9YZ2s42tSWgxch8kY/kgCjwsqTFViWmyxmc05TxulRr8BonIo8YAU6/5kBam+
sV5nfbBse5i9+nQqmjzVI7lVp7lIk+T9T4UsdH/mtbWv8cJjCBzbyObU+V9kjTSQ
O+cshOn59IMRvAkySKIHvm7keO4skazo2RMjdME9KW/ydc7iQ9YC0+MiDQF+eIAP
a/SGdTD8W/WNXT1rtD4DyTEZK1modAI7KukkrTwlaTW0GwssLq5TpwzQKK5W/ANZ
SU44yILArQrWZgXXxBBfGAH/asd4JgIxal/iM0hlYh6WYdSUa/QzJFFRngtE52jL
M1sTsUgXjItspH79oUD+my4ioDv6r2CAnlxl2MvqGzfBgItb5yq3bBwxNe/qOzWR
PbKbp3UvlzMbbpbeJHO2NHnu7Hha9mV3yr9+lsTv2SFeKGqFRbC7v+9kSDu6eOyC
lnARbzReZyZiYr9vCTxH76wCyUBBg7p59ZriBw0yaXvXcr4cO8IUPx4aPe9nHkbC
8G/rnKycuGGIDjslRTOJodxf2ud2UPYUTZDBi1QoV4+jzWKUjUxuHuN2WIwxnXKB
cJap0OI7VFpOjIJLzXRQ
-----END CERTIFICATE-----`)

	testCert = []byte(`-----BEGIN CERTIFICATE-----
MIID6DCCAdACFGX99Sw4aF2qKGLucoIWQRAXHrs1MA0GCSqGSIb3DQEBCwUAMDUx
EzARBgNVBAoMClJlZGlzIFRlc3QxHjAcBgNVBAMMFUNlcnRpZmljYXRlIEF1dGhv
cml0eTAeFw0yMTEwMTkxNzM0MzlaFw0yMjEwMTkxNzM0MzlaMCwxEzARBgNVBAoM
ClJlZGlzIFRlc3QxFTATBgNVBAMMDEdlbmVyaWMtY2VydDCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAKSHcH8EjSvp3Nm4IHAFxG9DZm8+0h1BwU0OX0VH
cJ+Cf+f6h0XYMcMo9LFEpnUJRRMjKrM4mkI75NIIufNBN+GrtqqTPTid8wfOGu/U
fa5EEU1hb2j7AiMlpM6i0+ZysXSNo+Vc/cNZT0PXfyOtJnYm6p9WZM84ID1t2ea0
bLwC12cTKv5oybVGtJHh76TRxAR3FeQ9+SY30vUAxYm6oWyYho8rRdKtUSe11pXj
6OhxxfTZnsSWn4lo0uBpXai63XtieTVpz74htSNC1bunIGv7//m5F60sH5MrF5JS
kPxfCfgqski84ICDSRNlvpT+eMPiygAAJ8zY8wYUXRYFYTUCAwEAATANBgkqhkiG
9w0BAQsFAAOCAgEAAAw+6Uz2bAcXgQ7fQfdOm+T6FLRBcr8PD4ajOvSu/T+HhVVj
E26Qt2IBwFEYve2FvDxrBCF8aQYZcyQqnP8bdKebnWAaqL8BbTwLWW+fDuZLO2b4
QHjAEdEKKdZC5/FRpQrkerf5CCPTHE+5M17OZg41wdVYnCEwJOkP5pUAVsmwtrSw
VeIquy20TZO0qbscDQETf7NIJgW0IXg82wBe53Rv4/wL3Ybq13XVRGYiJrwpaNTf
UNgsDWqgwlQ5L2GOLDgg8S2NoF9mWVgCGSp3a2eHW+EmBRQ1OP6EYQtIhKdGLrSn
dAOMJ2ER1pgHWUFKkWQaZ9i37Dx2j7P5c4/XNeVozcRQcLwKwN+n8k+bwIYcTX0H
MOVFYm+WiFi/gjI860Tx853Sc0nkpOXmBCeHSXigGUscgjBYbmJz4iExXuwgawLX
KLDKs0yyhLDnKEjmx/Vhz03JpsVFJ84kSWkTZkYsXiG306TxuJCX9zAt1z+6Clie
TTGiFY+D8DfkC4H82rlPEtImpZ6rInsMUlAykImpd58e4PMSa+w/wSHXDvwFP7py
1Gvz3XvcbGLmpBXblxTUpToqC7zSQJhHOMBBt6XnhcRwd6G9Vj/mQM3FvJIrxtKk
8O7FwMJloGivS85OEzCIur5A+bObXbM2pcI8y4ueHE4NtElRBwn859AdB2k=
-----END CERTIFICATE-----`)

	testKey = []byte(testingKey(`-----BEGIN RSA TESTING KEY-----
MIIEogIBAAKCAQEApIdwfwSNK+nc2bggcAXEb0Nmbz7SHUHBTQ5fRUdwn4J/5/qH
Rdgxwyj0sUSmdQlFEyMqsziaQjvk0gi580E34au2qpM9OJ3zB84a79R9rkQRTWFv
aPsCIyWkzqLT5nKxdI2j5Vz9w1lPQ9d/I60mdibqn1ZkzzggPW3Z5rRsvALXZxMq
/mjJtUa0keHvpNHEBHcV5D35JjfS9QDFibqhbJiGjytF0q1RJ7XWlePo6HHF9Nme
xJafiWjS4GldqLrde2J5NWnPviG1I0LVu6cga/v/+bkXrSwfkysXklKQ/F8J+Cqy
SLzggINJE2W+lP54w+LKAAAnzNjzBhRdFgVhNQIDAQABAoIBAAtUbFHC3XnVq+iu
PkWYkBNdX9NvTwbGvWnyAGuD5OSHFwnBfck4fwzCaD9Ay/mpPsF3nXwj/LNs7m/s
O+ndZty6d2S9qOyaK98wuTgkuNbkRxC+Ee73wgjrkbLNEax/32p4Sn4D7lGid8vj
LhUl2k0ult+MEnsWkVnJk8TITeiQaT2AHhMr3HKdaI86hJJfam3wEBiLBglnnKqA
TInMqHoudnFOn/C8iVCFuHCE0oo1dMalbc4rlZuRBqezVhbSMWPLypMVXQb7eixM
ScJ3m8+DooGDSIe+EW/afhN2VnFbrhQC9/DlxGfwTwsUseWv7pgp53ufyyAzzydn
2plW/4ECgYEA1Va5RzSUDxr75JX003YZiBcYrG268vosiNYWRhE7frvn5EorZBRW
t4R70Y2gcXA10aPHzpbq40t6voWtpkfynU3fyRzbBmwfiWLEgckrYMwtcNz8nhG2
ETAg4LXO9CufbwuDa66h76TpkBzQVNc5TSbBUr/apLDWjKPMz6qW7VUCgYEAxW4K
Yqp3NgJkC5DhuD098jir9AH96hGhUryOi2CasCvmbjWCgWdolD7SRZJfxOXFOtHv
7Dkp9glA1Cg/nSmEHKslaTJfBIWK+5rqVD6k6kZE/+4QQWQtUxXXVgGINnGrnPvo
6MlRJxqGUtYJ0GRTFJP4Py0gwuzf5BMIwe+fpGECgYAOhLRfMCjTTlbOG5ZpvaPH
Kys2sNEEMBpPxaIGaq3N1iPV2WZSjT/JhW6XuDevAJ/pAGhcmtCpXz2fMaG7qzHL
mr0cBqaxLTKIOvx8iKA3Gi4NfDyE1Ve6m7fhEv5eh4l2GSZ8cYn7sRFkCVH0NCFm
KrkFVKEgjBhNwefySf2zcQKBgHDVPgw7nlv4q9LMX6RbI98eMnAG/2XZ45gUeWcA
tAeBX3WXEVoBjoxDBwuJ5z/xjXHbb8JSvT+G9E0MH6cjhgSYb44aoqFD7TV0yP2S
u8/Ej0SxewrURO8aKXJW99Edz9WtRuRbwgyWJTSMbRlzbOPy2UrJ8NJWbHK9yiCE
YXmhAoGAA3QUiCCl11c1C4VsF68Fa2i7qwnty3fvFidZpW3ds0tzZdIvkpRLp5+u
XAJ5+zStdEGdnu0iXALQlY7ektawXguT/zYKg3nfS9RMGW6CxZotn4bqfQwDuttf
b1xn1jGQd/o0xFf9ojpDNy6vNojidQGHh6E3h0GYvxbnQmVNq5U=
-----END RSA TESTING KEY-----`))

	unrelatedTestKey = []byte(testingKey(`-----BEGIN TESTING KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDbIFkAXN3M1A2/
xRWwWJcuT+cXeD3+1W0EEHEYT+Ad8zNGB3yTsn7iw8MELalrXbUnKJkkiah8XbSh
bw9ngHLHXTIbhvl2ceg7dwWNrK+286k5e/vVH/wwWfImBl8gK6ksoDic5U1fTzGQ
1wkhfqIScD9j0hR7wwUxwJejKxzBq83gl3x5p9JgiVNcIo5h4EowshkVNo+xPuL9
ZBZGUp6w2XBAdOfOzQbqIMbDy/puKz3n5kckAOtcgH+T/AFn/asMCAde+Ym1mPl/
/wR7rPJHJvjrq6TSzga85pZJqszlleFQ74MLm7tXtVMlXHLR86Po9NXG1Z3G7Cna
Io4o2F+bAgMBAAECggEABh5jHdV6BAwvzhkMv/3ZStvEUi1zXbhL8P8ciVdBpNRz
rBLtcZpcXKymt2km/+5/7nX9wL1vTPm434EgZv15NwPtMEOWl64ak/6A0zHtPiiT
ox1JLOxVuGvqjRFEert9X9ehfRASFwU5FxhKEvtcPzOPMZReKg6KCJeeJFpB1U6P
l40F37bneQqHfOj+8h1m+7zL07w0Vfl2XMdvM1TKf1KACxBfgIhqKL1TO0LrI/fC
iJsL6948sBe//e/Ee11CA8+VcqbrJIlE+wouasFQWhKjJQoGcrdh/3PtgfRfTRWP
3HpOlPSLeizXdwOJOKmZv/XeOGlOpl2xBAC6rO9hAQKBgQDzz5Wxe4e0dBYzra7G
V046O9Q34k2Gfi4BlV5NqCnFjcBwvxjOGE/rfjsgQO5bdZWGUqzNIubYvOisNiv9
k1RqQXFibpHmKcWZz6zNHfOpZC6/ACKA72cR2TyOaj8YLk69do5zc4jz7vH6pU5C
q6jBIX5nrm62wycdBImpx2JhwQKBgQDmFNaB3dd8xvVtigNUVhAXEkZMj4N3IOiL
esvfLEoqiaNMgZTRH2sas1BATxXNAs+PzYTxn2+bq4/63AcSx9Pd4QbwuhN0D0Pp
24ZU5FYWVPtpztWXnTNgfP6rj7MngJmLFjqVJynwt1ZIA2mkymjDyOo7+bEtAXvV
evvW/4egWwKBgQDvcnPltxh0FX6oim8XxC7D6nZl3A+fgtTUIUpYoktEBg91q3hF
EIONGJAhASQXFsge/5tObHSjcARi/WD+zW8eW99reIQ5s9SpVtizKjNfrVBrrUo1
rulfEibzB02oBfK3CHSm1lUunQFx1F+kAsrdwnNOiHWbcNY9HXPGFld9AQKBgQC0
qFoCILGpzQM63mpc1zLNGtFOHkXIzXMqyeG4u6sEmYw6b2jthzDvBysVQ8PHdNSL
goFHw7u7zMtB23BGY9dM2fs8G69Yqv/VaUSh9aRO5q1+WCTIZmvH8H17MlsmwkhN
uMeJA/Zfh2VdKCjUdwYp7OFW9GkVAJw+dNG38G6LDwKBgQCC1zsyTKl0Y5DElJc5
e+Z1cALnWREYhEPv4JrR5U0VvqeIdExDD6Ida61yvd7oc59pn0kpfKjozPJr6FsU
2AUs1ibpKVgbzDfDZiX1xWgt1OJ9x9yMi9ZvQAU5oYckiV88VjS5c2PFc0m7g6ik
2GYiMrU/kZ2OfcPxYtZJpZG+kw==
-----END TESTING KEY-----`))
)

// prevent static analysis tools from raising issues due to detection of private key
// in code.
func testingKey(s string) string { return strings.ReplaceAll(s, "TESTING KEY", "PRIVATE KEY") }

func TestValidateCloudfrontURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		url        string
		publicKey  string
		privateKey string
		errMatches string
	}{
		{"happy path", "https://example.com", "public", "private", ""},
		{"bad URL", "bozo!://example.com", "public", "private", "parse"},
		{"non-HTTPS URL", "http://example.com", "public", "private", "cloudfront url scheme must be https"},
		{"missing URL", "", "public", "private", "`s3_software_installers_cloudfront_url` must be set"},
		{"missing public key", "https://example.com", "", "private",
			"Both `s3_software_installers_cloudfront_url_signing_public_key_id` and `s3_software_installers_cloudfront_url_signing_private_key` must be set"},
		{"missing private key", "https://example.com", "public", "",
			"Both `s3_software_installers_cloudfront_url_signing_public_key_id` and `s3_software_installers_cloudfront_url_signing_private_key` must be set"},
		{"missing keys", "https://example.com", "", "",
			"Both `s3_software_installers_cloudfront_url_signing_public_key_id` and `s3_software_installers_cloudfront_url_signing_private_key` must be set"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s3 := S3Config{
				SoftwareInstallersCloudFrontURL:                   c.url,
				SoftwareInstallersCloudFrontURLSigningPublicKeyID: c.publicKey,
				SoftwareInstallersCloudFrontURLSigningPrivateKey:  c.privateKey,
			}
			initFatal := func(err error, msg string) {
				if c.errMatches != "" {
					require.Error(t, err)
					require.Regexp(t, c.errMatches, err.Error())
				} else {
					t.Errorf("unexpected error: %v", err)
				}
			}
			s3.ValidateCloudFrontURL(initFatal)
		})
	}
}
