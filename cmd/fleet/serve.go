package main

import (
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"database/sql/driver"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/e-dard/netbug"
	"github.com/fleetdm/fleet/v4/ee/server/licensing"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server"
	configpkg "github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	licensectx "github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/cron"
	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysqlredis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/health"
	"github.com/fleetdm/fleet/v4/server/launcher"
	"github.com/fleetdm/fleet/v4/server/live_query"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/mail"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/cryptoutil"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/buford"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/fleetdm/fleet/v4/server/service/redis_lock"
	"github.com/fleetdm/fleet/v4/server/service/redis_policy_set"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/fleetdm/fleet/v4/server/version"
	"github.com/getsentry/sentry-go"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/ngrok/sqlmw"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.elastic.co/apm/module/apmhttp/v2"
	_ "go.elastic.co/apm/module/apmsql/v2"
	_ "go.elastic.co/apm/module/apmsql/v2/mysql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

var allowedURLPrefixRegexp = regexp.MustCompile("^(?:/[a-zA-Z0-9_.~-]+)+$")

const (
	liveQueryMemCacheDuration = 1 * time.Second
)

type initializer interface {
	// Initialize is used to populate a datastore with
	// preloaded data
	Initialize() error
}

func createServeCmd(configManager configpkg.Manager) *cobra.Command {
	// Whether to enable the debug endpoints
	debug := false
	// Whether to enable developer options
	dev := false
	// Whether to enable development Fleet Premium license
	devLicense := false
	// Whether to enable development Fleet Premium license with an expired license
	devExpiredLicense := false

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Launch the Fleet server",
		Long: `
Launch the Fleet server

Use fleet serve to run the main HTTPS server. The Fleet server bundles
together all static assets and dependent libraries into a statically linked go
binary (which you're executing right now). Use the options below to customize
the way that the Fleet server works.
`,
		Run: func(cmd *cobra.Command, args []string) {
			config := configManager.LoadConfig()

			if dev {
				applyDevFlags(&config)
			}

			license, err := initLicense(config, devLicense, devExpiredLicense)
			if err != nil {
				initFatal(
					err,
					"failed to load license - for help use https://fleetdm.com/contact",
				)
			}

			if license != nil && license.IsPremium() && license.IsExpired() {
				fleet.WriteExpiredLicenseBanner(os.Stderr)
			}

			logger := initLogger(config)

			if dev {
				createTestBucketForInstallers(&config, logger)
			}

			// Init tracing
			if config.Logging.TracingEnabled {
				ctx := context.Background()
				client := otlptracegrpc.NewClient()
				otlpTraceExporter, err := otlptrace.New(ctx, client)
				if err != nil {
					initFatal(err, "Failed to initialize tracing")
				}
				batchSpanProcessor := sdktrace.NewBatchSpanProcessor(otlpTraceExporter)
				tracerProvider := sdktrace.NewTracerProvider(
					sdktrace.WithSpanProcessor(batchSpanProcessor),
				)
				otel.SetTracerProvider(tracerProvider)
			}

			allowedHostIdentifiers := map[string]bool{
				"provided": true,
				"instance": true,
				"uuid":     true,
				"hostname": true,
			}
			if !allowedHostIdentifiers[config.Osquery.HostIdentifier] {
				initFatal(fmt.Errorf("%s is not a valid value for osquery_host_identifier", config.Osquery.HostIdentifier), "set host identifier")
			}

			if len(config.Server.URLPrefix) > 0 {
				// Massage provided prefix to match expected format
				config.Server.URLPrefix = strings.TrimSuffix(config.Server.URLPrefix, "/")
				if len(config.Server.URLPrefix) > 0 && !strings.HasPrefix(config.Server.URLPrefix, "/") {
					config.Server.URLPrefix = "/" + config.Server.URLPrefix
				}

				if !allowedURLPrefixRegexp.MatchString(config.Server.URLPrefix) {
					initFatal(
						fmt.Errorf("prefix must match regexp \"%s\"", allowedURLPrefixRegexp.String()),
						"setting server URL prefix",
					)
				}
			}

			if len(config.Server.PrivateKey) > 0 {
				if len(config.Server.PrivateKey) < 32 {
					initFatal(errors.New("private key must be at least 32 bytes long"), "validate private key")
				}

				// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
				// infra setups generate keys that are longer than 32 bytes.
				config.Server.PrivateKey = config.Server.PrivateKey[:32]
			}

			var ds fleet.Datastore
			var carveStore fleet.CarveStore
			var installerStore fleet.InstallerStore

			opts := []mysql.DBOption{mysql.Logger(logger), mysql.WithFleetConfig(&config)}
			if config.MysqlReadReplica.Address != "" {
				opts = append(opts, mysql.Replica(&config.MysqlReadReplica))
			}
			// NOTE this will disable OTEL/APM interceptor
			if dev && os.Getenv("FLEET_DEV_ENABLE_SQL_INTERCEPTOR") != "" {
				opts = append(opts, mysql.WithInterceptor(&devSQLInterceptor{
					logger: kitlog.With(logger, "component", "sql-interceptor"),
				}))
			}

			if config.Logging.TracingEnabled {
				opts = append(opts, mysql.TracingEnabled(&config.Logging))
			}

			mds, err := mysql.New(config.Mysql, clock.C, opts...)
			if err != nil {
				initFatal(err, "initializing datastore")
			}
			ds = mds

			if config.S3.CarvesBucket != "" || config.S3.Bucket != "" {
				carveStore, err = s3.NewCarveStore(config.S3, ds)
				if err != nil {
					initFatal(err, "initializing S3 carvestore")
				}
			} else {
				carveStore = ds
			}

			if config.Packaging.S3.Bucket != "" {
				var err error
				installerStore, err = s3.NewInstallerStore(config.Packaging.S3)
				if err != nil {
					initFatal(err, "initializing S3 installer store")
				}
			}

			migrationStatus, err := ds.MigrationStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

			switch migrationStatus.StatusCode {
			case fleet.AllMigrationsCompleted:
				// OK
			case fleet.UnknownMigrations:
				fmt.Printf("################################################################################\n"+
					"# WARNING:\n"+
					"#   Your Fleet database has unrecognized migrations. This could happen when\n"+
					"#   running an older version of Fleet on a newer migrated database.\n"+
					"#\n"+
					"#   Unknown migrations: tables=%v, data=%v.\n"+
					"################################################################################\n",
					migrationStatus.UnknownTable, migrationStatus.UnknownData)
				if dev {
					os.Exit(1)
				}
			case fleet.SomeMigrationsCompleted:
				fmt.Printf("################################################################################\n"+
					"# WARNING:\n"+
					"#   Your Fleet database is missing required migrations. This is likely to cause\n"+
					"#   errors in Fleet.\n"+
					"#\n"+
					"#   Missing migrations: tables=%v, data=%v.\n"+
					"#\n"+
					"#   Run `%s prepare db` to perform migrations.\n"+
					"#\n"+
					"#   To run the server without performing migrations:\n"+
					"#     - Set environment variable FLEET_UPGRADES_ALLOW_MISSING_MIGRATIONS=1, or,\n"+
					"#     - Set config updates.allow_missing_migrations to true, or,\n"+
					"#     - Use command line argument --upgrades_allow_missing_migrations=true\n"+
					"################################################################################\n",
					migrationStatus.MissingTable, migrationStatus.MissingData, os.Args[0])
				if !config.Upgrades.AllowMissingMigrations {
					os.Exit(1)
				}
			case fleet.NoMigrationsCompleted:
				fmt.Printf("################################################################################\n"+
					"# ERROR:\n"+
					"#   Your Fleet database is not initialized. Fleet cannot start up.\n"+
					"#\n"+
					"#   Run `%s prepare db` to initialize the database.\n"+
					"################################################################################\n",
					os.Args[0])
				os.Exit(1)
			}

			featureMigrationStatus, err := ds.FeatureMigrationStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving feature migration status")
			}

			// TODO(victor): Refactor this check to be more DRY
			switch featureMigrationStatus.StatusCode {
			case fleet.AllMigrationsCompleted:
				// OK
			case fleet.UnknownMigrations:
				fmt.Printf("################################################################################\n"+
					"# WARNING:\n"+
					"#   Your Fleet database has unrecognized feature migrations. This could happen when\n"+
					"#   running an older version of Fleet on a newer migrated database.\n"+
					"#\n"+
					"#   Unknown migrations: tables=%v, data=%v.\n"+
					"################################################################################\n",
					featureMigrationStatus.UnknownTable, featureMigrationStatus.UnknownData)
				if dev {
					os.Exit(1)
				}
			case fleet.SomeMigrationsCompleted:
				fmt.Printf("################################################################################\n"+
					"# WARNING:\n"+
					"#   Your Fleet database is missing required feature migrations. This is likely to cause\n"+
					"#   errors in Fleet.\n"+
					"#\n"+
					"#   Missing migrations: tables=%v, data=%v.\n"+
					"#\n"+
					"#   Run `%s prepare db` to perform migrations.\n"+
					"#\n"+
					"#   To run the server without performing migrations:\n"+
					"#     - Set environment variable FLEET_UPGRADES_ALLOW_MISSING_MIGRATIONS=1, or,\n"+
					"#     - Set config updates.allow_missing_migrations to true, or,\n"+
					"#     - Use command line argument --upgrades_allow_missing_migrations=true\n"+
					"################################################################################\n",
					featureMigrationStatus.MissingTable, featureMigrationStatus.MissingData, os.Args[0])
				if !config.Upgrades.AllowMissingMigrations {
					os.Exit(1)
				}
			case fleet.NoMigrationsCompleted:
				fmt.Printf("################################################################################\n"+
					"# ERROR:\n"+
					"#   Your Fleet database is not initialized. Fleet cannot start up.\n"+
					"#\n"+
					"#   Run `%s prepare db` to initialize the database.\n"+
					"################################################################################\n",
					os.Args[0])
				os.Exit(1)
			}

			if initializingDS, ok := ds.(initializer); ok {
				if err := initializingDS.Initialize(); err != nil {
					initFatal(err, "loading built in data")
				}
			}

			if config.Packaging.GlobalEnrollSecret != "" {
				secrets, err := ds.GetEnrollSecrets(cmd.Context(), nil)
				if err != nil {
					initFatal(err, "loading enroll secrets")
				}

				var globalEnrollSecret string
				for _, secret := range secrets {
					if secret.TeamID == nil {
						globalEnrollSecret = secret.Secret
						break
					}
				}

				if globalEnrollSecret != "" {
					if globalEnrollSecret != config.Packaging.GlobalEnrollSecret {
						fmt.Printf("################################################################################\n" +
							"# WARNING:\n" +
							"#  You have provided a global enroll secret config, but there's\n" +
							"#  already one set up for your application.\n" +
							"#\n" +
							"#  This is generally an error and the provided value will be\n" +
							"#  ignored, if you really need to configure an enroll secret please\n" +
							"#  remove the global enroll secret from the database manually.\n" +
							"################################################################################\n")
						os.Exit(1)
					}
				} else {
					if err := ds.ApplyEnrollSecrets(cmd.Context(), nil, []*fleet.EnrollSecret{{Secret: config.Packaging.GlobalEnrollSecret}}); err != nil {
						level.Debug(logger).Log("err", err, "msg", "failed to apply enroll secrets")
					}
				}
			}

			// Strip the Redis URI scheme if it's present. Scheme docs are at: https://www.iana.org/assignments/uri-schemes/uri-schemes.xhtml
			// This allows us to use Render's Redis service in render.yaml, including the free tier.
			// In the future, we could support the full Redis URI if needed (including username, password, database, etc.)
			redisAddress := strings.TrimPrefix(config.Redis.Address, "redis://")
			redisPool, err := redis.NewPool(redis.PoolConfig{
				Server:                    redisAddress,
				Username:                  config.Redis.Username,
				Password:                  config.Redis.Password,
				Database:                  config.Redis.Database,
				UseTLS:                    config.Redis.UseTLS,
				ConnTimeout:               config.Redis.ConnectTimeout,
				KeepAlive:                 config.Redis.KeepAlive,
				ConnectRetryAttempts:      config.Redis.ConnectRetryAttempts,
				ClusterFollowRedirections: config.Redis.ClusterFollowRedirections,
				ClusterReadFromReplica:    config.Redis.ClusterReadFromReplica,
				TLSCert:                   config.Redis.TLSCert,
				TLSKey:                    config.Redis.TLSKey,
				TLSCA:                     config.Redis.TLSCA,
				TLSServerName:             config.Redis.TLSServerName,
				TLSHandshakeTimeout:       config.Redis.TLSHandshakeTimeout,
				MaxIdleConns:              config.Redis.MaxIdleConns,
				MaxOpenConns:              config.Redis.MaxOpenConns,
				ConnMaxLifetime:           config.Redis.ConnMaxLifetime,
				IdleTimeout:               config.Redis.IdleTimeout,
				ConnWaitTimeout:           config.Redis.ConnWaitTimeout,
				WriteTimeout:              config.Redis.WriteTimeout,
				ReadTimeout:               config.Redis.ReadTimeout,
			})
			if err != nil {
				initFatal(err, "initialize Redis")
			}
			level.Info(logger).Log("component", "redis", "mode", redisPool.Mode())

			ds = cached_mysql.New(ds)
			var dsOpts []mysqlredis.Option
			if license.DeviceCount > 0 && config.License.EnforceHostLimit {
				dsOpts = append(dsOpts, mysqlredis.WithEnforcedHostLimit(license.DeviceCount))
			}
			redisWrapperDS := mysqlredis.New(ds, redisPool, dsOpts...)
			ds = redisWrapperDS

			resultStore := pubsub.NewRedisQueryResults(redisPool, config.Redis.DuplicateResults,
				log.With(logger, "component", "query-results"),
			)
			liveQueryStore := live_query.NewRedisLiveQuery(redisPool, logger, liveQueryMemCacheDuration)
			ssoSessionStore := sso.NewSessionStore(redisPool)

			// Set common configuration for all logging.
			loggingConfig := logging.Config{
				Filesystem: logging.FilesystemConfig{
					EnableLogRotation:    config.Filesystem.EnableLogRotation,
					EnableLogCompression: config.Filesystem.EnableLogCompression,
					MaxSize:              config.Filesystem.MaxSize,
					MaxAge:               config.Filesystem.MaxAge,
					MaxBackups:           config.Filesystem.MaxBackups,
				},
				Firehose: logging.FirehoseConfig{
					Region:           config.Firehose.Region,
					EndpointURL:      config.Firehose.EndpointURL,
					AccessKeyID:      config.Firehose.AccessKeyID,
					SecretAccessKey:  config.Firehose.SecretAccessKey,
					StsAssumeRoleArn: config.Firehose.StsAssumeRoleArn,
					StsExternalID:    config.Firehose.StsExternalID,
				},
				Kinesis: logging.KinesisConfig{
					Region:           config.Kinesis.Region,
					EndpointURL:      config.Kinesis.EndpointURL,
					AccessKeyID:      config.Kinesis.AccessKeyID,
					SecretAccessKey:  config.Kinesis.SecretAccessKey,
					StsAssumeRoleArn: config.Kinesis.StsAssumeRoleArn,
					StsExternalID:    config.Kinesis.StsExternalID,
				},
				Lambda: logging.LambdaConfig{
					Region:           config.Lambda.Region,
					AccessKeyID:      config.Lambda.AccessKeyID,
					SecretAccessKey:  config.Lambda.SecretAccessKey,
					StsAssumeRoleArn: config.Lambda.StsAssumeRoleArn,
					StsExternalID:    config.Lambda.StsExternalID,
				},
				PubSub: logging.PubSubConfig{
					Project: config.PubSub.Project,
				},
				KafkaREST: logging.KafkaRESTConfig{
					ProxyHost:        config.KafkaREST.ProxyHost,
					ContentTypeValue: config.KafkaREST.ContentTypeValue,
					Timeout:          config.KafkaREST.Timeout,
				},
			}

			// Set specific configuration to osqueryd status logs.
			loggingConfig.Plugin = config.Osquery.StatusLogPlugin
			loggingConfig.Filesystem.LogFile = config.Filesystem.StatusLogFile
			loggingConfig.Firehose.StreamName = config.Firehose.StatusStream
			loggingConfig.Kinesis.StreamName = config.Kinesis.StatusStream
			loggingConfig.Lambda.Function = config.Lambda.StatusFunction
			loggingConfig.PubSub.Topic = config.PubSub.StatusTopic
			loggingConfig.PubSub.AddAttributes = false // only used by result logs
			loggingConfig.KafkaREST.Topic = config.KafkaREST.StatusTopic

			osquerydStatusLogger, err := logging.NewJSONLogger("status", loggingConfig, logger)
			if err != nil {
				initFatal(err, "initializing osqueryd status logging")
			}

			// Set specific configuration to osqueryd result logs.
			loggingConfig.Plugin = config.Osquery.ResultLogPlugin
			loggingConfig.Filesystem.LogFile = config.Filesystem.ResultLogFile
			loggingConfig.Firehose.StreamName = config.Firehose.ResultStream
			loggingConfig.Kinesis.StreamName = config.Kinesis.ResultStream
			loggingConfig.Lambda.Function = config.Lambda.ResultFunction
			loggingConfig.PubSub.Topic = config.PubSub.ResultTopic
			loggingConfig.PubSub.AddAttributes = config.PubSub.AddAttributes
			loggingConfig.KafkaREST.Topic = config.KafkaREST.ResultTopic

			osquerydResultLogger, err := logging.NewJSONLogger("result", loggingConfig, logger)
			if err != nil {
				initFatal(err, "initializing osqueryd result logging")
			}

			var auditLogger fleet.JSONLogger
			if license.IsPremium() && config.Activity.EnableAuditLog {
				// Set specific configuration to audit logs.
				loggingConfig.Plugin = config.Activity.AuditLogPlugin
				loggingConfig.Filesystem.LogFile = config.Filesystem.AuditLogFile
				loggingConfig.Firehose.StreamName = config.Firehose.AuditStream
				loggingConfig.Kinesis.StreamName = config.Kinesis.AuditStream
				loggingConfig.Lambda.Function = config.Lambda.AuditFunction
				loggingConfig.PubSub.Topic = config.PubSub.AuditTopic
				loggingConfig.PubSub.AddAttributes = false // only used by result logs
				loggingConfig.KafkaREST.Topic = config.KafkaREST.AuditTopic

				auditLogger, err = logging.NewJSONLogger("audit", loggingConfig, logger)
				if err != nil {
					initFatal(err, "initializing audit logging")
				}
			}

			failingPolicySet := redis_policy_set.NewFailing(redisPool)

			task := async.NewTask(ds, redisPool, clock.C, config.Osquery)

			if config.Sentry.Dsn != "" {
				v := version.Version()
				err = sentry.Init(sentry.ClientOptions{
					Dsn:     config.Sentry.Dsn,
					Release: fmt.Sprintf("%s_%s_%s", v.Version, v.Branch, v.Revision),
				})
				if err != nil {
					initFatal(err, "initializing sentry")
				}
				level.Info(logger).Log("msg", "sentry initialized", "dsn", config.Sentry.Dsn)

				defer sentry.Recover()
				defer sentry.Flush(2 * time.Second)
			}

			var geoIP fleet.GeoIP
			geoIP = &fleet.NoOpGeoIP{}
			if config.GeoIP.DatabasePath != "" {
				maxmind, err := fleet.NewMaxMindGeoIP(logger, config.GeoIP.DatabasePath)
				if err != nil {
					level.Error(logger).Log("msg", "failed to initialize maxmind geoip, check database path", "database_path", config.GeoIP.DatabasePath, "error", err)
				} else {
					geoIP = maxmind
				}
			}

			mdmStorage, err := mds.NewMDMAppleMDMStorage()
			if err != nil {
				initFatal(err, "initialize mdm apple MySQL storage")
			}

			depStorage, err := mds.NewMDMAppleDEPStorage()
			if err != nil {
				initFatal(err, "initialize Apple BM DEP storage")
			}

			scepStorage, err := mds.NewSCEPDepot()
			if err != nil {
				initFatal(err, "initialize mdm apple scep storage")
			}

			var mdmPushService push.Pusher
			nanoMDMLogger := service.NewNanoMDMLogger(kitlog.With(logger, "component", "apple-mdm-push"))
			pushProviderFactory := buford.NewPushProviderFactory(buford.WithNewClient(func(cert *tls.Certificate) (*http.Client, error) {
				return fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
					Certificates: []tls.Certificate{*cert},
				})), nil
			}))
			if os.Getenv("FLEET_DEV_MDM_APPLE_DISABLE_PUSH") == "1" {
				mdmPushService = nopPusher{}
			} else {
				mdmPushService = nanomdm_pushsvc.New(mdmStorage, mdmStorage, pushProviderFactory, nanoMDMLogger)
			}

			checkMDMAssets := func(names []fleet.MDMAssetName) (bool, error) {
				_, err = ds.GetAllMDMConfigAssetsByName(context.Background(), names, nil)
				if err != nil {
					if fleet.IsNotFound(err) || errors.Is(err, mysql.ErrPartialResult) {
						return false, nil
					}
					return false, err
				}
				return true, nil
			}

			// reconcile Apple Business Manager configuration environment variables with the database
			if config.MDM.IsAppleAPNsSet() || config.MDM.IsAppleSCEPSet() {
				if len(config.Server.PrivateKey) == 0 {
					initFatal(errors.New("inserting MDM APNs and SCEP assets"), "missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
				}

				// first we'll check if the APNs and SCEP assets are already in the database and
				// only insert config values if they're not already present in the database
				toInsert := make(map[fleet.MDMAssetName]struct{}, 4)

				// check DB for APNs assets
				found, err := checkMDMAssets([]fleet.MDMAssetName{fleet.MDMAssetAPNSCert, fleet.MDMAssetAPNSKey})
				switch {
				case err != nil:
					initFatal(err, "reading APNs assets from database")
				case !found:
					toInsert[fleet.MDMAssetAPNSCert] = struct{}{}
					toInsert[fleet.MDMAssetAPNSKey] = struct{}{}
				default:
					level.Warn(logger).Log("msg", "Your server already has stored APNs certificates. Fleet will ignore any certificates provided via environment variables when this happens.")
				}

				// check DB for SCEP assets
				found, err = checkMDMAssets([]fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey})
				switch {
				case err != nil:
					initFatal(err, "reading SCEP assets from database")
				case !found:
					toInsert[fleet.MDMAssetCACert] = struct{}{}
					toInsert[fleet.MDMAssetCAKey] = struct{}{}
				default:
					level.Warn(logger).Log("msg", "Your server already has stored SCEP certificates. Fleet will ignore any certificates provided via environment variables when this happens.")
				}

				if len(toInsert) > 0 {
					if !config.MDM.IsAppleAPNsSet() {
						initFatal(errors.New("Apple APNs MDM configuration must be provided when Apple SCEP is provided"), "validate Apple MDM")
					} else if !config.MDM.IsAppleSCEPSet() {
						initFatal(errors.New("Apple SCEP MDM configuration must be provided when Apple APNs is provided"), "validate Apple MDM")
					}

					// parse the APNs and SCEP assets from the config
					_, apnsCertPEM, apnsKeyPEM, err := config.MDM.AppleAPNs()
					if err != nil {
						initFatal(err, "parse Apple APNs certificate and key from config")
					}
					_, appleSCEPCertPEM, appleSCEPKeyPEM, err := config.MDM.AppleSCEP()
					if err != nil {
						initFatal(err, "load Apple SCEP certificate and key from config")
					}

					var args []fleet.MDMConfigAsset
					for name := range toInsert {
						switch name {
						case fleet.MDMAssetAPNSCert:
							args = append(args, fleet.MDMConfigAsset{Name: name, Value: apnsCertPEM})
						case fleet.MDMAssetAPNSKey:
							args = append(args, fleet.MDMConfigAsset{Name: name, Value: apnsKeyPEM})
						case fleet.MDMAssetCACert:
							args = append(args, fleet.MDMConfigAsset{Name: name, Value: appleSCEPCertPEM})
						case fleet.MDMAssetCAKey:
							args = append(args, fleet.MDMConfigAsset{Name: name, Value: appleSCEPKeyPEM})
						}
					}

					if err := ds.InsertMDMConfigAssets(context.Background(), args, nil); err != nil {
						if mysql.IsDuplicate(err) {
							// we already checked for existing assets so we should never have a duplicate key error here; we'll add a debug log just in case
							level.Debug(logger).Log("msg", "unexpected duplicate key error inserting MDM APNs and SCEP assets")
						} else {
							initFatal(err, "inserting MDM APNs and SCEP assets")
						}
					}
				}
			}

			// reconcile Apple Business Manager configuration environment variables with the database
			if config.MDM.IsAppleBMSet() {
				if len(config.Server.PrivateKey) == 0 {
					initFatal(errors.New("inserting MDM ABM assets"), "missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
				}

				appleBM, err := config.MDM.AppleBM()
				if err != nil {
					initFatal(err, "parse Apple BM token, certificate and key from config")
				}

				toInsert := make([]fleet.MDMConfigAsset, 0, 2)

				found, err := checkMDMAssets([]fleet.MDMAssetName{fleet.MDMAssetABMKey, fleet.MDMAssetABMCert})
				switch {
				case err != nil:
					initFatal(err, "reading ABM assets from database")
				case !found:
					toInsert = append(toInsert, fleet.MDMConfigAsset{Name: fleet.MDMAssetABMKey, Value: appleBM.KeyPEM}, fleet.MDMConfigAsset{Name: fleet.MDMAssetABMCert, Value: appleBM.CertPEM})
				default:
					level.Warn(logger).Log("msg", "Your server already has stored ABM certificates and token. Fleet will ignore any certificates provided via environment variables when this happens.")
				}

				if len(toInsert) > 0 {
					err := ds.InsertMDMConfigAssets(context.Background(), toInsert, nil)
					switch {
					case err != nil && mysql.IsDuplicate(err):
						// we already checked for existing assets so we should never have a duplicate key error here; we'll add a debug log just in case
						level.Debug(logger).Log("msg", "unexpected duplicate key error inserting ABM assets")
					case err != nil:
						initFatal(err, "inserting ABM assets")
					default:
						// insert the ABM token without any metdata; it'll be picked by the
						// apple_mdm_dep_profile_assigner cron and backfilled
						if _, err := ds.InsertABMToken(context.Background(), &fleet.ABMToken{
							EncryptedToken: appleBM.EncryptedToken,
							RenewAt:        time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC), // 2000-01-01 is our "zero value" for time
						}); err != nil {
							initFatal(err, "save ABM token")
						}
					}
				}
			}

			appCfg, err := ds.AppConfig(context.Background())
			if err != nil {
				initFatal(err, "loading app config")
			}

			appCfg.MDM.EnabledAndConfigured = false
			appCfg.MDM.AppleBMEnabledAndConfigured = false
			if len(config.Server.PrivateKey) > 0 {
				appCfg.MDM.EnabledAndConfigured, err = checkMDMAssets([]fleet.MDMAssetName{
					fleet.MDMAssetCACert,
					fleet.MDMAssetCAKey,
					fleet.MDMAssetAPNSKey,
					fleet.MDMAssetAPNSCert,
				})
				if err != nil {
					initFatal(err, "loading MDM assets from database")
				}

				var appleBMCerts bool
				appleBMCerts, err = checkMDMAssets([]fleet.MDMAssetName{
					fleet.MDMAssetABMCert,
					fleet.MDMAssetABMKey,
				})
				if err != nil {
					initFatal(err, "loading MDM ABM assets from database")
				}
				if appleBMCerts {
					// the ABM certs are there, check if a token exists and if so, apple
					// BM is enabled and configured.
					count, err := ds.GetABMTokenCount(context.Background())
					if err != nil {
						initFatal(err, "loading MDM ABM token from database")
					}
					appCfg.MDM.AppleBMEnabledAndConfigured = count > 0
				}
			}
			if appCfg.MDM.EnabledAndConfigured {
				level.Info(logger).Log("msg", "Apple MDM enabled")
			}
			if appCfg.MDM.AppleBMEnabledAndConfigured {
				level.Info(logger).Log("msg", "Apple Business Manager enabled")
			}

			// register the Microsoft MDM services
			var (
				wstepCertManager microsoft_mdm.CertManager
			)

			// Configuring WSTEP certs
			if config.MDM.IsMicrosoftWSTEPSet() {
				_, crtPEM, keyPEM, err := config.MDM.MicrosoftWSTEP()
				if err != nil {
					initFatal(err, "validate Microsoft WSTEP certificate and key")
				}
				wstepCertManager, err = microsoft_mdm.NewCertManager(ds, crtPEM, keyPEM)
				if err != nil {
					initFatal(err, "initialize mdm microsoft wstep depot")
				}
			}

			// save the app config with the updated MDM.Enabled value
			if err := ds.SaveAppConfig(context.Background(), appCfg); err != nil {
				initFatal(err, "saving app config")
			}

			// setup mail service
			if appCfg.SMTPSettings != nil && appCfg.SMTPSettings.SMTPEnabled {
				// if SMTP is already enabled then default the backend to empty string, which fill force load the SMTP implementation
				if config.Email.EmailBackend != "" {
					config.Email.EmailBackend = ""
					level.Warn(logger).Log("msg", "SMTP is already enabled, first disable SMTP to utilize a different email backend")
				}
			}
			mailService, err := mail.NewService(config)
			if err != nil {
				level.Error(logger).Log("err", err, "msg", "failed to configure mailing service")
			}

			cronSchedules := fleet.NewCronSchedules()

			baseCtx := licensectx.NewContext(context.Background(), license)
			ctx, cancelFunc := context.WithCancel(baseCtx)
			defer cancelFunc()

			eh := errorstore.NewHandler(ctx, redisPool, logger, config.Logging.ErrorRetentionPeriod)
			ctx = ctxerr.NewContext(ctx, eh)
			svc, err := service.NewService(
				ctx,
				ds,
				task,
				resultStore,
				logger,
				&service.OsqueryLogger{
					Status: osquerydStatusLogger,
					Result: osquerydResultLogger,
				},
				config,
				mailService,
				clock.C,
				ssoSessionStore,
				liveQueryStore,
				carveStore,
				installerStore,
				failingPolicySet,
				geoIP,
				redisWrapperDS,
				depStorage,
				mdmStorage,
				mdmPushService,
				cronSchedules,
				wstepCertManager,
			)
			if err != nil {
				initFatal(err, "initializing service")
			}

			var softwareInstallStore fleet.SoftwareInstallerStore
			var bootstrapPackageStore fleet.MDMBootstrapPackageStore
			var distributedLock fleet.Lock
			if license.IsPremium() {
				profileMatcher := apple_mdm.NewProfileMatcher(redisPool)
				if config.S3.SoftwareInstallersBucket != "" {
					if config.S3.BucketsAndPrefixesMatch() {
						level.Warn(logger).Log("msg", "the S3 buckets and prefixes for carves and software installers appear to be identical, this can cause issues")
					}
					// Extract the CloudFront URL signer before creating the S3 stores.
					config.S3.ValidateCloudFrontURL(initFatal)
					if config.S3.SoftwareInstallersCloudFrontURLSigningPrivateKey != "" {
						// Strip newlines from private key
						signingPrivateKey := strings.ReplaceAll(config.S3.SoftwareInstallersCloudFrontURLSigningPrivateKey, "\\n", "\n")
						privateKey, err := cryptoutil.ParsePrivateKey([]byte(signingPrivateKey),
							"CloudFront URL signing private key")
						if err != nil {
							initFatal(err, "parsing CloudFront URL signing private key")
						}
						var ok bool
						config.S3.SoftwareInstallersCloudFrontSigner, ok = privateKey.(crypto.Signer)
						if !ok {
							initFatal(errors.New("CloudFront URL signing private key is not a crypto.Signer"),
								"parsing CloudFront URL signing private key")
						}
					}
					store, err := s3.NewSoftwareInstallerStore(config.S3)
					if err != nil {
						initFatal(err, "initializing S3 software installer store")
					}
					softwareInstallStore = store
					level.Info(logger).Log("msg", "using S3 software installer store", "bucket", config.S3.SoftwareInstallersBucket)

					bstore, err := s3.NewBootstrapPackageStore(config.S3)
					if err != nil {
						initFatal(err, "initializing S3 bootstrap package store")
					}
					bootstrapPackageStore = bstore
					level.Info(logger).Log("msg", "using S3 bootstrap package store", "bucket", config.S3.SoftwareInstallersBucket)

				} else {
					installerDir := os.TempDir()
					if dir := os.Getenv("FLEET_SOFTWARE_INSTALLER_STORE_DIR"); dir != "" {
						installerDir = dir
					}
					store, err := filesystem.NewSoftwareInstallerStore(installerDir)
					if err != nil {
						level.Error(logger).Log("err", err, "msg", "failed to configure local filesystem software installer store")
						softwareInstallStore = fleet.FailingSoftwareInstallerStore{}
					} else {
						softwareInstallStore = store
						level.Info(logger).Log("msg", "using local filesystem software installer store, this is not suitable for production use", "directory", installerDir)
					}
				}

				distributedLock = redis_lock.NewLock(redisPool)
				svc, err = eeservice.NewService(
					svc,
					ds,
					logger,
					config,
					mailService,
					clock.C,
					depStorage,
					apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService),
					ssoSessionStore,
					profileMatcher,
					softwareInstallStore,
					bootstrapPackageStore,
					distributedLock,
					redis_key_value.New(redisPool),
				)
				if err != nil {
					initFatal(err, "initial Fleet Premium service")
				}
			}

			instanceID, err := server.GenerateRandomText(64)
			if err != nil {
				initFatal(errors.New("Error generating random instance identifier"), "")
			}
			level.Info(logger).Log("instanceID", instanceID)

			// Perform a cleanup of cron_stats outside of the cronSchedules because the
			// schedule package uses cron_stats entries to decide whether a schedule will
			// run or not (see https://github.com/fleetdm/fleet/issues/9486).
			go func() {
				cleanupCronStats := func() {
					level.Debug(logger).Log("msg", "cleaning up cron_stats")
					// Datastore.CleanupCronStats should be safe to run by multiple fleet
					// instances at the same time and it should not be an expensive operation.
					if err := ds.CleanupCronStats(ctx); err != nil {
						level.Info(logger).Log("msg", "failed to clean up cron_stats", "err", err)
					}
				}

				cleanupCronStats()

				cleanUpCronStatsTick := time.NewTicker(1 * time.Hour)
				defer cleanUpCronStatsTick.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-cleanUpCronStatsTick.C:
						cleanupCronStats()
					}
				}
			}()

			if softwareInstallStore != nil {
				if err := cronSchedules.StartCronSchedule(
					func() (fleet.CronSchedule, error) {
						return cronUninstallSoftwareMigration(ctx, instanceID, ds, softwareInstallStore, logger)
					},
				); err != nil {
					initFatal(err, fmt.Sprintf("failed to register %s", fleet.CronUninstallSoftwareMigration))
				}
			}

			if config.Server.FrequentCleanupsEnabled {
				if err := cronSchedules.StartCronSchedule(
					func() (fleet.CronSchedule, error) {
						return newFrequentCleanupsSchedule(ctx, instanceID, ds, liveQueryStore, logger)
					},
				); err != nil {
					initFatal(err, "failed to register frequent_cleanups schedule")
				}
			}

			if err := cronSchedules.StartCronSchedule(
				func() (fleet.CronSchedule, error) {
					commander := apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService)
					return newCleanupsAndAggregationSchedule(
						ctx, instanceID, ds, logger, redisWrapperDS, &config, commander, softwareInstallStore, bootstrapPackageStore,
					)
				},
			); err != nil {
				initFatal(err, "failed to register cleanups_then_aggregations schedule")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newUsageStatisticsSchedule(ctx, instanceID, ds, config, license, logger)
			}); err != nil {
				initFatal(err, "failed to register stats schedule")
			}

			vulnerabilityScheduleDisabled := false
			if config.Vulnerabilities.DisableSchedule {
				vulnerabilityScheduleDisabled = true
				level.Info(logger).Log("msg", "vulnerabilities schedule disabled via vulnerabilities.disable_schedule")
			}
			if config.Vulnerabilities.CurrentInstanceChecks == "no" || config.Vulnerabilities.CurrentInstanceChecks == "0" {
				level.Info(logger).Log("msg", "vulnerabilities schedule disabled via vulnerabilities.current_instance_checks")
				vulnerabilityScheduleDisabled = true
			}
			if !vulnerabilityScheduleDisabled {
				// vuln processing by default is run by internal cron mechanism
				if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
					return newVulnerabilitiesSchedule(ctx, instanceID, ds, logger, &config.Vulnerabilities)
				}); err != nil {
					initFatal(err, "failed to register vulnerabilities schedule")
				}
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newAutomationsSchedule(ctx, instanceID, ds, logger, 5*time.Minute, failingPolicySet)
			}); err != nil {
				initFatal(err, "failed to register automations schedule")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				commander := apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService)
				return newWorkerIntegrationsSchedule(ctx, instanceID, ds, logger, depStorage, commander, bootstrapPackageStore)
			}); err != nil {
				initFatal(err, "failed to register worker integrations schedule")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newAppleMDMDEPProfileAssigner(ctx, instanceID, config.MDM.AppleDEPSyncPeriodicity, ds, depStorage, logger)
			}); err != nil {
				initFatal(err, "failed to register apple_mdm_dep_profile_assigner schedule")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newAppleMDMProfileManagerSchedule(
					ctx,
					instanceID,
					ds,
					apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService),
					logger,
				)
			}); err != nil {
				initFatal(err, "failed to register mdm_apple_profile_manager schedule")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newWindowsMDMProfileManagerSchedule(
					ctx,
					instanceID,
					ds,
					logger,
				)
			}); err != nil {
				initFatal(err, "failed to register mdm_windows_profile_manager schedule")
			}

			if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
				return newMDMAPNsPusher(
					ctx,
					instanceID,
					ds,
					apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService),
					logger,
				)
			}); err != nil {
				initFatal(err, "failed to register APNs pusher schedule")
			}

			if license.IsPremium() {
				if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
					commander := apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService)
					return newIPhoneIPadRefetcher(ctx, instanceID, 10*time.Minute, ds, commander, logger)
				}); err != nil {
					initFatal(err, "failed to register apple_mdm_iphone_ipad_refetcher schedule")
				}

				if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
					return newMaintainedAppSchedule(ctx, instanceID, ds, logger)
				}); err != nil {
					initFatal(err, "failed to register maintained apps schedule")
				}
			}

			if license.IsPremium() && config.Activity.EnableAuditLog {
				if err := cronSchedules.StartCronSchedule(func() (fleet.CronSchedule, error) {
					return newActivitiesStreamingSchedule(ctx, instanceID, ds, logger, auditLogger)
				}); err != nil {
					initFatal(err, "failed to register activities streaming schedule")
				}
			}

			if license.IsPremium() {
				if err := cronSchedules.StartCronSchedule(
					func() (fleet.CronSchedule, error) {
						if config.Calendar.Periodicity > 0 {
							config.Calendar.SetAlwaysReloadEvent(true)
						} else {
							config.Calendar.Periodicity = 5 * time.Minute
						}
						return cron.NewCalendarSchedule(ctx, instanceID, ds, distributedLock, config.Calendar, logger)
					},
				); err != nil {
					initFatal(err, "failed to register calendar schedule")
				}
			}

			level.Info(logger).Log("msg", fmt.Sprintf("started cron schedules: %s", strings.Join(cronSchedules.ScheduleNames(), ", ")))

			// StartCollectors starts a goroutine per collector, using ctx to cancel.
			task.StartCollectors(ctx, kitlog.With(logger, "cron", "async_task"))

			// Flush seen hosts every second
			hostsAsyncCfg := config.Osquery.AsyncConfigForTask(configpkg.AsyncTaskHostLastSeen)
			if !hostsAsyncCfg.Enabled {
				go func() {
					for range time.Tick(time.Duration(rand.Intn(10)+1) * time.Second) {
						if err := task.FlushHostsLastSeen(baseCtx, clock.C.Now()); err != nil {
							level.Info(logger).Log(
								"err", err,
								"msg", "failed to update host seen times",
							)
						}
					}
				}()
			}

			fieldKeys := []string{"method", "error"}
			requestCount := kitprometheus.NewCounterFrom(prometheus.CounterOpts{
				Namespace: "api",
				Subsystem: "service",
				Name:      "request_count",
				Help:      "Number of requests received.",
			}, fieldKeys)
			requestLatency := kitprometheus.NewSummaryFrom(prometheus.SummaryOpts{
				Namespace: "api",
				Subsystem: "service",
				Name:      "request_latency_microseconds",
				Help:      "Total duration of requests in microseconds.",
			}, fieldKeys)

			svc = service.NewMetricsService(svc, requestCount, requestLatency)

			httpLogger := kitlog.With(logger, "component", "http")

			limiterStore := &redis.ThrottledStore{
				Pool:      redisPool,
				KeyPrefix: "ratelimit::",
			}

			var apiHandler, frontendHandler, endUserEnrollOTAHandler http.Handler
			{
				frontendHandler = service.PrometheusMetricsHandler(
					"get_frontend",
					service.ServeFrontend(config.Server.URLPrefix, config.Server.SandboxEnabled, httpLogger),
				)

				frontendHandler = service.WithMDMEnrollmentMiddleware(svc, httpLogger, frontendHandler)

				apiHandler = service.MakeHandler(svc, config, httpLogger, limiterStore)

				setupRequired, err := svc.SetupRequired(baseCtx)
				if err != nil {
					initFatal(err, "fetching setup requirement")
				}
				// WithSetup will check if first time setup is required
				// By performing the same check inside main, we can make server startups
				// more efficient after the first startup.
				if setupRequired {
					apiHandler = service.WithSetup(svc, logger, apiHandler)
					frontendHandler = service.RedirectLoginToSetup(svc, logger, frontendHandler, config.Server.URLPrefix)
				} else {
					frontendHandler = service.RedirectSetupToLogin(svc, logger, frontendHandler, config.Server.URLPrefix)
				}

				endUserEnrollOTAHandler = service.ServeEndUserEnrollOTA(svc, config.Server.URLPrefix, logger)
			}

			healthCheckers := make(map[string]health.Checker)
			{
				// a list of dependencies which could affect the status of the app if unavailable.
				deps := map[string]interface{}{
					"mysql": ds,
					"redis": resultStore,
				}

				// convert all dependencies to health.Checker if they implement the healthz methods.
				for name, dep := range deps {
					if hc, ok := dep.(health.Checker); ok {
						healthCheckers[name] = hc
					} else {
						initFatal(errors.New(name+" should be a health.Checker"), "initializing health checks")
					}
				}

			}

			// Instantiate a gRPC service to handle launcher requests.
			launcher := launcher.New(svc, logger, grpc.NewServer(), healthCheckers)

			rootMux := http.NewServeMux()
			rootMux.Handle("/healthz", service.PrometheusMetricsHandler("healthz", health.Handler(httpLogger, healthCheckers)))
			rootMux.Handle("/version", service.PrometheusMetricsHandler("version", version.Handler()))
			rootMux.Handle("/assets/", service.PrometheusMetricsHandler("static_assets", service.ServeStaticAssets("/assets/")))

			if len(config.Server.PrivateKey) > 0 {
				commander := apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService)
				ddmService := service.NewMDMAppleDDMService(ds, logger)
				mdmCheckinAndCommandService := service.NewMDMAppleCheckinAndCommandService(ds, commander, logger)

				hasSCEPChallenge, err := checkMDMAssets([]fleet.MDMAssetName{fleet.MDMAssetSCEPChallenge})
				if err != nil {
					initFatal(err, "checking SCEP challenge in database")
				}
				if !hasSCEPChallenge {
					scepChallenge := config.MDM.AppleSCEPChallenge
					if scepChallenge == "" {
						scepChallenge = uuid.NewString()
					}

					err = ds.InsertMDMConfigAssets(context.Background(), []fleet.MDMConfigAsset{
						{Name: fleet.MDMAssetSCEPChallenge, Value: []byte(scepChallenge)},
					}, nil)
					if err != nil {
						// duplicate key errors mean that we already
						// have a value for those keys in the
						// database, fail to initalize on other
						// cases.
						if !mysql.IsDuplicate(err) {
							initFatal(err, "inserting SCEP challenge")
						}

						level.Warn(logger).Log("msg", "Your server already has stored a SCEP challenge. Fleet will ignore this value provided via environment variables when this happens.")
					}
				}
				if err := service.RegisterAppleMDMProtocolServices(
					rootMux,
					config.MDM,
					mdmStorage,
					scepStorage,
					logger,
					mdmCheckinAndCommandService,
					ddmService,
					commander,
				); err != nil {
					initFatal(err, "setup mdm apple services")
				}
			}

			// SCEP proxy (for NDES, etc.)
			if license.IsPremium() {
				if err = service.RegisterSCEPProxy(rootMux, ds, logger); err != nil {
					initFatal(err, "setup SCEP proxy")
				}
			}

			if config.Prometheus.BasicAuth.Username != "" && config.Prometheus.BasicAuth.Password != "" {
				rootMux.Handle("/metrics", basicAuthHandler(
					config.Prometheus.BasicAuth.Username,
					config.Prometheus.BasicAuth.Password,
					service.PrometheusMetricsHandler("metrics", promhttp.Handler()),
				))
			} else {
				if config.Prometheus.BasicAuth.Disable {
					level.Info(logger).Log("msg", "metrics endpoint enabled with http basic auth disabled")
					rootMux.Handle("/metrics", service.PrometheusMetricsHandler("metrics", promhttp.Handler()))
				} else {
					level.Info(logger).Log("msg", "metrics endpoint disabled (http basic auth credentials not set)")
				}
			}

			// We must wrap the Handler here to set special per-endpoint Read/Write
			// timeouts, so that we have access to the raw http.ResponseWriter.
			// Otherwise, the handler is wrapped by the promhttp response delegator,
			// which does not support the Unwrap call needed to work with
			// ResponseController.
			//
			// See https://pkg.go.dev/net/http#NewResponseController which explains
			// the Unwrap method that the prometheus wrapper of http.ResponseWriter
			// does not implement.
			rootMux.HandleFunc("/api/", func(rw http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/fleet/scripts/run/sync") {
					// when running a script synchronously, we wait a while for a script
					// execution result, so the write timeout (to write the response)
					// must be extended.
					rc := http.NewResponseController(rw)
					// add an additional 30 seconds to prevent race conditions where the
					// request is terminated early.
					if err := rc.SetWriteDeadline(time.Now().Add(scripts.MaxServerWaitTime + (30 * time.Second))); err != nil {
						level.Error(logger).Log("msg", "http middleware failed to override endpoint write timeout", "err", err)
					}
				}

				if (req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/fleet/software/package")) ||
					(req.Method == http.MethodPatch && strings.HasSuffix(req.URL.Path, "/package") && strings.Contains(req.URL.Path, "/fleet/software/titles/")) ||
					(req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/bootstrap")) {
					var zeroTime time.Time
					rc := http.NewResponseController(rw)
					// For large software installers and bootstrap packages, the server time needs time to read the full
					// request body so we use the zero value to remove the deadline and override the
					// default read timeout.
					// TODO: Is this really how we want to handle this? Or would an arbitrarily long
					// timeout be better?
					if err := rc.SetReadDeadline(zeroTime); err != nil {
						level.Error(logger).Log("msg", "http middleware failed to override endpoint read timeout", "err", err)
					}
					// For large software installers, the server time needs time to store the
					// installer to S3 (or the configured storage location) and write the response
					// body so we use the zero value to remove the deadline and override the
					// default write timeout.
					// TODO: Is this really how we want to handle this? Or would an arbitrarily long
					// timeout be better?
					if err := rc.SetWriteDeadline(zeroTime); err != nil {
						level.Error(logger).Log("msg", "http middleware failed to override endpoint write timeout", "err", err)
					}
					req.Body = http.MaxBytesReader(rw, req.Body, fleet.MaxSoftwareInstallerSize)
				}
				apiHandler.ServeHTTP(rw, req)
			})

			rootMux.Handle("/enroll", endUserEnrollOTAHandler)
			rootMux.Handle("/", frontendHandler)

			debugHandler := &debugMux{
				fleetAuthenticatedHandler: service.MakeDebugHandler(svc, config, logger, eh, ds),
			}
			rootMux.Handle("/debug/", debugHandler)

			if path, ok := os.LookupEnv("FLEET_TEST_PAGE_PATH"); ok {
				// test that we can load this
				_, err := os.ReadFile(path)
				if err != nil {
					initFatal(err, "loading FLEET_TEST_PAGE_PATH")
				}
				rootMux.HandleFunc("/test", func(rw http.ResponseWriter, req *http.Request) {
					testPage, err := os.ReadFile(path)
					if err != nil {
						rw.WriteHeader(http.StatusNotFound)
						return
					}
					rw.Write(testPage) //nolint:errcheck
					rw.WriteHeader(http.StatusOK)
				})
			}

			if debug {
				// Add debug endpoints with a random
				// authorization token
				debugToken, err := server.GenerateRandomText(24)
				if err != nil {
					initFatal(err, "generating debug token")
				}
				debugHandler.tokenAuthenticatedHandler = http.StripPrefix("/debug/", netbug.AuthHandler(debugToken))
				fmt.Printf("*** Debug mode enabled ***\nAccess the debug endpoints at /debug/?token=%s\n", url.QueryEscape(debugToken))
			}

			if len(config.Server.URLPrefix) > 0 {
				prefixMux := http.NewServeMux()
				prefixMux.Handle(config.Server.URLPrefix+"/", http.StripPrefix(config.Server.URLPrefix, rootMux))
				rootMux = prefixMux
			}

			// NOTE(lucas): It seems we missed updating this value from 90s (see #1798) to 25s after we
			// decided to make the synchronous live query API to take up to 25 seconds.
			// Not changing this to not break any long running requests (like when uploading software
			// packages via GitOps).
			liveQueryRestPeriod := 90 * time.Second
			if v := os.Getenv("FLEET_LIVE_QUERY_REST_PERIOD"); v != "" {
				duration, err := time.ParseDuration(v)
				if err != nil {
					level.Error(logger).Log("live_query_rest_period_err", err)
				} else {
					liveQueryRestPeriod = duration
				}
			}

			// The "GET /api/latest/fleet/queries/run" API requires
			// WriteTimeout to be higher than the live query rest period
			// (otherwise the response is not sent back to the client).
			//
			// We add 10s to the live query rest period to allow the writing
			// of the response.
			liveQueryRestPeriod += 10 * time.Second

			// Create the handler based on whether tracing should be there
			var handler http.Handler
			if config.Logging.TracingEnabled && config.Logging.TracingType == "elasticapm" {
				handler = launcher.Handler(apmhttp.Wrap(rootMux))
			} else {
				handler = launcher.Handler(rootMux)
			}

			srv := config.Server.DefaultHTTPServer(ctx, handler)
			if liveQueryRestPeriod > srv.WriteTimeout {
				srv.WriteTimeout = liveQueryRestPeriod
			}
			srv.SetKeepAlivesEnabled(config.Server.Keepalive)
			errs := make(chan error, 2)
			go func() {
				if !config.Server.TLS {
					logger.Log("transport", "http", "address", config.Server.Address, "msg", "listening")
					errs <- srv.ListenAndServe()
				} else {
					logger.Log("transport", "https", "address", config.Server.Address, "msg", "listening")
					srv.TLSConfig = getTLSConfig(config.Server.TLSProfile)
					errs <- srv.ListenAndServeTLS(
						config.Server.Cert,
						config.Server.Key,
					)
				}
			}()
			go func() {
				sig := make(chan os.Signal, 1)
				signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
				<-sig // block on signal
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				errs <- func() error {
					cancelFunc()
					cleanupCronStatsOnShutdown(ctx, ds, logger, instanceID)
					launcher.GracefulStop()
					return srv.Shutdown(ctx)
				}()
			}()

			// block on errs signal
			logger.Log("terminated", <-errs)
		},
	}

	serveCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug endpoints")
	serveCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")
	serveCmd.PersistentFlags().BoolVar(&devLicense, "dev_license", false, "Enable development license")
	serveCmd.PersistentFlags().BoolVar(&devExpiredLicense, "dev_expired_license", false, "Enable expired development license")

	return serveCmd
}

func initLicense(config configpkg.FleetConfig, devLicense, devExpiredLicense bool) (*fleet.LicenseInfo, error) {
	if devLicense {
		// This license key is valid for development only
		config.License.Key = "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNzUxMjQxNjAwLCJzdWIiOiJkZXZlbG9wbWVudC1vbmx5IiwiZGV2aWNlcyI6MTAwLCJub3RlIjoiZm9yIGRldmVsb3BtZW50IG9ubHkiLCJ0aWVyIjoicHJlbWl1bSIsImlhdCI6MTY1NjY5NDA4N30.dvfterOvfTGdrsyeWYH9_lPnyovxggM5B7tkSl1q1qgFYk_GgOIxbaqIZ6gJlL0cQuBF9nt5NgV0AUT9RmZUaA"
	} else if devExpiredLicense {
		// An expired license key
		config.License.Key = "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNjI5NzYzMjAwLCJzdWIiOiJEZXYgbGljZW5zZSAoZXhwaXJlZCkiLCJkZXZpY2VzIjo1MDAwMDAsIm5vdGUiOiJUaGlzIGxpY2Vuc2UgaXMgdXNlZCB0byBmb3IgZGV2ZWxvcG1lbnQgcHVycG9zZXMuIiwidGllciI6ImJhc2ljIiwiaWF0IjoxNjI5OTA0NzMyfQ.AOppRkl1Mlc_dYKH9zwRqaTcL0_bQzs7RM3WSmxd3PeCH9CxJREfXma8gm0Iand6uIWw8gHq5Dn0Ivtv80xKvQ"
	}
	return licensing.LoadLicense(config.License.Key)
}

// basicAuthHandler wraps the given handler behind HTTP Basic Auth.
func basicAuthHandler(username, password string, next http.Handler) http.HandlerFunc {
	hashFn := func(s string) []byte {
		h := sha256.Sum256([]byte(s))
		return h[:]
	}
	expectedUsernameHash := hashFn(username)
	expectedPasswordHash := hashFn(password)

	return func(w http.ResponseWriter, r *http.Request) {
		recvUsername, recvPassword, ok := r.BasicAuth()
		if ok {
			usernameMatch := subtle.ConstantTimeCompare(hashFn(recvUsername), expectedUsernameHash) == 1
			passwordMatch := subtle.ConstantTimeCompare(hashFn(recvPassword), expectedPasswordHash) == 1

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

// Support for TLS security profiles, we set up the TLS configuation based on
// value supplied to server_tls_compatibility command line flag. The default
// profile is 'modern'.
// See https://wiki.mozilla.org/index.php?title=Security/Server_Side_TLS&oldid=1229478
func getTLSConfig(profile string) *tls.Config {
	cfg := tls.Config{
		PreferServerCipherSuites: true,
	}

	switch profile {
	case configpkg.TLSProfileModern:
		cfg.MinVersion = tls.VersionTLS13
		cfg.CurvePreferences = append(cfg.CurvePreferences,
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		)
		cfg.CipherSuites = append(cfg.CipherSuites,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			// These cipher suites not explicitly listed by Mozilla, but
			// required by Go's HTTP/2 implementation
			// See: https://go-review.googlesource.com/c/net/+/200317/
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		)
	case configpkg.TLSProfileIntermediate:
		cfg.MinVersion = tls.VersionTLS12
		cfg.CurvePreferences = append(cfg.CurvePreferences,
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		)
		cfg.CipherSuites = append(cfg.CipherSuites,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		)
	default:
		initFatal(
			fmt.Errorf("%s is invalid", profile),
			"set TLS profile",
		)
	}

	return &cfg
}

// devSQLInterceptor is a sql interceptor to be used for development purposes.
type devSQLInterceptor struct {
	sqlmw.NullInterceptor

	logger kitlog.Logger
}

func (in *devSQLInterceptor) ConnQueryContext(ctx context.Context, conn driver.QueryerContext, query string, args []driver.NamedValue) (driver.Rows, error) {
	start := time.Now()
	rows, err := conn.QueryContext(ctx, query, args)
	in.logQuery(start, query, args, err)
	return rows, err
}

func (in *devSQLInterceptor) StmtQueryContext(ctx context.Context, stmt driver.StmtQueryContext, query string, args []driver.NamedValue) (driver.Rows, error) {
	start := time.Now()
	rows, err := stmt.QueryContext(ctx, args)
	in.logQuery(start, query, args, err)
	return rows, err
}

func (in *devSQLInterceptor) StmtExecContext(ctx context.Context, stmt driver.StmtExecContext, query string, args []driver.NamedValue) (driver.Result, error) {
	start := time.Now()
	result, err := stmt.ExecContext(ctx, args)
	in.logQuery(start, query, args, err)
	return result, err
}

var spaceRegex = regexp.MustCompile(`\s+`)

func (in *devSQLInterceptor) logQuery(start time.Time, query string, args []driver.NamedValue, err error) {
	logLevel := level.Debug
	if err != nil {
		logLevel = level.Error
	}
	query = strings.TrimSpace(spaceRegex.ReplaceAllString(query, " "))
	logLevel(in.logger).Log("duration", time.Since(start), "query", query, "args", argsToString(args), "err", err)
}

func argsToString(args []driver.NamedValue) string {
	var allArgs strings.Builder
	allArgs.WriteString("{")
	for i, arg := range args {
		if i > 0 {
			allArgs.WriteString(", ")
		}
		if arg.Name != "" {
			allArgs.WriteString(fmt.Sprintf("%s=", arg.Name))
		}
		allArgs.WriteString(fmt.Sprintf("%v", arg.Value))
	}
	allArgs.WriteString("}")
	return allArgs.String()
}

// The debugMux directs the request to either the fleet-authenticated handler,
// which is the standard handler for debug endpoints (using a Fleet
// authorization bearer token), or to the token-authenticated handler if a
// query-string token is provided and such a handler is set. The only wayt to
// set this handler is if the --debug flag was provided to the fleet serve
// command.
type debugMux struct {
	fleetAuthenticatedHandler http.Handler
	tokenAuthenticatedHandler http.Handler
}

func (m *debugMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Has("token") && m.tokenAuthenticatedHandler != nil {
		m.tokenAuthenticatedHandler.ServeHTTP(w, r)
		return
	}
	m.fleetAuthenticatedHandler.ServeHTTP(w, r)
}

// nopPusher is a no-op push.Pusher.
type nopPusher struct{}

var _ push.Pusher = nopPusher{}

// Push implements push.Pusher.
func (n nopPusher) Push(context.Context, []string) (map[string]*push.Response, error) {
	return nil, nil
}

func createTestBucketForInstallers(config *configpkg.FleetConfig, logger log.Logger) {
	store, err := s3.NewSoftwareInstallerStore(config.S3)
	if err != nil {
		initFatal(err, "initializing S3 software installer store")
	}
	if err := store.CreateTestBucket(config.S3.SoftwareInstallersBucket); err != nil {
		// Don't panic, allow devs to run Fleet without minio/S3 dependency.
		level.Info(logger).Log(
			"err", err,
			"msg", "failed to create test bucket",
			"name", config.S3.SoftwareInstallersBucket,
		)
	}
}
