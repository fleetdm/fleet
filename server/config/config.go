package config

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	envPrefix = "FLEET"
)

// MysqlConfig defines configs related to MySQL
type MysqlConfig struct {
	Protocol        string
	Address         string
	Username        string
	Password        string
	PasswordPath    string `yaml:"password_path"`
	Database        string
	TLSCert         string `yaml:"tls_cert"`
	TLSKey          string `yaml:"tls_key"`
	TLSCA           string `yaml:"tls_ca"`
	TLSServerName   string `yaml:"tls_server_name"`
	TLSConfig       string `yaml:"tls_config"` // tls=customValue in DSN
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
}

// RedisConfig defines configs related to Redis
type RedisConfig struct {
	Address                   string
	Password                  string
	Database                  int
	UseTLS                    bool          `yaml:"use_tls"`
	DuplicateResults          bool          `yaml:"duplicate_results"`
	ConnectTimeout            time.Duration `yaml:"connect_timeout"`
	KeepAlive                 time.Duration `yaml:"keep_alive"`
	ConnectRetryAttempts      int           `yaml:"connect_retry_attempts"`
	ClusterFollowRedirections bool          `yaml:"cluster_follow_redirections"`
	ClusterReadFromReplica    bool          `yaml:"cluster_read_from_replica"`
	TLSCert                   string        `yaml:"tls_cert"`
	TLSKey                    string        `yaml:"tls_key"`
	TLSCA                     string        `yaml:"tls_ca"`
	TLSServerName             string        `yaml:"tls_server_name"`
	TLSHandshakeTimeout       time.Duration `yaml:"tls_handshake_timeout"`
	// TODO(mna): should we allow insecure skip verify option?
	MaxIdleConns int `yaml:"max_idle_conns"`
	MaxOpenConns int `yaml:"max_open_conns"`
	// this config is an int on MysqlConfig, but it should be a time.Duration.
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ConnWaitTimeout time.Duration `yaml:"conn_wait_timeout"`
}

const (
	TLSProfileKey          = "server.tls_compatibility"
	TLSProfileModern       = "modern"
	TLSProfileIntermediate = "intermediate"
)

// ServerConfig defines configs related to the Fleet server
type ServerConfig struct {
	Address    string
	Cert       string
	Key        string
	TLS        bool
	TLSProfile string `yaml:"tls_compatibility"`
	URLPrefix  string `yaml:"url_prefix"`
	Keepalive  bool   `yaml:"keepalive"`
}

// AuthConfig defines configs related to user authorization
type AuthConfig struct {
	BcryptCost  int `yaml:"bcrypt_cost"`
	SaltKeySize int `yaml:"salt_key_size"`
}

// AppConfig defines configs related to HTTP
type AppConfig struct {
	TokenKeySize              int           `yaml:"token_key_size"`
	InviteTokenValidityPeriod time.Duration `yaml:"invite_token_validity_period"`
}

// SessionConfig defines configs related to user sessions
type SessionConfig struct {
	KeySize  int `yaml:"key_size"`
	Duration time.Duration
}

// OsqueryConfig defines configs related to osquery
type OsqueryConfig struct {
	NodeKeySize                      int           `yaml:"node_key_size"`
	HostIdentifier                   string        `yaml:"host_identifier"`
	EnrollCooldown                   time.Duration `yaml:"enroll_cooldown"`
	StatusLogPlugin                  string        `yaml:"status_log_plugin"`
	ResultLogPlugin                  string        `yaml:"result_log_plugin"`
	LabelUpdateInterval              time.Duration `yaml:"label_update_interval"`
	PolicyUpdateInterval             time.Duration `yaml:"policy_update_interval"`
	DetailUpdateInterval             time.Duration `yaml:"detail_update_interval"`
	StatusLogFile                    string        `yaml:"status_log_file"`
	ResultLogFile                    string        `yaml:"result_log_file"`
	EnableLogRotation                bool          `yaml:"enable_log_rotation"`
	MaxJitterPercent                 int           `yaml:"max_jitter_percent"`
	EnableAsyncHostProcessing        bool          `yaml:"enable_async_host_processing"`
	AsyncHostCollectInterval         time.Duration `yaml:"async_host_collect_interval"`
	AsyncHostCollectMaxJitterPercent int           `yaml:"async_host_collect_max_jitter_percent"`
	AsyncHostCollectLockTimeout      time.Duration `yaml:"async_host_collect_lock_timeout"`
	AsyncHostCollectLogStatsInterval time.Duration `yaml:"async_host_collect_log_stats_interval"`
	AsyncHostInsertBatch             int           `yaml:"async_host_insert_batch"`
	AsyncHostDeleteBatch             int           `yaml:"async_host_delete_batch"`
	AsyncHostUpdateBatch             int           `yaml:"async_host_update_batch"`
	AsyncHostRedisPopCount           int           `yaml:"async_host_redis_pop_count"`
	AsyncHostRedisScanKeysCount      int           `yaml:"async_host_redis_scan_keys_count"`
}

// LoggingConfig defines configs related to logging
type LoggingConfig struct {
	Debug                bool
	JSON                 bool
	DisableBanner        bool          `yaml:"disable_banner"`
	ErrorRetentionPeriod time.Duration `yaml:"error_retention_period"`
}

// FirehoseConfig defines configs for the AWS Firehose logging plugin
type FirehoseConfig struct {
	Region           string
	EndpointURL      string `yaml:"endpoint_url"`
	AccessKeyID      string `yaml:"access_key_id"`
	SecretAccessKey  string `yaml:"secret_access_key"`
	StsAssumeRoleArn string `yaml:"sts_assume_role_arn"`
	StatusStream     string `yaml:"status_stream"`
	ResultStream     string `yaml:"result_stream"`
}

// KinesisConfig defines configs for the AWS Kinesis logging plugin
type KinesisConfig struct {
	Region           string
	EndpointURL      string `yaml:"endpoint_url"`
	AccessKeyID      string `yaml:"access_key_id"`
	SecretAccessKey  string `yaml:"secret_access_key"`
	StsAssumeRoleArn string `yaml:"sts_assume_role_arn"`
	StatusStream     string `yaml:"status_stream"`
	ResultStream     string `yaml:"result_stream"`
}

// LambdaConfig defines configs for the AWS Lambda logging plugin
type LambdaConfig struct {
	Region           string
	AccessKeyID      string `yaml:"access_key_id"`
	SecretAccessKey  string `yaml:"secret_access_key"`
	StsAssumeRoleArn string `yaml:"sts_assume_role_arn"`
	StatusFunction   string `yaml:"status_function"`
	ResultFunction   string `yaml:"result_function"`
}

// S3Config defines config to enable file carving storage to an S3 bucket
type S3Config struct {
	Bucket           string `yaml:"bucket"`
	Prefix           string `yaml:"prefix"`
	Region           string `yaml:"region"`
	EndpointURL      string `yaml:"endpoint_url"`
	AccessKeyID      string `yaml:"access_key_id"`
	SecretAccessKey  string `yaml:"secret_access_key"`
	StsAssumeRoleArn string `yaml:"sts_assume_role_arn"`
	DisableSSL       bool   `yaml:"disable_ssl"`
	ForceS3PathStyle bool   `yaml:"force_s3_path_style"`
}

// PubSubConfig defines configs the for Google PubSub logging plugin
type PubSubConfig struct {
	Project       string `json:"project"`
	StatusTopic   string `json:"status_topic" yaml:"status_topic"`
	ResultTopic   string `json:"result_topic" yaml:"result_topic"`
	AddAttributes bool   `json:"add_attributes" yaml:"add_attributes"`
}

// FilesystemConfig defines configs for the Filesystem logging plugin
type FilesystemConfig struct {
	StatusLogFile        string `json:"status_log_file" yaml:"status_log_file"`
	ResultLogFile        string `json:"result_log_file" yaml:"result_log_file"`
	EnableLogRotation    bool   `json:"enable_log_rotation" yaml:"enable_log_rotation"`
	EnableLogCompression bool   `json:"enable_log_compression" yaml:"enable_log_compression"`
}

// KafkaRESTConfig defines configs for the Kafka REST Proxy logging plugin.
type KafkaRESTConfig struct {
	StatusTopic string `json:"status_topic" yaml:"status_topic"`
	ResultTopic string `json:"result_topic" yaml:"result_topic"`
	ProxyHost   string `json:"proxyhost" yaml:"proxyhost"`
	Timeout     int    `json:"timeout" yaml:"timeout"`
}

// LicenseConfig defines configs related to licensing Fleet.
type LicenseConfig struct {
	Key string `yaml:"key"`
}

// VulnerabilitiesConfig defines configs related to vulnerability processing within Fleet.
type VulnerabilitiesConfig struct {
	DatabasesPath         string        `json:"databases_path" yaml:"databases_path"`
	Periodicity           time.Duration `json:"periodicity" yaml:"periodicity"`
	CPEDatabaseURL        string        `json:"cpe_database_url" yaml:"cpe_database_url"`
	CVEFeedPrefixURL      string        `json:"cve_feed_prefix_url" yaml:"cve_feed_prefix_url"`
	CurrentInstanceChecks string        `json:"current_instance_checks" yaml:"current_instance_checks"`
	DisableDataSync       bool          `json:"disable_data_sync" yaml:"disable_data_sync"`
}

// UpgradesConfig defines configs related to fleet server upgrades.
type UpgradesConfig struct {
	AllowMissingMigrations bool `json:"allow_missing_migrations" yaml:"allow_missing_migrations"`
}

// FleetConfig stores the application configuration. Each subcategory is
// broken up into it's own struct, defined above. When editing any of these
// structs, Manager.addConfigs and Manager.LoadConfig should be
// updated to set and retrieve the configurations as appropriate.
type FleetConfig struct {
	Mysql            MysqlConfig
	MysqlReadReplica MysqlConfig `yaml:"mysql_read_replica"`
	Redis            RedisConfig
	Server           ServerConfig
	Auth             AuthConfig
	App              AppConfig
	Session          SessionConfig
	Osquery          OsqueryConfig
	Logging          LoggingConfig
	Firehose         FirehoseConfig
	Kinesis          KinesisConfig
	Lambda           LambdaConfig
	S3               S3Config
	PubSub           PubSubConfig
	Filesystem       FilesystemConfig
	KafkaREST        KafkaRESTConfig
	License          LicenseConfig
	Vulnerabilities  VulnerabilitiesConfig
	Upgrades         UpgradesConfig
}

type TLS struct {
	TLSCert       string
	TLSKey        string
	TLSCA         string
	TLSServerName string
}

func (t *TLS) ToTLSConfig() (*tls.Config, error) {
	var rootCertPool *x509.CertPool
	if t.TLSCA != "" {
		rootCertPool = x509.NewCertPool()
		pem, err := ioutil.ReadFile(t.TLSCA)
		if err != nil {
			return nil, fmt.Errorf("read server-ca pem: %w", err)
		}
		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			return nil, errors.New("failed to append PEM.")
		}
	}

	cfg := &tls.Config{
		RootCAs: rootCertPool,
	}
	if t.TLSCert != "" {
		clientCert := make([]tls.Certificate, 0, 1)
		certs, err := tls.LoadX509KeyPair(t.TLSCert, t.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("load client cert and key: %w", err)
		}
		clientCert = append(clientCert, certs)
		cfg.Certificates = clientCert
	}

	if t.TLSServerName != "" {
		cfg.ServerName = t.TLSServerName
	}
	return cfg, nil
}

// addConfigs adds the configuration keys and default values that will be
// filled into the FleetConfig struct
func (man Manager) addConfigs() {
	addMysqlConfig := func(prefix, defaultAddr, usageSuffix string) {
		man.addConfigString(prefix+".protocol", "tcp",
			"MySQL server communication protocol (tcp,unix,...)"+usageSuffix)
		man.addConfigString(prefix+".address", defaultAddr,
			"MySQL server address (host:port)"+usageSuffix)
		man.addConfigString(prefix+".username", "fleet",
			"MySQL server username"+usageSuffix)
		man.addConfigString(prefix+".password", "",
			"MySQL server password (prefer env variable for security)"+usageSuffix)
		man.addConfigString(prefix+".password_path", "",
			"Path to file containg MySQL server password"+usageSuffix)
		man.addConfigString(prefix+".database", "fleet",
			"MySQL database name"+usageSuffix)
		man.addConfigString(prefix+".tls_cert", "",
			"MySQL TLS client certificate path"+usageSuffix)
		man.addConfigString(prefix+".tls_key", "",
			"MySQL TLS client key path"+usageSuffix)
		man.addConfigString(prefix+".tls_ca", "",
			"MySQL TLS server CA"+usageSuffix)
		man.addConfigString(prefix+".tls_server_name", "",
			"MySQL TLS server name"+usageSuffix)
		man.addConfigString(prefix+".tls_config", "",
			"MySQL TLS config value"+usageSuffix+" Use skip-verify, true, false or custom key.")
		man.addConfigInt(prefix+".max_open_conns", 50, "MySQL maximum open connection handles"+usageSuffix)
		man.addConfigInt(prefix+".max_idle_conns", 50, "MySQL maximum idle connection handles"+usageSuffix)
		man.addConfigInt(prefix+".conn_max_lifetime", 0, "MySQL maximum amount of time a connection may be reused"+usageSuffix)
	}
	// MySQL
	addMysqlConfig("mysql", "localhost:3306", ".")
	addMysqlConfig("mysql_read_replica", "", " for the read replica.")

	// Redis
	man.addConfigString("redis.address", "localhost:6379",
		"Redis server address (host:port)")
	man.addConfigString("redis.password", "",
		"Redis server password (prefer env variable for security)")
	man.addConfigInt("redis.database", 0,
		"Redis server database number")
	man.addConfigBool("redis.use_tls", false, "Redis server enable TLS")
	man.addConfigBool("redis.duplicate_results", false, "Duplicate Live Query results to another Redis channel")
	man.addConfigDuration("redis.connect_timeout", 5*time.Second, "Timeout at connection time")
	man.addConfigDuration("redis.keep_alive", 10*time.Second, "Interval between keep alive probes")
	man.addConfigInt("redis.connect_retry_attempts", 0, "Number of attempts to retry a failed connection")
	man.addConfigBool("redis.cluster_follow_redirections", false, "Automatically follow Redis Cluster redirections")
	man.addConfigBool("redis.cluster_read_from_replica", false, "Prefer reading from a replica when possible (for Redis Cluster)")
	man.addConfigString("redis.tls_cert", "", "Redis TLS client certificate path")
	man.addConfigString("redis.tls_key", "", "Redis TLS client key path")
	man.addConfigString("redis.tls_ca", "", "Redis TLS server CA")
	man.addConfigString("redis.tls_server_name", "", "Redis TLS server name")
	man.addConfigDuration("redis.tls_handshake_timeout", 10*time.Second, "Redis TLS handshake timeout")
	man.addConfigInt("redis.max_idle_conns", 3, "Redis maximum idle connections")
	man.addConfigInt("redis.max_open_conns", 0, "Redis maximum open connections, 0 means no limit")
	man.addConfigDuration("redis.conn_max_lifetime", 0, "Redis maximum amount of time a connection may be reused, 0 means no limit")
	man.addConfigDuration("redis.idle_timeout", 240*time.Second, "Redis maximum amount of time a connection may stay idle, 0 means no limit")
	man.addConfigDuration("redis.conn_wait_timeout", 0, "Redis maximum amount of time to wait for a connection if the maximum is reached (0 for no wait, ignored in non-cluster Redis)")

	// Server
	man.addConfigString("server.address", "0.0.0.0:8080",
		"Fleet server address (host:port)")
	man.addConfigString("server.cert", "./tools/osquery/fleet.crt",
		"Fleet TLS certificate path")
	man.addConfigString("server.key", "./tools/osquery/fleet.key",
		"Fleet TLS key path")
	man.addConfigBool("server.tls", true,
		"Enable TLS (required for osqueryd communication)")
	man.addConfigString(TLSProfileKey, TLSProfileIntermediate,
		fmt.Sprintf("TLS security profile choose one of %s or %s",
			TLSProfileModern, TLSProfileIntermediate))
	man.addConfigString("server.url_prefix", "",
		"URL prefix used on server and frontend endpoints")
	man.addConfigBool("server.keepalive", true,
		"Controls wether HTTP keep-alives are enabled.")

	// Auth
	man.addConfigInt("auth.bcrypt_cost", 12,
		"Bcrypt iterations")
	man.addConfigInt("auth.salt_key_size", 24,
		"Size of salt for passwords")

	// App
	man.addConfigString("app.token_key", "CHANGEME",
		"Secret key for generating invite and reset tokens")
	man.addConfigDuration("app.invite_token_validity_period", 5*24*time.Hour,
		"Duration invite tokens remain valid (i.e. 1h)")
	man.addConfigInt("app.token_key_size", 24,
		"Size of generated tokens")

	// Session
	man.addConfigInt("session.key_size", 64,
		"Size of generated session keys")
	man.addConfigDuration("session.duration", 24*time.Hour,
		"Duration session keys remain valid (i.e. 4h)")

	// Osquery
	man.addConfigInt("osquery.node_key_size", 24,
		"Size of generated osqueryd node keys")
	man.addConfigString("osquery.host_identifier", "provided",
		"Identifier used to uniquely determine osquery clients")
	man.addConfigDuration("osquery.enroll_cooldown", 0,
		"Cooldown period for duplicate host enrollment (default off)")
	man.addConfigString("osquery.status_log_plugin", "filesystem",
		"Log plugin to use for status logs")
	man.addConfigString("osquery.result_log_plugin", "filesystem",
		"Log plugin to use for result logs")
	man.addConfigDuration("osquery.label_update_interval", 1*time.Hour,
		"Interval to update host label membership (i.e. 1h)")
	man.addConfigDuration("osquery.policy_update_interval", 1*time.Hour,
		"Interval to update host policy membership (i.e. 1h)")
	man.addConfigDuration("osquery.detail_update_interval", 1*time.Hour,
		"Interval to update host details (i.e. 1h)")
	man.addConfigString("osquery.status_log_file", "",
		"(DEPRECATED: Use filesystem.status_log_file) Path for osqueryd status logs")
	man.addConfigString("osquery.result_log_file", "",
		"(DEPRECATED: Use filesystem.result_log_file) Path for osqueryd result logs")
	man.addConfigBool("osquery.enable_log_rotation", false,
		"(DEPRECATED: Use filesystem.enable_log_rotation) Enable automatic rotation for osquery log files")
	man.addConfigInt("osquery.max_jitter_percent", 10,
		"Maximum percentage of the interval to add as jitter")
	man.addConfigBool("osquery.enable_async_host_processing", false,
		"Enable asynchronous processing of host-reported query results")
	man.addConfigDuration("osquery.async_host_collect_interval", 30*time.Second,
		"Interval to collect asynchronous host-reported query results (i.e. 30s)")
	man.addConfigInt("osquery.async_host_collect_max_jitter_percent", 10,
		"Maximum percentage of the interval to collect asynchronous host results")
	man.addConfigDuration("osquery.async_host_collect_lock_timeout", 1*time.Minute,
		"Timeout of the exclusive lock held during async host collection")
	man.addConfigDuration("osquery.async_host_collect_log_stats_interval", 1*time.Minute,
		"Interval at which async host collection statistics are logged (0 disables logging of stats)")
	man.addConfigInt("osquery.async_host_insert_batch", 2000,
		"Batch size for async collection inserts in mysql")
	man.addConfigInt("osquery.async_host_delete_batch", 2000,
		"Batch size for async collection deletes in mysql")
	man.addConfigInt("osquery.async_host_update_batch", 1000,
		"Batch size for async collection updates in mysql")
	man.addConfigInt("osquery.async_host_redis_pop_count", 1000,
		"Batch size to pop items from redis in async collection")
	man.addConfigInt("osquery.async_host_redis_scan_keys_count", 1000,
		"Batch size to scan redis keys in async collection")

	// Logging
	man.addConfigBool("logging.debug", false,
		"Enable debug logging")
	man.addConfigBool("logging.json", false,
		"Log in JSON format")
	man.addConfigBool("logging.disable_banner", false,
		"Disable startup banner")
	man.addConfigDuration("logging.error_retention_period", 24*time.Hour,
		"Amount of time to keep errors")

	// Firehose
	man.addConfigString("firehose.region", "", "AWS Region to use")
	man.addConfigString("firehose.endpoint_url", "",
		"AWS Service Endpoint to use (leave empty for default service endpoints)")
	man.addConfigString("firehose.access_key_id", "", "Access Key ID for AWS authentication")
	man.addConfigString("firehose.secret_access_key", "", "Secret Access Key for AWS authentication")
	man.addConfigString("firehose.sts_assume_role_arn", "",
		"ARN of role to assume for AWS")
	man.addConfigString("firehose.status_stream", "",
		"Firehose stream name for status logs")
	man.addConfigString("firehose.result_stream", "",
		"Firehose stream name for result logs")

	// Kinesis
	man.addConfigString("kinesis.region", "", "AWS Region to use")
	man.addConfigString("kinesis.endpoint_url", "",
		"AWS Service Endpoint to use (leave empty for default service endpoints)")
	man.addConfigString("kinesis.access_key_id", "", "Access Key ID for AWS authentication")
	man.addConfigString("kinesis.secret_access_key", "", "Secret Access Key for AWS authentication")
	man.addConfigString("kinesis.sts_assume_role_arn", "",
		"ARN of role to assume for AWS")
	man.addConfigString("kinesis.status_stream", "",
		"Kinesis stream name for status logs")
	man.addConfigString("kinesis.result_stream", "",
		"Kinesis stream name for result logs")

	// Lambda
	man.addConfigString("lambda.region", "", "AWS Region to use")
	man.addConfigString("lambda.access_key_id", "", "Access Key ID for AWS authentication")
	man.addConfigString("lambda.secret_access_key", "", "Secret Access Key for AWS authentication")
	man.addConfigString("lambda.sts_assume_role_arn", "",
		"ARN of role to assume for AWS")
	man.addConfigString("lambda.status_function", "",
		"Lambda function name for status logs")
	man.addConfigString("lambda.result_function", "",
		"Lambda function name for result logs")

	// S3 for file carving
	man.addConfigString("s3.bucket", "", "Bucket where to store file carves")
	man.addConfigString("s3.prefix", "", "Prefix under which carves are stored")
	man.addConfigString("s3.region", "", "AWS Region (if blank region is derived)")
	man.addConfigString("s3.endpoint_url", "", "AWS Service Endpoint to use (leave blank for default service endpoints)")
	man.addConfigString("s3.access_key_id", "", "Access Key ID for AWS authentication")
	man.addConfigString("s3.secret_access_key", "", "Secret Access Key for AWS authentication")
	man.addConfigString("s3.sts_assume_role_arn", "", "ARN of role to assume for AWS")
	man.addConfigBool("s3.disable_ssl", false, "Disable SSL (typically for local testing)")
	man.addConfigBool("s3.force_s3_path_style", false, "Set this to true to force path-style addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`")

	// PubSub
	man.addConfigString("pubsub.project", "", "Google Cloud Project to use")
	man.addConfigString("pubsub.status_topic", "", "PubSub topic for status logs")
	man.addConfigString("pubsub.result_topic", "", "PubSub topic for result logs")
	man.addConfigBool("pubsub.add_attributes", false, "Add PubSub attributes in addition to the message body")

	// Filesystem
	man.addConfigString("filesystem.status_log_file", filepath.Join(os.TempDir(), "osquery_status"),
		"Log file path to use for status logs")
	man.addConfigString("filesystem.result_log_file", filepath.Join(os.TempDir(), "osquery_result"),
		"Log file path to use for result logs")
	man.addConfigBool("filesystem.enable_log_rotation", false,
		"Enable automatic rotation for osquery log files")
	man.addConfigBool("filesystem.enable_log_compression", false,
		"Enable compression for the rotated osquery log files")

	// KafkaREST
	man.addConfigString("kafkarest.status_topic", "", "Kafka REST topic for status logs")
	man.addConfigString("kafkarest.result_topic", "", "Kafka REST topic for result logs")
	man.addConfigString("kafkarest.proxyhost", "", "Kafka REST proxy host url")
	man.addConfigInt("kafkarest.timeout", 5, "Kafka REST proxy json post timeout")

	// License
	man.addConfigString("license.key", "", "Fleet license key (to enable Fleet Premium features)")

	// Vulnerability processing
	man.addConfigString("vulnerabilities.databases_path", "/tmp/vulndbs",
		"Path where Fleet will download the data feeds to check CVEs")
	man.addConfigDuration("vulnerabilities.periodicity", 1*time.Hour,
		"How much time to wait between processing software for vulnerabilities.")
	man.addConfigString("vulnerabilities.cpe_database_url", "",
		"URL from which to get the latest CPE database. If empty, defaults to the official Github link.")
	man.addConfigString("vulnerabilities.cve_feed_prefix_url", "",
		"Prefix URL for the CVE data feed. If empty, default to https://nvd.nist.gov/")
	man.addConfigString("vulnerabilities.current_instance_checks", "auto",
		"Allows to manually select an instance to do the vulnerability processing.")
	man.addConfigBool("vulnerabilities.disable_data_sync", false,
		"Skips synchronizing data streams and expects them to be available in the databases_path.")

	// Upgrades
	man.addConfigBool("upgrades.allow_missing_migrations", false,
		"Allow serve to run even if migrations are missing.")
}

// LoadConfig will load the config variables into a fully initialized
// FleetConfig struct
func (man Manager) LoadConfig() FleetConfig {
	man.loadConfigFile()

	loadMysqlConfig := func(prefix string) MysqlConfig {
		return MysqlConfig{
			Protocol:        man.getConfigString(prefix + ".protocol"),
			Address:         man.getConfigString(prefix + ".address"),
			Username:        man.getConfigString(prefix + ".username"),
			Password:        man.getConfigString(prefix + ".password"),
			PasswordPath:    man.getConfigString(prefix + ".password_path"),
			Database:        man.getConfigString(prefix + ".database"),
			TLSCert:         man.getConfigString(prefix + ".tls_cert"),
			TLSKey:          man.getConfigString(prefix + ".tls_key"),
			TLSCA:           man.getConfigString(prefix + ".tls_ca"),
			TLSServerName:   man.getConfigString(prefix + ".tls_server_name"),
			TLSConfig:       man.getConfigString(prefix + ".tls_config"),
			MaxOpenConns:    man.getConfigInt(prefix + ".max_open_conns"),
			MaxIdleConns:    man.getConfigInt(prefix + ".max_idle_conns"),
			ConnMaxLifetime: man.getConfigInt(prefix + ".conn_max_lifetime"),
		}
	}

	return FleetConfig{
		Mysql:            loadMysqlConfig("mysql"),
		MysqlReadReplica: loadMysqlConfig("mysql_read_replica"),
		Redis: RedisConfig{
			Address:                   man.getConfigString("redis.address"),
			Password:                  man.getConfigString("redis.password"),
			Database:                  man.getConfigInt("redis.database"),
			UseTLS:                    man.getConfigBool("redis.use_tls"),
			DuplicateResults:          man.getConfigBool("redis.duplicate_results"),
			ConnectTimeout:            man.getConfigDuration("redis.connect_timeout"),
			KeepAlive:                 man.getConfigDuration("redis.keep_alive"),
			ConnectRetryAttempts:      man.getConfigInt("redis.connect_retry_attempts"),
			ClusterFollowRedirections: man.getConfigBool("redis.cluster_follow_redirections"),
			ClusterReadFromReplica:    man.getConfigBool("redis.cluster_read_from_replica"),
			TLSCert:                   man.getConfigString("redis.tls_cert"),
			TLSKey:                    man.getConfigString("redis.tls_key"),
			TLSCA:                     man.getConfigString("redis.tls_ca"),
			TLSServerName:             man.getConfigString("redis.tls_server_name"),
			TLSHandshakeTimeout:       man.getConfigDuration("redis.tls_handshake_timeout"),
			MaxIdleConns:              man.getConfigInt("redis.max_idle_conns"),
			MaxOpenConns:              man.getConfigInt("redis.max_open_conns"),
			ConnMaxLifetime:           man.getConfigDuration("redis.conn_max_lifetime"),
			IdleTimeout:               man.getConfigDuration("redis.idle_timeout"),
			ConnWaitTimeout:           man.getConfigDuration("redis.conn_wait_timeout"),
		},
		Server: ServerConfig{
			Address:    man.getConfigString("server.address"),
			Cert:       man.getConfigString("server.cert"),
			Key:        man.getConfigString("server.key"),
			TLS:        man.getConfigBool("server.tls"),
			TLSProfile: man.getConfigTLSProfile(),
			URLPrefix:  man.getConfigString("server.url_prefix"),
			Keepalive:  man.getConfigBool("server.keepalive"),
		},
		Auth: AuthConfig{
			BcryptCost:  man.getConfigInt("auth.bcrypt_cost"),
			SaltKeySize: man.getConfigInt("auth.salt_key_size"),
		},
		App: AppConfig{
			TokenKeySize:              man.getConfigInt("app.token_key_size"),
			InviteTokenValidityPeriod: man.getConfigDuration("app.invite_token_validity_period"),
		},
		Session: SessionConfig{
			KeySize:  man.getConfigInt("session.key_size"),
			Duration: man.getConfigDuration("session.duration"),
		},
		Osquery: OsqueryConfig{
			NodeKeySize:                      man.getConfigInt("osquery.node_key_size"),
			HostIdentifier:                   man.getConfigString("osquery.host_identifier"),
			EnrollCooldown:                   man.getConfigDuration("osquery.enroll_cooldown"),
			StatusLogPlugin:                  man.getConfigString("osquery.status_log_plugin"),
			ResultLogPlugin:                  man.getConfigString("osquery.result_log_plugin"),
			StatusLogFile:                    man.getConfigString("osquery.status_log_file"),
			ResultLogFile:                    man.getConfigString("osquery.result_log_file"),
			LabelUpdateInterval:              man.getConfigDuration("osquery.label_update_interval"),
			PolicyUpdateInterval:             man.getConfigDuration("osquery.policy_update_interval"),
			DetailUpdateInterval:             man.getConfigDuration("osquery.detail_update_interval"),
			EnableLogRotation:                man.getConfigBool("osquery.enable_log_rotation"),
			MaxJitterPercent:                 man.getConfigInt("osquery.max_jitter_percent"),
			EnableAsyncHostProcessing:        man.getConfigBool("osquery.enable_async_host_processing"),
			AsyncHostCollectInterval:         man.getConfigDuration("osquery.async_host_collect_interval"),
			AsyncHostCollectMaxJitterPercent: man.getConfigInt("osquery.async_host_collect_max_jitter_percent"),
			AsyncHostCollectLockTimeout:      man.getConfigDuration("osquery.async_host_collect_lock_timeout"),
			AsyncHostCollectLogStatsInterval: man.getConfigDuration("osquery.async_host_collect_log_stats_interval"),
			AsyncHostInsertBatch:             man.getConfigInt("osquery.async_host_insert_batch"),
			AsyncHostDeleteBatch:             man.getConfigInt("osquery.async_host_delete_batch"),
			AsyncHostUpdateBatch:             man.getConfigInt("osquery.async_host_update_batch"),
			AsyncHostRedisPopCount:           man.getConfigInt("osquery.async_host_redis_pop_count"),
			AsyncHostRedisScanKeysCount:      man.getConfigInt("osquery.async_host_redis_scan_keys_count"),
		},
		Logging: LoggingConfig{
			Debug:                man.getConfigBool("logging.debug"),
			JSON:                 man.getConfigBool("logging.json"),
			DisableBanner:        man.getConfigBool("logging.disable_banner"),
			ErrorRetentionPeriod: man.getConfigDuration("logging.error_retention_period"),
		},
		Firehose: FirehoseConfig{
			Region:           man.getConfigString("firehose.region"),
			EndpointURL:      man.getConfigString("firehose.endpoint_url"),
			AccessKeyID:      man.getConfigString("firehose.access_key_id"),
			SecretAccessKey:  man.getConfigString("firehose.secret_access_key"),
			StsAssumeRoleArn: man.getConfigString("firehose.sts_assume_role_arn"),
			StatusStream:     man.getConfigString("firehose.status_stream"),
			ResultStream:     man.getConfigString("firehose.result_stream"),
		},
		Kinesis: KinesisConfig{
			Region:           man.getConfigString("kinesis.region"),
			EndpointURL:      man.getConfigString("kinesis.endpoint_url"),
			AccessKeyID:      man.getConfigString("kinesis.access_key_id"),
			SecretAccessKey:  man.getConfigString("kinesis.secret_access_key"),
			StatusStream:     man.getConfigString("kinesis.status_stream"),
			ResultStream:     man.getConfigString("kinesis.result_stream"),
			StsAssumeRoleArn: man.getConfigString("kinesis.sts_assume_role_arn"),
		},
		Lambda: LambdaConfig{
			Region:           man.getConfigString("lambda.region"),
			AccessKeyID:      man.getConfigString("lambda.access_key_id"),
			SecretAccessKey:  man.getConfigString("lambda.secret_access_key"),
			StatusFunction:   man.getConfigString("lambda.status_function"),
			ResultFunction:   man.getConfigString("lambda.result_function"),
			StsAssumeRoleArn: man.getConfigString("lambda.sts_assume_role_arn"),
		},
		S3: S3Config{
			Bucket:           man.getConfigString("s3.bucket"),
			Prefix:           man.getConfigString("s3.prefix"),
			Region:           man.getConfigString("s3.region"),
			EndpointURL:      man.getConfigString("s3.endpoint_url"),
			AccessKeyID:      man.getConfigString("s3.access_key_id"),
			SecretAccessKey:  man.getConfigString("s3.secret_access_key"),
			StsAssumeRoleArn: man.getConfigString("s3.sts_assume_role_arn"),
			DisableSSL:       man.getConfigBool("s3.disable_ssl"),
			ForceS3PathStyle: man.getConfigBool("s3.force_s3_path_style"),
		},
		PubSub: PubSubConfig{
			Project:       man.getConfigString("pubsub.project"),
			StatusTopic:   man.getConfigString("pubsub.status_topic"),
			ResultTopic:   man.getConfigString("pubsub.result_topic"),
			AddAttributes: man.getConfigBool("pubsub.add_attributes"),
		},
		Filesystem: FilesystemConfig{
			StatusLogFile:        man.getConfigString("filesystem.status_log_file"),
			ResultLogFile:        man.getConfigString("filesystem.result_log_file"),
			EnableLogRotation:    man.getConfigBool("filesystem.enable_log_rotation"),
			EnableLogCompression: man.getConfigBool("filesystem.enable_log_compression"),
		},
		KafkaREST: KafkaRESTConfig{
			StatusTopic: man.getConfigString("kafkarest.status_topic"),
			ResultTopic: man.getConfigString("kafkarest.result_topic"),
			ProxyHost:   man.getConfigString("kafkarest.proxyhost"),
			Timeout:     man.getConfigInt("kafkarest.timeout"),
		},
		License: LicenseConfig{
			Key: man.getConfigString("license.key"),
		},
		Vulnerabilities: VulnerabilitiesConfig{
			DatabasesPath:         man.getConfigString("vulnerabilities.databases_path"),
			Periodicity:           man.getConfigDuration("vulnerabilities.periodicity"),
			CPEDatabaseURL:        man.getConfigString("vulnerabilities.cpe_database_url"),
			CVEFeedPrefixURL:      man.getConfigString("vulnerabilities.cve_feed_prefix_url"),
			CurrentInstanceChecks: man.getConfigString("vulnerabilities.current_instance_checks"),
			DisableDataSync:       man.getConfigBool("vulnerabilities.disable_data_sync"),
		},
		Upgrades: UpgradesConfig{
			AllowMissingMigrations: man.getConfigBool("upgrades.allow_missing_migrations"),
		},
	}
}

// IsSet determines whether a given config key has been explicitly set by any
// of the configuration sources. If false, the default value is being used.
func (man Manager) IsSet(key string) bool {
	return man.viper.IsSet(key)
}

// envNameFromConfigKey converts a config key into the corresponding
// environment variable name
func envNameFromConfigKey(key string) string {
	return envPrefix + "_" + strings.ToUpper(strings.Replace(key, ".", "_", -1))
}

// flagNameFromConfigKey converts a config key into the corresponding flag name
func flagNameFromConfigKey(key string) string {
	return strings.Replace(key, ".", "_", -1)
}

// Manager manages the addition and retrieval of config values for Fleet
// configs. It's only public API method is LoadConfig, which will return the
// populated FleetConfig struct.
type Manager struct {
	viper    *viper.Viper
	command  *cobra.Command
	defaults map[string]interface{}
}

// NewManager initializes a Manager wrapping the provided cobra
// command. All config flags will be attached to that command (and inherited by
// the subcommands). Typically this should be called just once, with the root
// command.
func NewManager(command *cobra.Command) Manager {
	man := Manager{
		viper:    viper.New(),
		command:  command,
		defaults: map[string]interface{}{},
	}
	man.addConfigs()
	return man
}

// addDefault will check for duplication, then add a default value to the
// defaults map
func (man Manager) addDefault(key string, defVal interface{}) {
	if _, exists := man.defaults[key]; exists {
		panic("Trying to add duplicate config for key " + key)
	}

	man.defaults[key] = defVal
}

func getFlagUsage(key string, usage string) string {
	return fmt.Sprintf("Env: %s\n\t\t%s", envNameFromConfigKey(key), usage)
}

// getInterfaceVal is a helper function used by the getConfig* functions to
// retrieve the config value as interface{}, which will then be cast to the
// appropriate type by the getConfig* function.
func (man Manager) getInterfaceVal(key string) interface{} {
	interfaceVal := man.viper.Get(key)
	if interfaceVal == nil {
		var ok bool
		interfaceVal, ok = man.defaults[key]
		if !ok {
			panic("Tried to look up default value for nonexistent config option: " + key)
		}
	}
	return interfaceVal
}

// addConfigString adds a string config to the config options
func (man Manager) addConfigString(key, defVal, usage string) {
	man.command.PersistentFlags().String(flagNameFromConfigKey(key), defVal, getFlagUsage(key, usage))
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key)))
	man.viper.BindEnv(key, envNameFromConfigKey(key))

	// Add default
	man.addDefault(key, defVal)
}

// getConfigString retrieves a string from the loaded config
func (man Manager) getConfigString(key string) string {
	interfaceVal := man.getInterfaceVal(key)
	stringVal, err := cast.ToStringE(interfaceVal)
	if err != nil {
		panic("Unable to cast to string for key " + key + ": " + err.Error())
	}

	return stringVal
}

// Custom handling for TLSProfile which can only accept specific values
// for the argument
func (man Manager) getConfigTLSProfile() string {
	ival := man.getInterfaceVal(TLSProfileKey)
	sval, err := cast.ToStringE(ival)
	if err != nil {
		panic(fmt.Sprintf("%s requires a string value: %s", TLSProfileKey, err.Error()))
	}
	switch sval {
	case TLSProfileModern, TLSProfileIntermediate:
	default:
		panic(fmt.Sprintf("%s must be one of %s or %s", TLSProfileKey,
			TLSProfileModern, TLSProfileIntermediate))
	}
	return sval
}

// addConfigInt adds a int config to the config options
func (man Manager) addConfigInt(key string, defVal int, usage string) {
	man.command.PersistentFlags().Int(flagNameFromConfigKey(key), defVal, getFlagUsage(key, usage))
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key)))
	man.viper.BindEnv(key, envNameFromConfigKey(key))

	// Add default
	man.addDefault(key, defVal)
}

// getConfigInt retrieves a int from the loaded config
func (man Manager) getConfigInt(key string) int {
	interfaceVal := man.getInterfaceVal(key)
	intVal, err := cast.ToIntE(interfaceVal)
	if err != nil {
		panic("Unable to cast to int for key " + key + ": " + err.Error())
	}

	return intVal
}

// addConfigBool adds a bool config to the config options
func (man Manager) addConfigBool(key string, defVal bool, usage string) {
	man.command.PersistentFlags().Bool(flagNameFromConfigKey(key), defVal, getFlagUsage(key, usage))
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key)))
	man.viper.BindEnv(key, envNameFromConfigKey(key))

	// Add default
	man.addDefault(key, defVal)
}

// getConfigBool retrieves a bool from the loaded config
func (man Manager) getConfigBool(key string) bool {
	interfaceVal := man.getInterfaceVal(key)
	boolVal, err := cast.ToBoolE(interfaceVal)
	if err != nil {
		panic("Unable to cast to bool for key " + key + ": " + err.Error())
	}

	return boolVal
}

// addConfigDuration adds a duration config to the config options
func (man Manager) addConfigDuration(key string, defVal time.Duration, usage string) {
	man.command.PersistentFlags().Duration(flagNameFromConfigKey(key), defVal, getFlagUsage(key, usage))
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key)))
	man.viper.BindEnv(key, envNameFromConfigKey(key))

	// Add default
	man.addDefault(key, defVal)
}

// getConfigDuration retrieves a duration from the loaded config
func (man Manager) getConfigDuration(key string) time.Duration {
	interfaceVal := man.getInterfaceVal(key)
	durationVal, err := cast.ToDurationE(interfaceVal)
	if err != nil {
		panic("Unable to cast to duration for key " + key + ": " + err.Error())
	}

	return durationVal
}

// loadConfigFile handles the loading of the config file.
func (man Manager) loadConfigFile() {
	man.viper.SetConfigType("yaml")

	configFile := man.command.PersistentFlags().Lookup("config").Value.String()

	if configFile == "" {
		// No config file set, only use configs from env
		// vars/flags/defaults
		return
	}

	man.viper.SetConfigFile(configFile)
	err := man.viper.ReadInConfig()
	if err != nil {
		fmt.Println("Error loading config file:", err)
		os.Exit(1)
	}

	fmt.Println("Using config file: ", man.viper.ConfigFileUsed())
}

// TestConfig returns a barebones configuration suitable for use in tests.
// Individual tests may want to override some of the values provided.
func TestConfig() FleetConfig {
	testLogFile := "/dev/null"
	if runtime.GOOS == "windows" {
		testLogFile = "NUL"
	}
	return FleetConfig{
		App: AppConfig{
			TokenKeySize:              24,
			InviteTokenValidityPeriod: 5 * 24 * time.Hour,
		},
		Auth: AuthConfig{
			BcryptCost:  6, // Low cost keeps tests fast
			SaltKeySize: 24,
		},
		Session: SessionConfig{
			KeySize:  64,
			Duration: 24 * 90 * time.Hour,
		},
		Osquery: OsqueryConfig{
			NodeKeySize:          24,
			HostIdentifier:       "instance",
			EnrollCooldown:       42 * time.Minute,
			StatusLogPlugin:      "filesystem",
			ResultLogPlugin:      "filesystem",
			LabelUpdateInterval:  1 * time.Hour,
			PolicyUpdateInterval: 1 * time.Hour,
			DetailUpdateInterval: 1 * time.Hour,
			MaxJitterPercent:     0,
		},
		Logging: LoggingConfig{
			Debug:         true,
			DisableBanner: true,
		},
		Filesystem: FilesystemConfig{
			StatusLogFile: testLogFile,
			ResultLogFile: testLogFile,
		},
	}
}
