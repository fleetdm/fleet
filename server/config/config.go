package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	nanodep_client "github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/tokenpki"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	envPrefix = "FLEET"
)

// MysqlConfig defines configs related to MySQL
type MysqlConfig struct {
	Protocol        string `yaml:"protocol"`
	Address         string `yaml:"address"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	PasswordPath    string `yaml:"password_path"`
	Database        string `yaml:"database"`
	TLSCert         string `yaml:"tls_cert"`
	TLSKey          string `yaml:"tls_key"`
	TLSCA           string `yaml:"tls_ca"`
	TLSServerName   string `yaml:"tls_server_name"`
	TLSConfig       string `yaml:"tls_config"` // tls=customValue in DSN
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
	SQLMode         string `yaml:"sql_mode"`
}

// RedisConfig defines configs related to Redis
type RedisConfig struct {
	Address                   string
	Username                  string
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
	MaxIdleConns              int           `yaml:"max_idle_conns"`
	MaxOpenConns              int           `yaml:"max_open_conns"`
	// this config is an int on MysqlConfig, but it should be a time.Duration.
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ConnWaitTimeout time.Duration `yaml:"conn_wait_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
}

const (
	TLSProfileKey          = "server.tls_compatibility"
	TLSProfileModern       = "modern"
	TLSProfileIntermediate = "intermediate"
)

// ServerConfig defines configs related to the Fleet server
type ServerConfig struct {
	Address                     string
	Cert                        string
	Key                         string
	TLS                         bool
	TLSProfile                  string `yaml:"tls_compatibility"`
	URLPrefix                   string `yaml:"url_prefix"`
	Keepalive                   bool   `yaml:"keepalive"`
	SandboxEnabled              bool   `yaml:"sandbox_enabled"`
	WebsocketsAllowUnsafeOrigin bool   `yaml:"websockets_allow_unsafe_origin"`
}

func (s *ServerConfig) DefaultHTTPServer(ctx context.Context, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              s.Address,
		Handler:           handler,
		ReadTimeout:       25 * time.Second,
		WriteTimeout:      40 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       5 * time.Minute,
		MaxHeaderBytes:    1 << 18, // 0.25 MB (262144 bytes)
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
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
	EnableScheduledQueryStats bool          `yaml:"enable_scheduled_query_stats"`
}

// SessionConfig defines configs related to user sessions
type SessionConfig struct {
	KeySize  int `yaml:"key_size"`
	Duration time.Duration
}

// OsqueryConfig defines configs related to osquery
type OsqueryConfig struct {
	NodeKeySize          int           `yaml:"node_key_size"`
	HostIdentifier       string        `yaml:"host_identifier"`
	EnrollCooldown       time.Duration `yaml:"enroll_cooldown"`
	StatusLogPlugin      string        `yaml:"status_log_plugin"`
	ResultLogPlugin      string        `yaml:"result_log_plugin"`
	LabelUpdateInterval  time.Duration `yaml:"label_update_interval"`
	PolicyUpdateInterval time.Duration `yaml:"policy_update_interval"`
	DetailUpdateInterval time.Duration `yaml:"detail_update_interval"`

	// StatusLogFile is deprecated. It was replaced by FilesystemConfig.StatusLogFile.
	//
	// TODO(lucas): We should at least add a warning if this field is populated.
	StatusLogFile string `yaml:"status_log_file"`
	// ResultLogFile is deprecated. It was replaced by FilesystemConfig.ResultLogFile.
	//
	// TODO(lucas): We should at least add a warning if this field is populated.
	ResultLogFile string `yaml:"result_log_file"`

	EnableLogRotation                bool          `yaml:"enable_log_rotation"`
	MaxJitterPercent                 int           `yaml:"max_jitter_percent"`
	EnableAsyncHostProcessing        string        `yaml:"enable_async_host_processing"` // true/false or per-task
	AsyncHostCollectInterval         string        `yaml:"async_host_collect_interval"`  // duration or per-task
	AsyncHostCollectMaxJitterPercent int           `yaml:"async_host_collect_max_jitter_percent"`
	AsyncHostCollectLockTimeout      string        `yaml:"async_host_collect_lock_timeout"` // duration or per-task
	AsyncHostCollectLogStatsInterval time.Duration `yaml:"async_host_collect_log_stats_interval"`
	AsyncHostInsertBatch             int           `yaml:"async_host_insert_batch"`
	AsyncHostDeleteBatch             int           `yaml:"async_host_delete_batch"`
	AsyncHostUpdateBatch             int           `yaml:"async_host_update_batch"`
	AsyncHostRedisPopCount           int           `yaml:"async_host_redis_pop_count"`
	AsyncHostRedisScanKeysCount      int           `yaml:"async_host_redis_scan_keys_count"`
	MinSoftwareLastOpenedAtDiff      time.Duration `yaml:"min_software_last_opened_at_diff"`
}

// AsyncTaskName is the type of names that identify tasks supporting
// asynchronous execution.
type AsyncTaskName string

// List of names for supported async tasks.
const (
	AsyncTaskLabelMembership     AsyncTaskName = "label_membership"
	AsyncTaskPolicyMembership    AsyncTaskName = "policy_membership"
	AsyncTaskHostLastSeen        AsyncTaskName = "host_last_seen"
	AsyncTaskScheduledQueryStats AsyncTaskName = "scheduled_query_stats"
)

var knownAsyncTasks = map[AsyncTaskName]struct{}{
	AsyncTaskLabelMembership:     {},
	AsyncTaskPolicyMembership:    {},
	AsyncTaskHostLastSeen:        {},
	AsyncTaskScheduledQueryStats: {},
}

// AsyncConfigForTask returns the applicable configuration for the specified
// async task.
func (o OsqueryConfig) AsyncConfigForTask(name AsyncTaskName) AsyncProcessingConfig {
	strName := string(name)
	return AsyncProcessingConfig{
		Enabled:                 configForKeyOrBool("osquery.enable_async_host_processing", strName, o.EnableAsyncHostProcessing, false),
		CollectInterval:         configForKeyOrDuration("osquery.async_host_collect_interval", strName, o.AsyncHostCollectInterval, 30*time.Second),
		CollectMaxJitterPercent: o.AsyncHostCollectMaxJitterPercent,
		CollectLockTimeout:      configForKeyOrDuration("osquery.async_host_collect_lock_timeout", strName, o.AsyncHostCollectLockTimeout, 1*time.Minute),
		CollectLogStatsInterval: o.AsyncHostCollectLogStatsInterval,
		InsertBatch:             o.AsyncHostInsertBatch,
		DeleteBatch:             o.AsyncHostDeleteBatch,
		UpdateBatch:             o.AsyncHostUpdateBatch,
		RedisPopCount:           o.AsyncHostRedisPopCount,
		RedisScanKeysCount:      o.AsyncHostRedisScanKeysCount,
	}
}

// AsyncProcessingConfig is the configuration for a specific async task.
type AsyncProcessingConfig struct {
	Enabled                 bool
	CollectInterval         time.Duration
	CollectMaxJitterPercent int
	CollectLockTimeout      time.Duration
	CollectLogStatsInterval time.Duration
	InsertBatch             int
	DeleteBatch             int
	UpdateBatch             int
	RedisPopCount           int
	RedisScanKeysCount      int
}

// LoggingConfig defines configs related to logging
type LoggingConfig struct {
	Debug                bool
	JSON                 bool
	DisableBanner        bool          `yaml:"disable_banner"`
	ErrorRetentionPeriod time.Duration `yaml:"error_retention_period"`
	TracingEnabled       bool          `yaml:"tracing_enabled"`
	// TracingType can either be opentelemetry or elasticapm for whichever type of tracing wanted
	TracingType string `yaml:"tracing_type"`
}

// ActivityConfig defines configs related to activities.
type ActivityConfig struct {
	// EnableAuditLog enables logging for audit activities.
	EnableAuditLog bool `yaml:"enable_audit_log"`
	// AuditLogPlugin sets the plugin to use to log activities.
	AuditLogPlugin string `yaml:"audit_log_plugin"`
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
	AuditStream      string `yaml:"audit_stream"`
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
	AuditStream      string `yaml:"audit_stream"`
}

// SESConfig defines configs for the AWS SES service for emailing
type SESConfig struct {
	Region           string
	EndpointURL      string `yaml:"endpoint_url"`
	AccessKeyID      string `yaml:"access_key_id"`
	SecretAccessKey  string `yaml:"secret_access_key"`
	StsAssumeRoleArn string `yaml:"sts_assume_role_arn"`
	SourceArn        string `yaml:"source_arn"`
}

type EmailConfig struct {
	EmailBackend string `yaml:"backend"`
}

// LambdaConfig defines configs for the AWS Lambda logging plugin
type LambdaConfig struct {
	Region           string
	AccessKeyID      string `yaml:"access_key_id"`
	SecretAccessKey  string `yaml:"secret_access_key"`
	StsAssumeRoleArn string `yaml:"sts_assume_role_arn"`
	StatusFunction   string `yaml:"status_function"`
	ResultFunction   string `yaml:"result_function"`
	AuditFunction    string `yaml:"audit_function"`
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
	AuditTopic    string `json:"audit_topic" yaml:"audit_topic"`
	AddAttributes bool   `json:"add_attributes" yaml:"add_attributes"`
}

// FilesystemConfig defines configs for the Filesystem logging plugin
type FilesystemConfig struct {
	StatusLogFile        string `json:"status_log_file" yaml:"status_log_file"`
	ResultLogFile        string `json:"result_log_file" yaml:"result_log_file"`
	AuditLogFile         string `json:"audit_log_file" yaml:"audit_log_file"`
	EnableLogRotation    bool   `json:"enable_log_rotation" yaml:"enable_log_rotation"`
	EnableLogCompression bool   `json:"enable_log_compression" yaml:"enable_log_compression"`
	MaxSize              int    `json:"max_size" yaml:"max_size"`
	MaxAge               int    `json:"max_age" yaml:"max_age"`
	MaxBackups           int    `json:"max_backups" yaml:"max_backups"`
}

// KafkaRESTConfig defines configs for the Kafka REST Proxy logging plugin.
type KafkaRESTConfig struct {
	StatusTopic      string `json:"status_topic" yaml:"status_topic"`
	ResultTopic      string `json:"result_topic" yaml:"result_topic"`
	AuditTopic       string `json:"audit_topic" yaml:"audit_topic"`
	ProxyHost        string `json:"proxyhost" yaml:"proxyhost"`
	ContentTypeValue string `json:"content_type_value" yaml:"content_type_value"`
	Timeout          int    `json:"timeout" yaml:"timeout"`
}

// LicenseConfig defines configs related to licensing Fleet.
type LicenseConfig struct {
	Key              string `yaml:"key"`
	EnforceHostLimit bool   `yaml:"enforce_host_limit"`
}

// VulnerabilitiesConfig defines configs related to vulnerability processing within Fleet.
type VulnerabilitiesConfig struct {
	DatabasesPath               string        `json:"databases_path" yaml:"databases_path"`
	Periodicity                 time.Duration `json:"periodicity" yaml:"periodicity"`
	CPEDatabaseURL              string        `json:"cpe_database_url" yaml:"cpe_database_url"`
	CPETranslationsURL          string        `json:"cpe_translations_url" yaml:"cpe_translations_url"`
	CVEFeedPrefixURL            string        `json:"cve_feed_prefix_url" yaml:"cve_feed_prefix_url"`
	CurrentInstanceChecks       string        `json:"current_instance_checks" yaml:"current_instance_checks"`
	DisableSchedule             bool          `json:"disable_schedule" yaml:"disable_schedule"`
	DisableDataSync             bool          `json:"disable_data_sync" yaml:"disable_data_sync"`
	RecentVulnerabilityMaxAge   time.Duration `json:"recent_vulnerability_max_age" yaml:"recent_vulnerability_max_age"`
	DisableWinOSVulnerabilities bool          `json:"disable_win_os_vulnerabilities" yaml:"disable_win_os_vulnerabilities"`
}

// UpgradesConfig defines configs related to fleet server upgrades.
type UpgradesConfig struct {
	AllowMissingMigrations bool `json:"allow_missing_migrations" yaml:"allow_missing_migrations"`
}

type SentryConfig struct {
	Dsn string `json:"dsn"`
}

type GeoIPConfig struct {
	DatabasePath string `json:"database_path" yaml:"database_path"`
}

// PrometheusConfig holds the configuration for Fleet's prometheus metrics.
type PrometheusConfig struct {
	// BasicAuth is the HTTP Basic BasicAuth configuration.
	BasicAuth HTTPBasicAuthConfig `json:"basic_auth" yaml:"basic_auth"`
}

// HTTPBasicAuthConfig holds configuration for HTTP Basic Auth.
type HTTPBasicAuthConfig struct {
	// Username is the HTTP Basic Auth username.
	Username string `json:"username" yaml:"username"`
	// Password is the HTTP Basic Auth password.
	Password string `json:"password" yaml:"password"`
	// Disable allows running the Prometheus metrics endpoint without Basic Auth.
	Disable bool `json:"disable" yaml:"disable"`
}

// PackagingConfig holds configuration to build and retrieve Fleet packages
type PackagingConfig struct {
	// GlobalEnrollSecret is the enroll secret that will be used to enroll
	// hosts in the global scope
	GlobalEnrollSecret string `yaml:"global_enroll_secret"`
	// S3 configuration used to retrieve pre-built installers
	S3 S3Config `yaml:"s3"`
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
	Activity         ActivityConfig
	Logging          LoggingConfig
	Firehose         FirehoseConfig
	Kinesis          KinesisConfig
	Lambda           LambdaConfig
	S3               S3Config
	Email            EmailConfig
	SES              SESConfig
	PubSub           PubSubConfig
	Filesystem       FilesystemConfig
	KafkaREST        KafkaRESTConfig
	License          LicenseConfig
	Vulnerabilities  VulnerabilitiesConfig
	Upgrades         UpgradesConfig
	Sentry           SentryConfig
	GeoIP            GeoIPConfig
	Prometheus       PrometheusConfig
	Packaging        PackagingConfig
	MDM              MDMConfig
}

type MDMConfig struct {
	AppleAPNsCert      string `yaml:"apple_apns_cert"`
	AppleAPNsCertBytes string `yaml:"apple_apns_cert_bytes"`
	AppleAPNsKey       string `yaml:"apple_apns_key"`
	AppleAPNsKeyBytes  string `yaml:"apple_apns_key_bytes"`
	AppleSCEPCert      string `yaml:"apple_scep_cert"`
	AppleSCEPCertBytes string `yaml:"apple_scep_cert_bytes"`
	AppleSCEPKey       string `yaml:"apple_scep_key"`
	AppleSCEPKeyBytes  string `yaml:"apple_scep_key_bytes"`

	// the following fields hold the parsed, validated TLS certificate set the
	// first time AppleAPNs or AppleSCEP is called, as well as the PEM-encoded
	// bytes for the certificate and private key.
	appleAPNs        *tls.Certificate
	appleAPNsPEMCert []byte
	appleAPNsPEMKey  []byte
	appleSCEP        *tls.Certificate
	appleSCEPPEMCert []byte
	appleSCEPPEMKey  []byte

	AppleBMServerToken      string `yaml:"apple_bm_server_token"`
	AppleBMServerTokenBytes string `yaml:"apple_bm_server_token_bytes"`
	AppleBMCert             string `yaml:"apple_bm_cert"`
	AppleBMCertBytes        string `yaml:"apple_bm_cert_bytes"`
	AppleBMKey              string `yaml:"apple_bm_key"`
	AppleBMKeyBytes         string `yaml:"apple_bm_key_bytes"`

	// the following fields hold the decrypted, validated Apple BM token set the
	// first time AppleBM is called.
	appleBMToken *nanodep_client.OAuth1Tokens

	// AppleEnable enables Apple MDM functionality on Fleet.
	AppleEnable bool `yaml:"apple_enable"`
	// AppleDEPSyncPeriodicity is the duration between DEP device syncing
	// (fetching and setting of DEP profiles).
	AppleDEPSyncPeriodicity time.Duration `yaml:"apple_dep_sync_periodicity"`
	// AppleSCEPChallenge is the SCEP challenge for SCEP enrollment requests.
	AppleSCEPChallenge string `yaml:"apple_scep_challenge"`
	// AppleSCEPSignerValidityDays are the days signed client certificates will
	// be valid.
	AppleSCEPSignerValidityDays int `yaml:"apple_scep_signer_validity_days"`
	// AppleSCEPSignerAllowRenewalDays are the allowable renewal days for
	// certificates.
	AppleSCEPSignerAllowRenewalDays int `yaml:"apple_scep_signer_allow_renewal_days"`

	// WindowsWSTEPIdentityCert is the path to the certificate used to sign
	// WSTEP responses.
	WindowsWSTEPIdentityCert string `yaml:"windows_wstep_identity_cert"`
	// WindowsWSTEPIdentityCertBytes is the content of the certificate used to sign
	// WSTEP responses.
	WindowsWSTEPIdentityCertBytes string `yaml:"windows_wstep_identity_cert_bytes"`
	// WindowsWSTEPIdentityKey is the path to the private key used to sign
	// WSTEP responses.
	WindowsWSTEPIdentityKey string `yaml:"windows_wstep_identity_key"`
	// WindowsWSTEPIdentityKey is the content of the private key used to sign
	// WSTEP responses.
	WindowsWSTEPIdentityKeyBytes string `yaml:"windows_wstep_identity_key_bytes"`

	// the following fields hold the parsed, validated TLS certificate set the
	// first time Microsoft WSTEP is called, as well as the PEM-encoded
	// bytes for the certificate and private key.
	microsoftWSTEP        *tls.Certificate
	microsoftWSTEPCertPEM []byte
	microsoftWSTEPKeyPEM  []byte
}

type x509KeyPairConfig struct {
	certPath  string
	certBytes []byte
	keyPath   string
	keyBytes  []byte
}

func (x *x509KeyPairConfig) IsSet() bool {
	// if any setting is provided, then the key pair is considered set
	return x.certPath != "" || len(x.certBytes) != 0 || x.keyPath != "" || len(x.keyBytes) != 0
}

func (x *x509KeyPairConfig) Parse(keepLeaf bool) (*tls.Certificate, error) {
	if x.certPath == "" && len(x.certBytes) == 0 {
		return nil, errors.New("no certificate provided")
	}
	if x.certPath != "" && len(x.certBytes) != 0 {
		return nil, errors.New("only one of the certificate path or bytes must be provided")
	}
	if x.keyPath == "" && len(x.keyBytes) == 0 {
		return nil, errors.New("no key provided")
	}
	if x.keyPath != "" && len(x.keyBytes) != 0 {
		return nil, errors.New("only one of the key path or bytes must be provided")
	}

	if len(x.certBytes) == 0 {
		b, err := os.ReadFile(x.certPath)
		if err != nil {
			return nil, fmt.Errorf("reading certificate file: %w", err)
		}
		x.certBytes = b
	}
	if len(x.keyBytes) == 0 {
		b, err := os.ReadFile(x.keyPath)
		if err != nil {
			return nil, fmt.Errorf("reading key file: %w", err)
		}
		x.keyBytes = b
	}

	cert, err := tls.X509KeyPair(x.certBytes, x.keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse key pair: %w", err)
	}

	if keepLeaf {
		// X509KeyPair does not store the parsed certificate leaf
		parsed, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, fmt.Errorf("parse leaf certificate: %w", err)
		}
		cert.Leaf = parsed
	}
	return &cert, nil
}

func (m *MDMConfig) IsAppleAPNsSet() bool {
	pair := x509KeyPairConfig{
		m.AppleAPNsCert,
		[]byte(m.AppleAPNsCertBytes),
		m.AppleAPNsKey,
		[]byte(m.AppleAPNsKeyBytes),
	}
	return pair.IsSet()
}

func (m *MDMConfig) IsAppleSCEPSet() bool {
	pair := x509KeyPairConfig{
		m.AppleSCEPCert,
		[]byte(m.AppleSCEPCertBytes),
		m.AppleSCEPKey,
		[]byte(m.AppleSCEPKeyBytes),
	}
	return pair.IsSet()
}

func (m *MDMConfig) IsAppleBMSet() bool {
	pair := x509KeyPairConfig{
		m.AppleBMCert,
		[]byte(m.AppleBMCertBytes),
		m.AppleBMKey,
		[]byte(m.AppleBMKeyBytes),
	}
	// the BM token options is not taken into account by pair.IsSet
	return pair.IsSet() || m.AppleBMServerToken != "" || m.AppleBMServerTokenBytes != ""
}

// AppleAPNs returns the parsed and validated TLS certificate for Apple APNs.
// It parses and validates it if it hasn't been done yet.
func (m *MDMConfig) AppleAPNs() (cert *tls.Certificate, pemCert, pemKey []byte, err error) {
	if m.appleAPNs == nil {
		pair := x509KeyPairConfig{
			m.AppleAPNsCert,
			[]byte(m.AppleAPNsCertBytes),
			m.AppleAPNsKey,
			[]byte(m.AppleAPNsKeyBytes),
		}
		cert, err := pair.Parse(true)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Apple MDM APNs configuration: %w", err)
		}
		m.appleAPNs = cert
		m.appleAPNsPEMCert = pair.certBytes
		m.appleAPNsPEMKey = pair.keyBytes
	}
	return m.appleAPNs, m.appleAPNsPEMCert, m.appleAPNsPEMKey, nil
}

func (m *MDMConfig) AppleAPNsTopic() (string, error) {
	apnsCert, _, _, err := m.AppleAPNs()
	if err != nil {
		return "", fmt.Errorf("parsing APNs certificates: %w", err)
	}

	mdmPushCertTopic, err := cryptoutil.TopicFromCert(apnsCert.Leaf)
	if err != nil {
		return "", fmt.Errorf("extracting topic from APNs certificate: %w", err)
	}

	return mdmPushCertTopic, nil
}

// AppleSCEP returns the parsed and validated TLS certificate for Apple SCEP.
// It parses and validates it if it hasn't been done yet.
func (m *MDMConfig) AppleSCEP() (cert *tls.Certificate, pemCert, pemKey []byte, err error) {
	if m.appleSCEP == nil {
		pair := x509KeyPairConfig{
			m.AppleSCEPCert,
			[]byte(m.AppleSCEPCertBytes),
			m.AppleSCEPKey,
			[]byte(m.AppleSCEPKeyBytes),
		}
		cert, err := pair.Parse(true)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Apple MDM SCEP configuration: %w", err)
		}
		m.appleSCEP = cert
		m.appleSCEPPEMCert = pair.certBytes
		m.appleSCEPPEMKey = pair.keyBytes
	}
	return m.appleSCEP, m.appleSCEPPEMCert, m.appleSCEPPEMKey, nil
}

// AppleBM returns the parsed, validated and decrypted server token for Apple
// Business Manager. It also parses and validates the Apple BM certificate and
// private key in the process, in order to decrypt the token.
func (m *MDMConfig) AppleBM() (tok *nanodep_client.OAuth1Tokens, err error) {
	if m.appleBMToken == nil {
		pair := x509KeyPairConfig{
			m.AppleBMCert,
			[]byte(m.AppleBMCertBytes),
			m.AppleBMKey,
			[]byte(m.AppleBMKeyBytes),
		}
		cert, err := pair.Parse(true)
		if err != nil {
			return nil, fmt.Errorf("Apple BM configuration: %w", err)
		}
		encToken, err := m.loadAppleBMEncryptedToken()
		if err != nil {
			return nil, fmt.Errorf("Apple BM configuration: %w", err)
		}
		bmKey, err := tokenpki.RSAKeyFromPEM(pair.keyBytes)
		if err != nil {
			return nil, fmt.Errorf("Apple BM configuration: parse private key: %w", err)
		}
		token, err := tokenpki.DecryptTokenJSON(encToken, cert.Leaf, bmKey)
		if err != nil {
			return nil, fmt.Errorf("Apple BM configuration: decrypt token: %w", err)
		}
		var jsonTok nanodep_client.OAuth1Tokens
		if err := json.Unmarshal(token, &jsonTok); err != nil {
			return nil, fmt.Errorf("Apple BM configuration: unmarshal JSON token: %w", err)
		}
		if jsonTok.AccessTokenExpiry.Before(time.Now()) {
			return nil, errors.New("Apple BM configuration: token is expired")
		}
		m.appleBMToken = &jsonTok
	}
	return m.appleBMToken, nil
}

func (m *MDMConfig) loadAppleBMEncryptedToken() ([]byte, error) {
	if m.AppleBMServerToken == "" && m.AppleBMServerTokenBytes == "" {
		return nil, errors.New("no token provided")
	}
	if m.AppleBMServerToken != "" && m.AppleBMServerTokenBytes != "" {
		return nil, errors.New("only one of the token path or bytes must be provided")
	}

	tokBytes := []byte(m.AppleBMServerTokenBytes)
	if m.AppleBMServerTokenBytes == "" {
		b, err := os.ReadFile(m.AppleBMServerToken)
		if err != nil {
			return nil, fmt.Errorf("reading token file: %w", err)
		}
		tokBytes = b
	}
	return tokBytes, nil
}

// MicrosoftWSTEP returns the parsed and validated TLS certificate for Microsoft WSTEP.
// It parses and validates it if it hasn't been done yet.
func (m *MDMConfig) MicrosoftWSTEP() (cert *tls.Certificate, pemCert, pemKey []byte, err error) {
	if m.microsoftWSTEP == nil {
		pair := x509KeyPairConfig{
			m.WindowsWSTEPIdentityCert,
			[]byte(m.WindowsWSTEPIdentityCertBytes),
			m.WindowsWSTEPIdentityKey,
			[]byte(m.WindowsWSTEPIdentityKeyBytes),
		}
		cert, err := pair.Parse(true)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Microsoft MDM WSTEP configuration: %w", err)
		}
		m.microsoftWSTEP = cert
		m.microsoftWSTEPCertPEM = pair.certBytes
		m.microsoftWSTEPKeyPEM = pair.keyBytes
	}
	return m.microsoftWSTEP, m.microsoftWSTEPCertPEM, m.microsoftWSTEPKeyPEM, nil
}

func (m *MDMConfig) IsMicrosoftWSTEPSet() bool {
	pair := x509KeyPairConfig{
		m.WindowsWSTEPIdentityCert,
		[]byte(m.WindowsWSTEPIdentityCertBytes),
		m.WindowsWSTEPIdentityKey,
		[]byte(m.WindowsWSTEPIdentityKeyBytes),
	}
	return pair.IsSet()
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
		pem, err := os.ReadFile(t.TLSCA)
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
		man.addConfigString(prefix+".sql_mode", "", "MySQL sql_mode"+usageSuffix)
	}
	// MySQL
	addMysqlConfig("mysql", "localhost:3306", ".")
	addMysqlConfig("mysql_read_replica", "", " for the read replica.")

	// Redis
	man.addConfigString("redis.address", "localhost:6379",
		"Redis server address (host:port)")
	man.addConfigString("redis.username", "",
		"Redis server username")
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
	man.addConfigDuration("redis.write_timeout", 10*time.Second, "Redis maximum amount of time to wait for a write (send) on a connection")
	man.addConfigDuration("redis.read_timeout", 10*time.Second, "Redis maximum amount of time to wait for a read (receive) on a connection")

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
		"Controls whether HTTP keep-alives are enabled.")
	man.addConfigBool("server.sandbox_enabled", false,
		"When enabled, Fleet limits some features for the Sandbox")
	man.addConfigBool("server.websockets_allow_unsafe_origin", false, "Disable checking the origin header on websocket connections, this is sometimes necessary when proxies rewrite origin headers between the client and the Fleet webserver")

	// Hide the sandbox flag as we don't want it to be discoverable for users for now
	sandboxFlag := man.command.PersistentFlags().Lookup(flagNameFromConfigKey("server.sandbox_enabled"))
	if sandboxFlag != nil {
		sandboxFlag.Hidden = true
	}

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
	man.addConfigBool("app.enable_scheduled_query_stats", true,
		"If true (default) it gets scheduled query stats from hosts")

	// Session
	man.addConfigInt("session.key_size", 64,
		"Size of generated session keys")
	man.addConfigDuration("session.duration", 24*5*time.Hour,
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
	man.addConfigString("osquery.enable_async_host_processing", "false",
		"Enable asynchronous processing of host-reported query results (either 'true'/'false' or set per task, e.g., 'label_membership=true&policy_membership=true')")
	man.addConfigString("osquery.async_host_collect_interval", (30 * time.Second).String(),
		"Interval to collect asynchronous host-reported query results (e.g. '30s' or set per task 'label_membership=10s&policy_membership=1m')")
	man.addConfigInt("osquery.async_host_collect_max_jitter_percent", 10,
		"Maximum percentage of the interval to collect asynchronous host results")
	man.addConfigString("osquery.async_host_collect_lock_timeout", (1 * time.Minute).String(),
		"Timeout of the exclusive lock held during async host collection (e.g., '30s' or set per task 'label_membership=10s&policy_membership=1m'")
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
	man.addConfigDuration("osquery.min_software_last_opened_at_diff", 1*time.Hour,
		"Minimum time difference of the software's last opened timestamp (compared to the last one saved) to trigger an update to the database")

	// Activities
	man.addConfigBool("activity.enable_audit_log", false,
		"Enable audit logs")
	man.addConfigString("activity.audit_log_plugin", "filesystem",
		"Log plugin to use for audit logs")

	// Logging
	man.addConfigBool("logging.debug", false,
		"Enable debug logging")
	man.addConfigBool("logging.json", false,
		"Log in JSON format")
	man.addConfigBool("logging.disable_banner", false,
		"Disable startup banner")
	man.addConfigDuration("logging.error_retention_period", 24*time.Hour,
		"Amount of time to keep errors, 0 means no expiration, < 0 means disable storage of errors")
	man.addConfigBool("logging.tracing_enabled", false,
		"Enable Tracing, further configured via standard env variables")
	man.addConfigString("logging.tracing_type", "opentelemetry",
		"Select the kind of tracing, defaults to opentelemetry, can also be elasticapm")

	// Email
	man.addConfigString("email.backend", "", "Provide the email backend type, acceptable values are currently \"ses\" and \"default\" or empty string which will default to SMTP")
	// SES
	man.addConfigString("ses.region", "", "AWS Region to use")
	man.addConfigString("ses.endpoint_url", "", "AWS Service Endpoint to use (leave empty for default service endpoints)")
	man.addConfigString("ses.access_key_id", "", "Access Key ID for AWS authentication")
	man.addConfigString("ses.secret_access_key", "", "Secret Access Key for AWS authentication")
	man.addConfigString("ses.sts_assume_role_arn", "", "ARN of role to assume for AWS")
	man.addConfigString("ses.source_arn", "", "ARN of the identity that is associated with the sending authorization policy that permits you to send for the email address specified in the Source parameter")

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
	man.addConfigString("firehose.audit_stream", "",
		"Firehose stream name for audit logs")

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
	man.addConfigString("kinesis.audit_stream", "",
		"Kinesis stream name for audit logs")

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
	man.addConfigString("lambda.audit_function", "",
		"Lambda function name for audit logs")

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
	man.addConfigString("pubsub.audit_topic", "", "PubSub topic for audit logs")
	man.addConfigBool("pubsub.add_attributes", false, "Add PubSub attributes in addition to the message body")

	// Filesystem
	man.addConfigString("filesystem.status_log_file", filepath.Join(os.TempDir(), "osquery_status"),
		"Log file path to use for status logs")
	man.addConfigString("filesystem.result_log_file", filepath.Join(os.TempDir(), "osquery_result"),
		"Log file path to use for result logs")
	man.addConfigString("filesystem.audit_log_file", filepath.Join(os.TempDir(), "audit"),
		"Log file path to use for audit logs")
	man.addConfigBool("filesystem.enable_log_rotation", false,
		"Enable automatic rotation for osquery log files")
	man.addConfigBool("filesystem.enable_log_compression", false,
		"Enable compression for the rotated osquery log files")
	man.addConfigInt("filesystem.max_size", 500, "Maximum size in megabytes log files will grow until rotated (only valid if enable_log_rotation is true) default is 500MB")
	man.addConfigInt("filesystem.max_age", 28, "Maximum number of days to retain old log files based on the timestamp encoded in their filename. Setting to zero wil retain old log files indefinitely (only valid if enable_log_rotation is true) default is 28 days")
	man.addConfigInt("filesystem.max_backups", 3, "Maximum number of old log files to retain. Setting to zero will retain all old log files (only valid if enable_log_rotation is true) default is 3")

	// KafkaREST
	man.addConfigString("kafkarest.status_topic", "", "Kafka REST topic for status logs")
	man.addConfigString("kafkarest.result_topic", "", "Kafka REST topic for result logs")
	man.addConfigString("kafkarest.audit_topic", "", "Kafka REST topic for audit logs")
	man.addConfigString("kafkarest.proxyhost", "", "Kafka REST proxy host url")
	man.addConfigString("kafkarest.content_type_value", "application/vnd.kafka.json.v1+json",
		"Kafka REST proxy content type header (defaults to \"application/vnd.kafka.json.v1+json\"")
	man.addConfigInt("kafkarest.timeout", 5, "Kafka REST proxy json post timeout")

	// License
	man.addConfigString("license.key", "", "Fleet license key (to enable Fleet Premium features)")
	man.addConfigBool("license.enforce_host_limit", false, "Enforce license limit of enrolled hosts")

	// Vulnerability processing
	man.addConfigString("vulnerabilities.databases_path", "/tmp/vulndbs",
		"Path where Fleet will download the data feeds to check CVEs")
	man.addConfigDuration("vulnerabilities.periodicity", 1*time.Hour,
		"How much time to wait between processing software for vulnerabilities.")
	man.addConfigString("vulnerabilities.cpe_database_url", "",
		"URL from which to get the latest CPE database. If empty, it will be downloaded from the latest release available at https://github.com/fleetdm/nvd/releases.")
	man.addConfigString("vulnerabilities.cpe_translations_url", "",
		"URL from which to get the latest CPE translations. If empty, it will be downloaded from the latest release available at https://github.com/fleetdm/nvd/releases.")
	man.addConfigString("vulnerabilities.cve_feed_prefix_url", "",
		"Prefix URL for the CVE data feed. If empty, default to https://nvd.nist.gov/")
	man.addConfigString("vulnerabilities.current_instance_checks", "auto",
		"Allows to manually select an instance to do the vulnerability processing.")
	man.addConfigBool("vulnerabilities.disable_schedule", false,
		"Set this to true when the vulnerability processing job is scheduled by an external mechanism")
	man.addConfigBool("vulnerabilities.disable_data_sync", false,
		"Skips synchronizing data streams and expects them to be available in the databases_path.")
	man.addConfigDuration("vulnerabilities.recent_vulnerability_max_age", 30*24*time.Hour,
		"Maximum age of the published date of a vulnerability (CVE) to be considered 'recent'.")
	man.addConfigBool(
		"vulnerabilities.disable_win_os_vulnerabilities",
		false,
		"Don't sync installed Windows updates nor perform Windows OS vulnerability processing.",
	)

	// Upgrades
	man.addConfigBool("upgrades.allow_missing_migrations", false,
		"Allow serve to run even if migrations are missing.")

	// Sentry
	man.addConfigString("sentry.dsn", "", "DSN for Sentry")

	// GeoIP
	man.addConfigString("geoip.database_path", "", "path to mmdb file")

	// Prometheus
	man.addConfigString("prometheus.basic_auth.username", "", "Prometheus username for HTTP Basic Auth")
	man.addConfigString("prometheus.basic_auth.password", "", "Prometheus password for HTTP Basic Auth")
	man.addConfigBool("prometheus.basic_auth.disable", false, "Disable HTTP Basic Auth for Prometheus")

	// Packaging config
	man.addConfigString("packaging.global_enroll_secret", "", "Enroll secret to be used for the global domain (instead of randomly generating one)")
	man.addConfigString("packaging.s3.bucket", "", "Bucket where to retrieve installers")
	man.addConfigString("packaging.s3.prefix", "", "Prefix under which installers are stored")
	man.addConfigString("packaging.s3.region", "", "AWS Region (if blank region is derived)")
	man.addConfigString("packaging.s3.endpoint_url", "", "AWS Service Endpoint to use (leave blank for default service endpoints)")
	man.addConfigString("packaging.s3.access_key_id", "", "Access Key ID for AWS authentication")
	man.addConfigString("packaging.s3.secret_access_key", "", "Secret Access Key for AWS authentication")
	man.addConfigString("packaging.s3.sts_assume_role_arn", "", "ARN of role to assume for AWS")
	man.addConfigBool("packaging.s3.disable_ssl", false, "Disable SSL (typically for local testing)")
	man.addConfigBool("packaging.s3.force_s3_path_style", false, "Set this to true to force path-style addressing, i.e., `http://s3.amazonaws.com/BUCKET/KEY`")

	// MDM config
	man.addConfigString("mdm.apple_apns_cert", "", "Apple APNs PEM-encoded certificate path")
	man.addConfigString("mdm.apple_apns_cert_bytes", "", "Apple APNs PEM-encoded certificate bytes")
	man.addConfigString("mdm.apple_apns_key", "", "Apple APNs PEM-encoded private key path")
	man.addConfigString("mdm.apple_apns_key_bytes", "", "Apple APNs PEM-encoded private key bytes")
	man.addConfigString("mdm.apple_scep_cert", "", "Apple SCEP PEM-encoded certificate path")
	man.addConfigString("mdm.apple_scep_cert_bytes", "", "Apple SCEP PEM-encoded certificate bytes")
	man.addConfigString("mdm.apple_scep_key", "", "Apple SCEP PEM-encoded private key path")
	man.addConfigString("mdm.apple_scep_key_bytes", "", "Apple SCEP PEM-encoded private key bytes")
	man.addConfigString("mdm.apple_bm_server_token", "", "Apple Business Manager encrypted server token path (.p7m file)")
	man.addConfigString("mdm.apple_bm_server_token_bytes", "", "Apple Business Manager encrypted server token bytes")
	man.addConfigString("mdm.apple_bm_cert", "", "Apple Business Manager PEM-encoded certificate path")
	man.addConfigString("mdm.apple_bm_cert_bytes", "", "Apple Business Manager PEM-encoded certificate bytes")
	man.addConfigString("mdm.apple_bm_key", "", "Apple Business Manager PEM-encoded private key path")
	man.addConfigString("mdm.apple_bm_key_bytes", "", "Apple Business Manager PEM-encoded private key bytes")
	man.addConfigBool("mdm.apple_enable", false, "Enable MDM Apple functionality")
	man.addConfigInt("mdm.apple_scep_signer_validity_days", 365, "Days signed client certificates will be valid")
	man.addConfigInt("mdm.apple_scep_signer_allow_renewal_days", 14, "Allowable renewal days for client certificates")
	man.addConfigString("mdm.apple_scep_challenge", "", "SCEP static challenge for enrollment")
	man.addConfigDuration("mdm.apple_dep_sync_periodicity", 1*time.Minute, "How much time to wait for DEP profile assignment")
	man.addConfigString("mdm.windows_wstep_identity_cert", "", "Microsoft WSTEP PEM-encoded certificate path")
	man.addConfigString("mdm.windows_wstep_identity_key", "", "Microsoft WSTEP PEM-encoded private key path")
	man.addConfigString("mdm.windows_wstep_identity_cert_bytes", "", "Microsoft WSTEP PEM-encoded certificate bytes")
	man.addConfigString("mdm.windows_wstep_identity_key_bytes", "", "Microsoft WSTEP PEM-encoded private key bytes")

	// Hide Microsoft/Windows MDM flags as we don't want it to be discoverable for users for now
	betaMDMFlags := []string{
		"mdm.windows_wstep_identity_cert",
		"mdm.windows_wstep_identity_key",
		"mdm.windows_wstep_identity_cert_bytes",
		"mdm.windows_wstep_identity_key_bytes",
	}
	for _, mdmFlag := range betaMDMFlags {
		if flag := man.command.PersistentFlags().Lookup(flagNameFromConfigKey(mdmFlag)); flag != nil {
			flag.Hidden = true
		}
	}
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
			SQLMode:         man.getConfigString(prefix + ".sql_mode"),
		}
	}

	cfg := FleetConfig{
		Mysql:            loadMysqlConfig("mysql"),
		MysqlReadReplica: loadMysqlConfig("mysql_read_replica"),
		Redis: RedisConfig{
			Address:                   man.getConfigString("redis.address"),
			Username:                  man.getConfigString("redis.username"),
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
			WriteTimeout:              man.getConfigDuration("redis.write_timeout"),
			ReadTimeout:               man.getConfigDuration("redis.read_timeout"),
		},
		Server: ServerConfig{
			Address:                     man.getConfigString("server.address"),
			Cert:                        man.getConfigString("server.cert"),
			Key:                         man.getConfigString("server.key"),
			TLS:                         man.getConfigBool("server.tls"),
			TLSProfile:                  man.getConfigTLSProfile(),
			URLPrefix:                   man.getConfigString("server.url_prefix"),
			Keepalive:                   man.getConfigBool("server.keepalive"),
			SandboxEnabled:              man.getConfigBool("server.sandbox_enabled"),
			WebsocketsAllowUnsafeOrigin: man.getConfigBool("server.websockets_allow_unsafe_origin"),
		},
		Auth: AuthConfig{
			BcryptCost:  man.getConfigInt("auth.bcrypt_cost"),
			SaltKeySize: man.getConfigInt("auth.salt_key_size"),
		},
		App: AppConfig{
			TokenKeySize:              man.getConfigInt("app.token_key_size"),
			InviteTokenValidityPeriod: man.getConfigDuration("app.invite_token_validity_period"),
			EnableScheduledQueryStats: man.getConfigBool("app.enable_scheduled_query_stats"),
		},
		Session: SessionConfig{
			KeySize:  man.getConfigInt("session.key_size"),
			Duration: man.getConfigDuration("session.duration"),
		},
		Osquery: OsqueryConfig{
			NodeKeySize:     man.getConfigInt("osquery.node_key_size"),
			HostIdentifier:  man.getConfigString("osquery.host_identifier"),
			EnrollCooldown:  man.getConfigDuration("osquery.enroll_cooldown"),
			StatusLogPlugin: man.getConfigString("osquery.status_log_plugin"),
			ResultLogPlugin: man.getConfigString("osquery.result_log_plugin"),
			// StatusLogFile is deprecated. FilesystemConfig.StatusLogFile is used instead.
			StatusLogFile: man.getConfigString("osquery.status_log_file"),
			// ResultLogFile is deprecated. FilesystemConfig.ResultLogFile is used instead.
			ResultLogFile:                    man.getConfigString("osquery.result_log_file"),
			LabelUpdateInterval:              man.getConfigDuration("osquery.label_update_interval"),
			PolicyUpdateInterval:             man.getConfigDuration("osquery.policy_update_interval"),
			DetailUpdateInterval:             man.getConfigDuration("osquery.detail_update_interval"),
			EnableLogRotation:                man.getConfigBool("osquery.enable_log_rotation"),
			MaxJitterPercent:                 man.getConfigInt("osquery.max_jitter_percent"),
			EnableAsyncHostProcessing:        man.getConfigString("osquery.enable_async_host_processing"),
			AsyncHostCollectInterval:         man.getConfigString("osquery.async_host_collect_interval"),
			AsyncHostCollectMaxJitterPercent: man.getConfigInt("osquery.async_host_collect_max_jitter_percent"),
			AsyncHostCollectLockTimeout:      man.getConfigString("osquery.async_host_collect_lock_timeout"),
			AsyncHostCollectLogStatsInterval: man.getConfigDuration("osquery.async_host_collect_log_stats_interval"),
			AsyncHostInsertBatch:             man.getConfigInt("osquery.async_host_insert_batch"),
			AsyncHostDeleteBatch:             man.getConfigInt("osquery.async_host_delete_batch"),
			AsyncHostUpdateBatch:             man.getConfigInt("osquery.async_host_update_batch"),
			AsyncHostRedisPopCount:           man.getConfigInt("osquery.async_host_redis_pop_count"),
			AsyncHostRedisScanKeysCount:      man.getConfigInt("osquery.async_host_redis_scan_keys_count"),
			MinSoftwareLastOpenedAtDiff:      man.getConfigDuration("osquery.min_software_last_opened_at_diff"),
		},
		Activity: ActivityConfig{
			EnableAuditLog: man.getConfigBool("activity.enable_audit_log"),
			AuditLogPlugin: man.getConfigString("activity.audit_log_plugin"),
		},
		Logging: LoggingConfig{
			Debug:                man.getConfigBool("logging.debug"),
			JSON:                 man.getConfigBool("logging.json"),
			DisableBanner:        man.getConfigBool("logging.disable_banner"),
			ErrorRetentionPeriod: man.getConfigDuration("logging.error_retention_period"),
			TracingEnabled:       man.getConfigBool("logging.tracing_enabled"),
			TracingType:          man.getConfigString("logging.tracing_type"),
		},
		Firehose: FirehoseConfig{
			Region:           man.getConfigString("firehose.region"),
			EndpointURL:      man.getConfigString("firehose.endpoint_url"),
			AccessKeyID:      man.getConfigString("firehose.access_key_id"),
			SecretAccessKey:  man.getConfigString("firehose.secret_access_key"),
			StsAssumeRoleArn: man.getConfigString("firehose.sts_assume_role_arn"),
			StatusStream:     man.getConfigString("firehose.status_stream"),
			ResultStream:     man.getConfigString("firehose.result_stream"),
			AuditStream:      man.getConfigString("firehose.audit_stream"),
		},
		Kinesis: KinesisConfig{
			Region:           man.getConfigString("kinesis.region"),
			EndpointURL:      man.getConfigString("kinesis.endpoint_url"),
			AccessKeyID:      man.getConfigString("kinesis.access_key_id"),
			SecretAccessKey:  man.getConfigString("kinesis.secret_access_key"),
			StatusStream:     man.getConfigString("kinesis.status_stream"),
			ResultStream:     man.getConfigString("kinesis.result_stream"),
			AuditStream:      man.getConfigString("kinesis.audit_stream"),
			StsAssumeRoleArn: man.getConfigString("kinesis.sts_assume_role_arn"),
		},
		Lambda: LambdaConfig{
			Region:           man.getConfigString("lambda.region"),
			AccessKeyID:      man.getConfigString("lambda.access_key_id"),
			SecretAccessKey:  man.getConfigString("lambda.secret_access_key"),
			StatusFunction:   man.getConfigString("lambda.status_function"),
			ResultFunction:   man.getConfigString("lambda.result_function"),
			AuditFunction:    man.getConfigString("lambda.audit_function"),
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
		Email: EmailConfig{
			EmailBackend: man.getConfigString("email.backend"),
		},
		SES: SESConfig{
			Region:           man.getConfigString("ses.region"),
			EndpointURL:      man.getConfigString("ses.endpoint_url"),
			AccessKeyID:      man.getConfigString("ses.access_key_id"),
			SecretAccessKey:  man.getConfigString("ses.secret_access_key"),
			StsAssumeRoleArn: man.getConfigString("ses.sts_assume_role_arn"),
			SourceArn:        man.getConfigString("ses.source_arn"),
		},
		PubSub: PubSubConfig{
			Project:       man.getConfigString("pubsub.project"),
			StatusTopic:   man.getConfigString("pubsub.status_topic"),
			ResultTopic:   man.getConfigString("pubsub.result_topic"),
			AuditTopic:    man.getConfigString("pubsub.audit_topic"),
			AddAttributes: man.getConfigBool("pubsub.add_attributes"),
		},
		Filesystem: FilesystemConfig{
			StatusLogFile:        man.getConfigString("filesystem.status_log_file"),
			ResultLogFile:        man.getConfigString("filesystem.result_log_file"),
			AuditLogFile:         man.getConfigString("filesystem.audit_log_file"),
			EnableLogRotation:    man.getConfigBool("filesystem.enable_log_rotation"),
			EnableLogCompression: man.getConfigBool("filesystem.enable_log_compression"),
			MaxSize:              man.getConfigInt("filesystem.max_size"),
			MaxAge:               man.getConfigInt("filesystem.max_age"),
			MaxBackups:           man.getConfigInt("filesystem.max_backups"),
		},
		KafkaREST: KafkaRESTConfig{
			StatusTopic:      man.getConfigString("kafkarest.status_topic"),
			ResultTopic:      man.getConfigString("kafkarest.result_topic"),
			AuditTopic:       man.getConfigString("kafkarest.audit_topic"),
			ProxyHost:        man.getConfigString("kafkarest.proxyhost"),
			ContentTypeValue: man.getConfigString("kafkarest.content_type_value"),
			Timeout:          man.getConfigInt("kafkarest.timeout"),
		},
		License: LicenseConfig{
			Key:              man.getConfigString("license.key"),
			EnforceHostLimit: man.getConfigBool("license.enforce_host_limit"),
		},
		Vulnerabilities: VulnerabilitiesConfig{
			DatabasesPath:               man.getConfigString("vulnerabilities.databases_path"),
			Periodicity:                 man.getConfigDuration("vulnerabilities.periodicity"),
			CPEDatabaseURL:              man.getConfigString("vulnerabilities.cpe_database_url"),
			CPETranslationsURL:          man.getConfigString("vulnerabilities.cpe_translations_url"),
			CVEFeedPrefixURL:            man.getConfigString("vulnerabilities.cve_feed_prefix_url"),
			CurrentInstanceChecks:       man.getConfigString("vulnerabilities.current_instance_checks"),
			DisableSchedule:             man.getConfigBool("vulnerabilities.disable_schedule"),
			DisableDataSync:             man.getConfigBool("vulnerabilities.disable_data_sync"),
			RecentVulnerabilityMaxAge:   man.getConfigDuration("vulnerabilities.recent_vulnerability_max_age"),
			DisableWinOSVulnerabilities: man.getConfigBool("vulnerabilities.disable_win_os_vulnerabilities"),
		},
		Upgrades: UpgradesConfig{
			AllowMissingMigrations: man.getConfigBool("upgrades.allow_missing_migrations"),
		},
		Sentry: SentryConfig{
			Dsn: man.getConfigString("sentry.dsn"),
		},
		GeoIP: GeoIPConfig{
			DatabasePath: man.getConfigString("geoip.database_path"),
		},
		Prometheus: PrometheusConfig{
			BasicAuth: HTTPBasicAuthConfig{
				Username: man.getConfigString("prometheus.basic_auth.username"),
				Password: man.getConfigString("prometheus.basic_auth.password"),
				Disable:  man.getConfigBool("prometheus.basic_auth.disable"),
			},
		},
		Packaging: PackagingConfig{
			GlobalEnrollSecret: man.getConfigString("packaging.global_enroll_secret"),
			S3: S3Config{
				Bucket:           man.getConfigString("packaging.s3.bucket"),
				Prefix:           man.getConfigString("packaging.s3.prefix"),
				Region:           man.getConfigString("packaging.s3.region"),
				EndpointURL:      man.getConfigString("packaging.s3.endpoint_url"),
				AccessKeyID:      man.getConfigString("packaging.s3.access_key_id"),
				SecretAccessKey:  man.getConfigString("packaging.s3.secret_access_key"),
				StsAssumeRoleArn: man.getConfigString("packaging.s3.sts_assume_role_arn"),
				DisableSSL:       man.getConfigBool("packaging.s3.disable_ssl"),
				ForceS3PathStyle: man.getConfigBool("packaging.s3.force_s3_path_style"),
			},
		},
		MDM: MDMConfig{
			AppleAPNsCert:                   man.getConfigString("mdm.apple_apns_cert"),
			AppleAPNsCertBytes:              man.getConfigString("mdm.apple_apns_cert_bytes"),
			AppleAPNsKey:                    man.getConfigString("mdm.apple_apns_key"),
			AppleAPNsKeyBytes:               man.getConfigString("mdm.apple_apns_key_bytes"),
			AppleSCEPCert:                   man.getConfigString("mdm.apple_scep_cert"),
			AppleSCEPCertBytes:              man.getConfigString("mdm.apple_scep_cert_bytes"),
			AppleSCEPKey:                    man.getConfigString("mdm.apple_scep_key"),
			AppleSCEPKeyBytes:               man.getConfigString("mdm.apple_scep_key_bytes"),
			AppleBMServerToken:              man.getConfigString("mdm.apple_bm_server_token"),
			AppleBMServerTokenBytes:         man.getConfigString("mdm.apple_bm_server_token_bytes"),
			AppleBMCert:                     man.getConfigString("mdm.apple_bm_cert"),
			AppleBMCertBytes:                man.getConfigString("mdm.apple_bm_cert_bytes"),
			AppleBMKey:                      man.getConfigString("mdm.apple_bm_key"),
			AppleBMKeyBytes:                 man.getConfigString("mdm.apple_bm_key_bytes"),
			AppleEnable:                     man.getConfigBool("mdm.apple_enable"),
			AppleSCEPSignerValidityDays:     man.getConfigInt("mdm.apple_scep_signer_validity_days"),
			AppleSCEPSignerAllowRenewalDays: man.getConfigInt("mdm.apple_scep_signer_allow_renewal_days"),
			AppleSCEPChallenge:              man.getConfigString("mdm.apple_scep_challenge"),
			AppleDEPSyncPeriodicity:         man.getConfigDuration("mdm.apple_dep_sync_periodicity"),
			WindowsWSTEPIdentityCert:        man.getConfigString("mdm.windows_wstep_identity_cert"),
			WindowsWSTEPIdentityKey:         man.getConfigString("mdm.windows_wstep_identity_key"),
			WindowsWSTEPIdentityCertBytes:   man.getConfigString("mdm.windows_wstep_identity_cert_bytes"),
			WindowsWSTEPIdentityKeyBytes:    man.getConfigString("mdm.windows_wstep_identity_key_bytes"),
		},
	}

	// ensure immediately that the async config is valid for all known tasks
	for task := range knownAsyncTasks {
		cfg.Osquery.AsyncConfigForTask(task)
	}

	return cfg
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
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key))) //nolint:errcheck
	man.viper.BindEnv(key, envNameFromConfigKey(key))                                          //nolint:errcheck

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
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key))) //nolint:errcheck
	man.viper.BindEnv(key, envNameFromConfigKey(key))                                          //nolint:errcheck

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
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key))) //nolint:errcheck
	man.viper.BindEnv(key, envNameFromConfigKey(key))                                          //nolint:errcheck

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
	man.viper.BindPFlag(key, man.command.PersistentFlags().Lookup(flagNameFromConfigKey(key))) //nolint:errcheck
	man.viper.BindEnv(key, envNameFromConfigKey(key))                                          //nolint:errcheck

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

// panics if the config is invalid, this is handled by Viper (this is how all
// getConfigT helpers indicate errors). The default value is only applied if
// there is no task-specific config (i.e., no "task=true" config format for that
// task). If the configuration key was not set at all, it automatically
// inherited the general default configured for that key (via
// man.addConfigBool).
func configForKeyOrBool(key, task, val string, def bool) bool {
	parseVal := func(v string) bool {
		if v == "" {
			return false
		}

		b, err := strconv.ParseBool(v)
		if err != nil {
			panic("Unable to cast to bool for key " + key + ": " + err.Error())
		}
		return b
	}

	if !strings.Contains(val, "=") {
		// simple case, val is a bool
		return parseVal(val)
	}

	q, err := url.ParseQuery(val)
	if err != nil {
		panic("Invalid query format for key " + key + ": " + err.Error())
	}
	if v := q.Get(task); v != "" {
		return parseVal(v)
	}
	return def
}

// panics if the config is invalid, this is handled by Viper (this is how all
// getConfigT helpers indicate errors). The default value is only applied if
// there is no task-specific config (i.e. no "task=10s" config format for that
// task). If the configuration key was not set at all, it automatically
// inherited the general default configured for that key (via
// man.addConfigDuration).
func configForKeyOrDuration(key, task, val string, def time.Duration) time.Duration {
	parseVal := func(v string) time.Duration {
		if v == "" {
			return 0
		}

		d, err := time.ParseDuration(v)
		if err != nil {
			panic("Unable to cast to time.Duration for key " + key + ": " + err.Error())
		}
		return d
	}

	if !strings.Contains(val, "=") {
		// simple case, val is a duration
		return parseVal(val)
	}

	q, err := url.ParseQuery(val)
	if err != nil {
		panic("Invalid query format for key " + key + ": " + err.Error())
	}
	if v := q.Get(task); v != "" {
		return parseVal(v)
	}
	return def
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

	fmt.Println("Using config file:", man.viper.ConfigFileUsed())
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
			Duration: 24 * 5 * time.Hour,
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
		Activity: ActivityConfig{
			EnableAuditLog: true,
			AuditLogPlugin: "filesystem",
		},
		Logging: LoggingConfig{
			Debug:         true,
			DisableBanner: true,
		},
		Filesystem: FilesystemConfig{
			StatusLogFile: testLogFile,
			ResultLogFile: testLogFile,
			AuditLogFile:  testLogFile,
			MaxSize:       500,
		},
	}
}

// SetTestMDMConfig modifies the provided cfg so that MDM is enabled and
// configured properly. The provided certificate and private key are used for
// all required pairs and the Apple BM token is used as-is, instead of
// decrypting the encrypted value that is usually provided via the fleet
// server's flags.
func SetTestMDMConfig(t testing.TB, cfg *FleetConfig, cert, key []byte, appleBMToken *nanodep_client.OAuth1Tokens, wstepCertAndKeyDir string) {
	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	tlsCert.Leaf = parsed

	cfg.MDM.AppleAPNsCertBytes = string(cert)
	cfg.MDM.AppleAPNsKeyBytes = string(key)
	cfg.MDM.AppleSCEPCertBytes = string(cert)
	cfg.MDM.AppleSCEPKeyBytes = string(key)
	cfg.MDM.AppleBMCertBytes = string(cert)
	cfg.MDM.AppleBMKeyBytes = string(key)
	cfg.MDM.AppleBMServerTokenBytes = "whatever-will-not-be-accessed"

	cfg.MDM.appleAPNs = &tlsCert
	cfg.MDM.appleAPNsPEMCert = cert
	cfg.MDM.appleAPNsPEMKey = key
	cfg.MDM.appleSCEP = &tlsCert
	cfg.MDM.appleSCEPPEMCert = cert
	cfg.MDM.appleSCEPPEMKey = key
	cfg.MDM.appleBMToken = appleBMToken
	cfg.MDM.AppleSCEPSignerValidityDays = 365
	cfg.MDM.AppleSCEPChallenge = "testchallenge"

	if wstepCertAndKeyDir == "" {
		wstepCertAndKeyDir = "testdata"
	}
	certPath := filepath.Join(wstepCertAndKeyDir, "server.pem")
	keyPath := filepath.Join(wstepCertAndKeyDir, "server.key")

	cfg.MDM.WindowsWSTEPIdentityCert = certPath
	cfg.MDM.WindowsWSTEPIdentityKey = keyPath
	if _, _, _, err := cfg.MDM.MicrosoftWSTEP(); err != nil {
		t.Fatal(err)
	}
}
