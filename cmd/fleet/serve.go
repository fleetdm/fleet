package main

import (
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/ee/server/licensing"
	"github.com/fleetdm/fleet/v4/ee/server/scim"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/ee/server/service/condaccess"
	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/ee/server/service/est"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/httpsig"
	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/str"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/acl/acmeacl"
	"github.com/fleetdm/fleet/v4/server/acl/activityacl"
	"github.com/fleetdm/fleet/v4/server/acl/chartacl"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	activity_bootstrap "github.com/fleetdm/fleet/v4/server/activity/bootstrap"
	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/chart"
	chart_api "github.com/fleetdm/fleet/v4/server/chart/api"
	chart_bootstrap "github.com/fleetdm/fleet/v4/server/chart/bootstrap"
	configpkg "github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	licensectx "github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/datastore/failing"
	"github.com/fleetdm/fleet/v4/server/datastore/filesystem"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysqlredis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/errorstore"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/health"
	"github.com/fleetdm/fleet/v4/server/launcher"
	"github.com/fleetdm/fleet/v4/server/live_query"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	acme_api "github.com/fleetdm/fleet/v4/server/mdm/acme/api"
	acme_bootstrap "github.com/fleetdm/fleet/v4/server/mdm/acme/bootstrap"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/cryptoutil"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	platform_logging "github.com/fleetdm/fleet/v4/server/platform/logging"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/platform/tracing"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_microsoft_proxy"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/log"
	otelmw "github.com/fleetdm/fleet/v4/server/service/middleware/otel"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/fleetdm/fleet/v4/server/service/redis_lock"
	"github.com/fleetdm/fleet/v4/server/service/redis_policy_set"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/fleetdm/fleet/v4/server/version"
	"github.com/getsentry/sentry-go"
	"github.com/go-kit/kit/endpoint"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/google/uuid"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/ngrok/sqlmw"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/throttled/throttled/v2"
	"go.elastic.co/apm/module/apmhttp/v2"
	_ "go.elastic.co/apm/module/apmsql/v2"
	_ "go.elastic.co/apm/module/apmsql/v2/mysql"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // Because we use gzip compression for OTLP
)

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
		// runServeCmd is a named function so that NilAway can analyze it for nil-safety.
		Run: func(cmd *cobra.Command, args []string) {
			runServeCmd(cmd, configManager, debug, devLicense, devExpiredLicense)
		},
	}

	serveCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug endpoints")
	serveCmd.PersistentFlags().BoolVar(&dev_mode.IsEnabled, "dev", false, "Enable developer options")
	serveCmd.PersistentFlags().BoolVar(&devLicense, "dev_license", false, "Enable development license")
	serveCmd.PersistentFlags().BoolVar(&devExpiredLicense, "dev_expired_license", false, "Enable expired development license")

	return serveCmd
}

// networkBlockingModeFor decides the outbound network-blocking mode for
// integration HTTP requests. BypassNetworkBlocking is an infra-level escape
// hatch, only settable via server startup config, never a runtime admin
// operation.
func networkBlockingModeFor(devModeEnabled bool, serverConfig configpkg.ServerConfig) fleethttp.NetworkBlockingMode {
	switch {
	case devModeEnabled, serverConfig.BypassNetworkBlocking:
		return fleethttp.BlockingBypassAll
	case serverConfig.AllowPrivateNetworkIntegrations:
		return fleethttp.BlockingPrivateAllowed
	default:
		return fleethttp.BlockingFull
	}
}

// runServeCmd is a named function so that NilAway can analyze it for nil-safety.
func runServeCmd(cmd *cobra.Command, configManager configpkg.Manager, debug, devLicense, devExpiredLicense bool) {
	config := configManager.LoadConfig()

	if dev_mode.IsEnabled {
		applyDevFlags(&config)
	}

	// Set network blocking mode for outbound integration requests.
	fleethttp.SetNetworkBlockingMode(networkBlockingModeFor(dev_mode.IsEnabled, config.Server))

	license, err := initLicense(&config, devLicense, devExpiredLicense)
	if err != nil {
		initFatal(
			err,
			"failed to load license - for help use https://fleetdm.com/contact",
		)
	}

	if license != nil && license.IsPremium() && license.IsExpired() {
		fleet.WriteExpiredLicenseBanner(os.Stderr)
	}

	// Validate OTEL server options
	config.Logging.Validate(initFatal)

	// Trace sampler tier registry. Bounded contexts register their own route to tier classifications below. The
	// platform/tracing package stays free of cross context coupling. Infra paths (/healthz, /version, /metrics) are registered
	// alongside their WrapHandler calls further down in this function.
	traceRegistry := tracing.NewRegistry()
	service.RegisterTracingTiers(traceRegistry)
	activity_bootstrap.RegisterTracingTiers(traceRegistry)
	// Future bounded contexts: each exposes its own RegisterTracingTiers.

	// Init OTEL providers (traces, metrics, logs) and the route aware sampler.
	loggerProvider, tracerProvider, meterProvider, traceSampler := initOTELProviders(config, traceRegistry, initFatal)

	logger := initLogger(config, loggerProvider)

	// If you want to disable any logs by default, this is where to do it.
	//
	// For example:
	// platform_logging.DisableTopic("deprecated-api-keys")

	// Apply log topic overrides from config. Enables run first, then
	// disables, so disable wins on conflict.
	// Note that any topic not included in these lists will be considered
	// enabled if it's encountered in a log.
	for _, topic := range str.SplitAndTrim(config.Logging.EnableLogTopics, ",", true) {
		platform_logging.EnableTopic(topic)
	}
	for _, topic := range str.SplitAndTrim(config.Logging.DisableLogTopics, ",", true) {
		platform_logging.DisableTopic(topic)
	}

	if dev_mode.IsEnabled {
		createTestBuckets(cmd.Context(), &config, logger)
	}

	config.Osquery.Validate(initFatal)

	config.ConditionalAccess.Validate(initFatal)

	config.Server.NormalizeURLPrefix()
	config.Server.ValidateURLPrefix(initFatal)

	// Handle server private key configuration - either direct or via AWS Secrets Manager.
	config.Server.Validate(initFatal)

	// Retrieve private key from AWS Secrets Manager if specified
	if config.Server.PrivateKeySecretArn != "" {
		privateKey, err := configpkg.RetrieveSecretsManagerSecret(
			context.Background(),
			config.Server.PrivateKeySecretArn,
			config.Server.PrivateKeySecretRegion,
			config.Server.PrivateKeySecretSTSAssumeRoleArn,
			config.Server.PrivateKeySecretSTSExternalID,
		)
		if err != nil {
			initFatal(err, "retrieve private key from secrets manager")
		}
		config.Server.PrivateKey = privateKey
	}

	config.Server.ValidatePrivateKeyLength(initFatal)
	if len(config.Server.PrivateKey) > 0 {
		// Truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK;
		// some infra setups generate keys that are longer than 32 bytes.
		config.Server.PrivateKey = config.Server.PrivateKey[:32]
	}

	if config.MDM.CertificateProfilesLimit < 0 {
		config.MDM.CertificateProfilesLimit = 0
	}

	// Configure default max request body size based on config
	platform_http.MaxRequestBodySize = config.Server.DefaultMaxRequestBodySize

	mds, dbConns, carveStore := initDatastore(config, logger, clock.C, initFatal)
	if mds == nil {
		initFatal(errors.New("datastore was nil after initialization"), "initializing datastore")
		return
	}
	var ds fleet.Datastore = mds

	migrationStatus, err := ds.MigrationStatus(cmd.Context())
	if err != nil {
		initFatal(err, "retrieving migration status")
	}
	if migrationStatus == nil {
		initFatal(errors.New("migration status was nil"), "retrieving migration status")
		return
	}
	if evalMigrationStatus(migrationStatus, dev_mode.IsEnabled, config.Upgrades.AllowMissingMigrations) {
		os.Exit(1)
	}

	if initializingDS, ok := ds.(initializer); ok {
		if err := initializingDS.Initialize(); err != nil {
			initFatal(err, "loading built in data")
		}
	}

	var redisPool fleet.RedisPool
	var redisWrapperDS *mysqlredis.Datastore
	redisPool, ds, redisWrapperDS = initRedis(cmd.Context(), config, license, ds, logger, initFatal)
	if redisPool == nil {
		initFatal(errors.New("redis pool was nil after initialization"), "initialize Redis")
		return
	}

	resultStore := pubsub.NewRedisQueryResults(redisPool, config.Redis.DuplicateResults,
		logger.With("component", "query-results"),
	)
	liveQueryStore := live_query.NewRedisLiveQuery(redisPool, logger, liveQueryMemCacheDuration,
		config.Redis.LiveQuerySmallTargetThreshold)
	ssoSessionStore := sso.NewSessionStore(redisPool)

	osquerydStatusLogger, osquerydResultLogger, auditLogger := initOsqueryLogging(cmd.Context(), config, license, logger, initFatal)
	if osquerydStatusLogger == nil || osquerydResultLogger == nil {
		initFatal(errors.New("osquery loggers were nil after initialization"), "initializing osqueryd logging")
		return
	}

	failingPolicySet := redis_policy_set.NewFailing(redisPool)

	task := async.NewTask(ds, redisPool, clock.C, &config)

	if config.Sentry.Dsn != "" {
		v := version.Version()
		err = sentry.Init(sentry.ClientOptions{
			Dsn:     config.Sentry.Dsn,
			Release: fmt.Sprintf("%s_%s_%s", v.Version, v.Branch, v.Revision),
		})
		if err != nil {
			initFatal(err, "initializing sentry")
		}
		logger.InfoContext(cmd.Context(), "sentry initialized", "dsn", config.Sentry.Dsn)

		defer sentry.Recover()
		defer sentry.Flush(2 * time.Second)
	}

	geoIP := initGeoIP(cmd.Context(), config, logger)

	if config.MDM.EnableCustomOSUpdatesAndFileVault && !license.IsPremium() {
		config.MDM.EnableCustomOSUpdatesAndFileVault = false
		logger.WarnContext(cmd.Context(), "Disabling custom OS updates and FileVault management because Fleet Premium license is not present")
	}

	if config.MDM.EnableCustomFileVault && !license.IsPremium() {
		config.MDM.EnableCustomFileVault = false
		logger.WarnContext(cmd.Context(), "Disabling custom FileVault management because Fleet Premium license is not present")
	}

	mdmStorage, depStorage, scepStorage := initAppleMDMStorages(mds, initFatal)

	mdmPushService := initAppleMDMPushService(mdmStorage, logger)
	mds.WithPusher(mdmPushService)

	// reconcile Apple MDM and Business Manager configuration with the database
	reconcileAppleMDMAPNsAndSCEPAssets(context.Background(), config, ds, logger, initFatal)
	reconcileAppleMDMABMAssets(context.Background(), config, ds, logger, initFatal)

	appCfg, err := ds.AppConfig(context.Background())
	if err != nil {
		initFatal(err, "loading app config")
	}

	appCfg.MDM.EnabledAndConfigured = false
	appCfg.MDM.AppleBMEnabledAndConfigured = false
	if len(config.Server.PrivateKey) > 0 {
		appCfg.MDM.EnabledAndConfigured, err = checkMDMAssetsExist(context.Background(), ds, []fleet.MDMAssetName{
			fleet.MDMAssetCACert,
			fleet.MDMAssetCAKey,
			fleet.MDMAssetAPNSKey,
			fleet.MDMAssetAPNSCert,
		})
		if err != nil {
			initFatal(err, "loading MDM assets from database")
		}

		var appleBMCerts bool
		appleBMCerts, err = checkMDMAssetsExist(context.Background(), ds, []fleet.MDMAssetName{
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
		logger.InfoContext(cmd.Context(), "Apple MDM enabled")
	}
	if appCfg.MDM.AppleBMEnabledAndConfigured {
		logger.InfoContext(cmd.Context(), "Apple Business enabled")
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

	mailService := initMailService(cmd.Context(), config, appCfg, logger)

	cronSchedules := fleet.NewCronSchedules()

	baseCtx := licensectx.NewContext(context.Background(), license)
	ctx, cancelFunc := context.WithCancel(baseCtx)
	defer cancelFunc()

	// Channel used to trigger graceful shutdown on fatal DB errors (e.g. Aurora failover).
	dbFatalCh := make(chan error, 1)
	common_mysql.SetFatalErrorHandler(func(ctx context.Context, err error) {
		logger.ErrorContext(ctx, "fatal database error detected, initiating graceful shutdown", "err", err)
		select {
		case dbFatalCh <- err:
		default:
		}
	})

	var conditionalAccessMicrosoftProxy *conditional_access_microsoft_proxy.Proxy
	if config.MicrosoftCompliancePartner.IsSet() {
		var err error
		conditionalAccessMicrosoftProxy, err = conditional_access_microsoft_proxy.New(
			config.MicrosoftCompliancePartner.ProxyURI,
			config.MicrosoftCompliancePartner.ProxyAPIKey,
			func() (string, error) {
				appCfg, err := ds.AppConfig(ctx)
				if err != nil {
					return "", fmt.Errorf("failed to load appconfig: %w", err)
				}
				return appCfg.ServerSettings.ServerURL, nil
			},
		)
		if err != nil {
			initFatal(err, "new microsoft compliance proxy")
		}
	}

	eh := errorstore.NewHandler(ctx, redisPool, logger, config.Logging.ErrorRetentionPeriod)
	scepConfigMgr := scep.NewSCEPConfigService(logger, nil)
	digiCertService := digicert.NewService(digicert.WithLogger(logger))
	ctx = ctxerr.NewContext(ctx, eh)

	// Declare svc early so the closure below can capture it.
	var svc fleet.Service
	config.MDM.AndroidAgent.Validate(initFatal)
	config.MDM.ValidateAndroidBatchSize(initFatal)
	androidSvc, err := android_service.NewService(
		ctx,
		logger,
		ds,
		config.License.Key,
		config.Server.PrivateKey,
		ds,
		func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return svc.NewActivity(ctx, user, activity)
		},
		config.MDM.AndroidAgent,
	)
	if err != nil {
		initFatal(err, "initializing android service")
	}

	orgLogoStore := initOrgLogoStore(ctx, config.S3, mds, logger)

	svc, err = service.NewService(
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
		failingPolicySet,
		geoIP,
		redisWrapperDS,
		depStorage,
		mdmStorage,
		mdmPushService,
		cronSchedules,
		wstepCertManager,
		scepConfigMgr,
		digiCertService,
		conditionalAccessMicrosoftProxy,
		redis_key_value.New(redisPool),
		androidSvc,
		orgLogoStore,
	)
	if err != nil {
		initFatal(err, "initializing service")
	}

	var softwareInstallStore fleet.SoftwareInstallerStore
	var bootstrapPackageStore fleet.MDMBootstrapPackageStore
	var softwareTitleIconStore fleet.SoftwareTitleIconStore
	var distributedLock fleet.Lock
	if license.IsPremium() {
		hydrantService := est.NewService(est.WithLogger(logger))
		profileMatcher := apple_mdm.NewProfileMatcher(redisPool)
		if config.S3.SoftwareInstallersBucket != "" {
			if config.S3.BucketsAndPrefixesMatch() {
				logger.WarnContext(ctx,
					"the S3 buckets and prefixes for carves and software installers appear to be identical, this can cause issues")
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
			logger.InfoContext(ctx, "using S3 software installer store", "bucket", config.S3.SoftwareInstallersBucket)

			bstore, err := s3.NewBootstrapPackageStore(config.S3)
			if err != nil {
				initFatal(err, "initializing S3 bootstrap package store")
			}
			bootstrapPackageStore = bstore
			logger.InfoContext(ctx, "using S3 bootstrap package store", "bucket", config.S3.SoftwareInstallersBucket)

			softwareTitleIconStore, err = s3.NewSoftwareTitleIconStore(config.S3)
			if err != nil {
				initFatal(err, "initializing S3 software title icon store")
			}
			logger.InfoContext(ctx, "using S3 software title icon store", "bucket", config.S3.SoftwareInstallersBucket)
		} else {
			installerDir := os.TempDir()
			if dir := os.Getenv("FLEET_SOFTWARE_INSTALLER_STORE_DIR"); dir != "" {
				installerDir = dir
			}
			store, err := filesystem.NewSoftwareInstallerStore(installerDir)
			if err != nil {
				logger.ErrorContext(ctx, "failed to configure local filesystem software installer store", "err", err)
				softwareInstallStore = failing.NewFailingSoftwareInstallerStore()
			} else {
				softwareInstallStore = store
				logger.InfoContext(ctx,
					"using local filesystem software installer store, this is not suitable for production use", "directory",
					installerDir)
			}

			iconDir := os.TempDir()
			if dir := os.Getenv("FLEET_SOFTWARE_TITLE_ICON_STORE_DIR"); dir != "" {
				iconDir = dir
			}
			iconStore, err := filesystem.NewSoftwareTitleIconStore(iconDir)
			if err != nil {
				logger.ErrorContext(ctx, "failed to configure local filesystem software title icon store", "err", err)
				softwareTitleIconStore = failing.NewFailingSoftwareTitleIconStore()
			} else {
				softwareTitleIconStore = iconStore
				logger.WarnContext(ctx,
					"using local filesystem software title icon store, this is not suitable for production use", "directory",
					iconDir)
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
			softwareTitleIconStore,
			distributedLock,
			redis_key_value.New(redisPool),
			scepConfigMgr,
			digiCertService,
			androidSvc,
			hydrantService,
		)
		if err != nil {
			initFatal(err, "initial Fleet Premium service")
		}
	}

	instanceID, err := server.GenerateRandomText(64)
	if err != nil {
		initFatal(errors.New("Error generating random instance identifier"), "")
	}
	logger.InfoContext(ctx, "instance info", "instanceID", instanceID)

	// Bootstrap activity bounded context (needed for cron schedules and HTTP routes)
	activitySvc, activityRoutes := createActivityBoundedContext(svc, ds, dbConns, logger)
	// Inject the activity bounded context into the main service
	svc.SetActivityService(activitySvc)

	// Bootstrap ACME service module
	acmeSigner := &acmeCSRSigner{signer: scepdepot.NewSigner(scepStorage, scepdepot.WithValidityDays(config.MDM.AppleSCEPSignerValidityDays), scepdepot.WithAllowRenewalDays(14))}
	acmeSvc, acmeRoutes := createACMEServiceModule(ds, dbConns, redisPool, logger, acmeSigner)
	// Inject the ACME service module into the main service
	svc.SetACMEService(acmeSvc)

	// Bootstrap chart bounded context
	chartSvc, chartRoutes := createChartBoundedContext(dbConns, svc, logger)

	// Trace sampler runtime control. The poller re-reads trace_sampler_settings every 60s and atomically swaps the sampler's
	// ratios and force_full so support can flip a 100% debug window via PATCH /debug/trace_sampler without restarting any
	// replicas. No-op when OTEL is disabled.
	if traceSampler != nil {
		go tracing.StartSettingsPoller(ctx, traceSampler, ds, logger)
	}

	startCronSchedules(ctx, cronSchedulesDeps{
		instanceID:             instanceID,
		config:                 &config,
		license:                license,
		logger:                 logger,
		cronSchedules:          cronSchedules,
		ds:                     ds,
		svc:                    svc,
		carveStore:             carveStore,
		enrollHostLimiter:      redisWrapperDS,
		liveQueryStore:         liveQueryStore,
		failingPolicySet:       failingPolicySet,
		redisPool:              redisPool,
		commander:              apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService),
		depStorage:             depStorage,
		softwareInstallStore:   softwareInstallStore,
		bootstrapPackageStore:  bootstrapPackageStore,
		softwareTitleIconStore: softwareTitleIconStore,
		androidSvc:             androidSvc,
		activitySvc:            activitySvc,
		acmeSvc:                acmeSvc,
		chartSvc:               chartSvc,
		auditLogger:            auditLogger,
		distributedLock:        distributedLock,
		initFatal:              initFatal,
	})

	// StartCollectors starts a goroutine per collector, using ctx to cancel.
	task.StartCollectors(ctx, logger.With("cron", "async_task"))

	// Flush seen hosts every second
	hostsAsyncCfg := config.Osquery.AsyncConfigForTask(configpkg.AsyncTaskHostLastSeen)
	if !hostsAsyncCfg.Enabled {
		go func() {
			for range time.Tick(time.Duration(rand.Intn(10)+1) * time.Second) {
				if err := task.FlushHostsLastSeen(baseCtx, clock.C.Now()); err != nil {
					logger.InfoContext(ctx, "failed to update host seen times", "err", err)
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

	httpLogger := logger.With("component", "http")

	limiterStore := &redis.ThrottledStore{
		Pool:      redisPool,
		KeyPrefix: "ratelimit::",
	}

	var httpSigVerifier func(http.Handler) http.Handler
	if license.IsPremium() {
		httpSigVerifier, err = httpsig.Middleware(ds, config.Auth.RequireHTTPMessageSignature, logger.With("component", "http-sig-verifier"))
		if err != nil {
			initFatal(err, "initializing HTTP signature verifier")
		}
	}

	// This is off by default for testing and development uses only.
	cspEV := os.Getenv("FLEET_SERVER_ENABLE_CSP")
	serveCSP := cspEV == "1" || cspEV == "true"

	var apiHandler, frontendHandler, endUserEnrollOTAHandler http.Handler
	{
		frontendHandler = service.PrometheusMetricsHandler(
			"get_frontend",
			service.ServeFrontend(config.Server.URLPrefix, config.Server.SandboxEnabled, httpLogger, serveCSP),
		)

		frontendHandler = service.WithMDMEnrollmentMiddleware(svc, httpLogger, frontendHandler)

		var extra []service.ExtraHandlerOption
		if config.MDM.SSORateLimitPerMinute > 0 {
			extra = append(extra, service.WithMdmSsoRateLimit(throttled.PerMin(config.MDM.SSORateLimitPerMinute)))
		}
		if config.Auth.SSORateLimitPerMinute > 0 {
			extra = append(extra, service.WithSsoRateLimit(throttled.PerMin(config.Auth.SSORateLimitPerMinute)))
		}
		extra = append(extra, service.WithHTTPSigVerifier(httpSigVerifier))

		apiHandler = service.MakeHandler(svc, config, httpLogger, limiterStore, redisPool, carveStore,
			[]endpointer.HandlerRoutesFunc{android_service.GetRoutes(svc, androidSvc), activityRoutes, acmeRoutes, chartRoutes}, extra...)

		// SCIM endpoints are served by a prefix-mounted handler (see
		// scim.RegisterSCIM) that gorilla/mux can't introspect, so surface
		// their routes to the validator explicitly.
		if err := apiendpoints.Validate(apiHandler, scim.RegisterValidationRoutes); err != nil {
			panic(fmt.Sprintf("error initializing API endpoints: %v", err))
		}
		apiHandler = service.WithMDMSSOCallbackRedirect(svc, logger, apiHandler)

		if serveCSP {
			// Only injecting this if CSP is turned on since the default security headers add some overhead to each request
			apiHandler = endpointer.BrowserSecurityHeadersHandler(serveCSP, apiHandler)
		}

		setupRequired, err := svc.SetupRequired(baseCtx)
		if err != nil {
			initFatal(err, "fetching setup requirement")
		}
		// WithSetup will check if first time setup is required
		// By performing the same check inside main, we can make server startups
		// more efficient after the first startup.
		if setupRequired {
			// Pass in a closure to run the fleetctl command, so that the service layer
			// doesn't need to import the CLI package.
			// When Primo mode is enabled, skip the starter library.
			var applyStarterLibrary func(ctx context.Context, serverURL, token string) error
			if config.Partnerships.EnablePrimo {
				applyStarterLibrary = func(ctx context.Context, _, _ string) error {
					logger.DebugContext(ctx, "Skipping starter library application in Primo mode")
					return nil
				}
			} else {
				applyStarterLibrary = func(ctx context.Context, serverURL, token string) error {
					return service.ApplyStarterLibrary(ctx, serverURL, token, logger, func(args []string) error {
						_, err := fleetctl.RunApp(args)
						return err
					})
				}
			}
			apiHandler = service.WithSetup(svc, logger, applyStarterLibrary, apiHandler)
			frontendHandler = service.RedirectLoginToSetup(svc, logger, frontendHandler, config.Server.URLPrefix)
		} else {
			frontendHandler = service.RedirectSetupToLogin(svc, logger, frontendHandler, config.Server.URLPrefix)
		}

		endUserEnrollOTAHandler = service.ServeEndUserEnrollOTA(
			svc,
			config.Server.URLPrefix,
			ds,
			logger,
			serveCSP,
		)
	}

	healthCheckers := make(map[string]health.Checker)
	{
		// a list of dependencies which could affect the status of the app if unavailable.
		deps := map[string]any{
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
	launcher := launcher.New(svc, logger, grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_recovery.UnaryServerInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			grpc_recovery.StreamServerInterceptor(),
		),
	), healthCheckers)

	rootMux := http.NewServeMux()
	// Infra paths (liveness, version, metrics) are platform owned and have zero diagnostic value as traces once their metric
	// counterparts exist. Drop them unconditionally, even under ForceFull, since a 100% debug window should not flood SigNoz
	// with probe spans. Register /metrics here even though its handler is mounted conditionally below. Registering a route
	// whose handler isn't installed is a harmless lookup table entry and keeps the policy in one place.
	traceRegistry.Register(http.MethodGet, "/healthz", tracing.TierNever)
	traceRegistry.Register(http.MethodGet, "/version", tracing.TierNever)
	traceRegistry.Register(http.MethodGet, "/metrics", tracing.TierNever)
	rootMux.Handle("/healthz", service.PrometheusMetricsHandler("healthz", otelmw.WrapHandler(health.Handler(httpLogger, healthCheckers), "/healthz", config)))
	rootMux.Handle("/version", service.PrometheusMetricsHandler("version", otelmw.WrapHandler(version.Handler(), "/version", config)))
	rootMux.Handle("/assets/", service.PrometheusMetricsHandler("static_assets", otelmw.WrapHandlerDynamic(service.ServeStaticAssets("/assets/", serveCSP), config)))

	if len(config.Server.PrivateKey) > 0 {
		commander := apple_mdm.NewMDMAppleCommander(mdmStorage, mdmPushService)
		ddmService := service.NewMDMAppleDDMService(ds, logger)
		vppInstaller := svc.(fleet.AppleMDMVPPInstaller)
		mdmCheckinAndCommandService := service.NewMDMAppleCheckinAndCommandService(
			ds,
			commander,
			vppInstaller,
			license.IsPremium(),
			logger,
			redis_key_value.New(redisPool),
			svc.NewActivity,
		)

		mdmCheckinAndCommandService.RegisterResultsHandler("InstalledApplicationList", service.NewInstalledApplicationListResultsHandler(ds, commander, logger, config.Server.VPPVerifyTimeout, config.Server.VPPVerifyRequestDelay, svc.NewActivity))
		mdmCheckinAndCommandService.RegisterResultsHandler(fleet.DeviceLocationCmdName, service.NewDeviceLocationResultsHandler(ds, commander, logger))
		mdmCheckinAndCommandService.RegisterResultsHandler(fleet.SetRecoveryLockCmdName, service.NewSetRecoveryLockResultsHandler(ds, logger, svc.NewActivity))

		hasSCEPChallenge, err := checkMDMAssetsExist(context.Background(), ds, []fleet.MDMAssetName{fleet.MDMAssetSCEPChallenge})
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

				logger.WarnContext(ctx,
					"Your server already has stored a SCEP challenge. Fleet will ignore this value provided via environment variables when this happens.")
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
			appCfg.ServerSettings.ServerURL,
			config,
		); err != nil {
			initFatal(err, "setup mdm apple services")
		}
	}

	if license.IsPremium() {
		// SCEP proxy (for NDES, etc.)
		if err = service.RegisterSCEPProxy(rootMux, ds, logger, nil, &config); err != nil {
			initFatal(err, "setup SCEP proxy")
		}
		if err = scim.RegisterSCIM(rootMux, ds, svc, logger, &config); err != nil {
			initFatal(err, "setup SCIM")
		}
		// Host identify and conditional access SCEP feature only works if a private key has been set up
		if len(config.Server.PrivateKey) > 0 {
			hostIdentitySCEPDepot, err := mds.NewHostIdentitySCEPDepot(logger.With("component", "host-id-scep-depot"), &config)
			if err != nil {
				initFatal(err, "setup host identity SCEP depot")
			}
			if err = hostidentity.RegisterSCEP(rootMux, hostIdentitySCEPDepot, ds, logger, &config); err != nil {
				initFatal(err, "setup host identity SCEP")
			}

			// Conditional Access SCEP
			condAccessSCEPDepot, err := mds.NewConditionalAccessSCEPDepot(logger.With("component", "conditional-access-scep-depot"), &config)
			if err != nil {
				initFatal(err, "setup conditional access SCEP depot")
			}
			if err = condaccess.RegisterSCEP(ctx, rootMux, condAccessSCEPDepot, ds, logger, &config); err != nil {
				initFatal(err, "setup conditional access SCEP")
			}

			// Conditional Access IdP (Okta)
			if err = condaccess.RegisterIdP(rootMux, ds, logger, &config, limiterStore); err != nil {
				initFatal(err, "setup conditional access IdP")
			}
		} else {
			logger.WarnContext(ctx,
				"Host identity and conditional access SCEP is not available because no server private key has been set up.")
		}
	}

	if config.Prometheus.BasicAuth.Username != "" && config.Prometheus.BasicAuth.Password != "" {
		rootMux.Handle("/metrics", basicAuthHandler(
			config.Prometheus.BasicAuth.Username,
			config.Prometheus.BasicAuth.Password,
			service.PrometheusMetricsHandler("metrics", otelmw.WrapHandler(promhttp.Handler(), "/metrics", config)),
		))
	} else {
		if config.Prometheus.BasicAuth.Disable {
			logger.InfoContext(ctx, "metrics endpoint enabled with http basic auth disabled")
			rootMux.Handle("/metrics", service.PrometheusMetricsHandler("metrics", otelmw.WrapHandler(promhttp.Handler(), "/metrics", config)))
		} else {
			logger.InfoContext(ctx, "metrics endpoint disabled (http basic auth credentials not set)")
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
	rootMux.HandleFunc("/api/", apiTimeoutOverrideHandler(apiHandler, config, logger))
	// The `/api/{version}/fleet/scim` base path is used by SCIM handler. In order to route the `details` route to the apiHandler,
	// we have to explicitly handle that path at the root. The Go router takes precedence for a more specific path. The v1/latest are used in the path for it to be more specific.
	// The Fleet API was designed this way for end-user simplicity.
	rootMux.Handle("/api/v1/fleet/scim/details", apiHandler)
	rootMux.Handle("/api/latest/fleet/scim/details", apiHandler)

	rootMux.Handle("/enroll", otelmw.WrapHandler(endUserEnrollOTAHandler, "/enroll", config))
	rootMux.Handle("/", otelmw.WrapHandler(frontendHandler, "/", config))

	debugHandler := &debugMux{
		fleetAuthenticatedHandler: service.MakeDebugHandler(svc, config, logger, eh, ds),
	}
	rootMux.Handle("/debug/", otelmw.WrapHandlerDynamic(debugHandler, config))

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
			logger.ErrorContext(ctx, "failed to parse live query rest period", "err", err)
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
			logger.InfoContext(ctx, "listening", "transport", "http", "address", config.Server.Address)
			errs <- srv.ListenAndServe()
		} else {
			logger.InfoContext(ctx, "listening", "transport", "https", "address", config.Server.Address)
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
		select {
		case <-sig:
		case <-dbFatalCh:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		errs <- func() error {
			cancelFunc()
			cleanupCronStatsOnShutdown(ctx, ds, logger, instanceID)
			launcher.GracefulStop()
			// Flush any pending OTEL data before shutting down
			if tracerProvider != nil {
				if err := tracerProvider.Shutdown(ctx); err != nil {
					logger.ErrorContext(ctx, "failed to shutdown OTEL tracer provider", "err", err)
				}
			}
			if meterProvider != nil {
				if err := meterProvider.Shutdown(ctx); err != nil {
					logger.ErrorContext(ctx, "failed to shutdown OTEL meter provider", "err", err)
				}
			}
			if loggerProvider != nil {
				if err := loggerProvider.Shutdown(ctx); err != nil {
					logger.ErrorContext(ctx, "failed to shutdown OTEL logger provider", "err", err)
				}
			}
			return srv.Shutdown(ctx)
		}()
	}()

	// block on errs signal
	logger.InfoContext(ctx, "terminated", "err", <-errs)
}

// acmeCSRSigner adapts a depot.Signer to the acme.CSRSigner interface.
type acmeCSRSigner struct {
	signer *scepdepot.Signer
}

func (a *acmeCSRSigner) SignCSR(_ context.Context, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	return a.signer.Signx509CSR(csr)
}

func createACMEServiceModule(ds fleet.Datastore, dbConns *common_mysql.DBConnections, redisPool fleet.RedisPool, logger *slog.Logger, csrSigner acme.CSRSigner) (acme_api.Service, endpointer.HandlerRoutesFunc) {
	providers := acmeacl.NewFleetDatastoreAdapter(ds, csrSigner)
	acmeSvc, acmeRoutesFn := acme_bootstrap.New(dbConns, redisPool, providers, logger)
	acmeRoutes := acmeRoutesFn(log.Logged)
	return acmeSvc, acmeRoutes
}

func createChartBoundedContext(dbConns *common_mysql.DBConnections, svc fleet.Service, logger *slog.Logger) (chart_api.Service, endpointer.HandlerRoutesFunc) {
	legacyAuthorizer, err := authz.NewAuthorizer()
	if err != nil {
		initFatal(err, "initializing chart authorizer")
	}
	chartAuthorizer := authz.NewAuthorizerAdapter(legacyAuthorizer)
	chartViewer := chartacl.NewFleetViewerAdapter()
	chartSvc, chartRoutesFn := chart_bootstrap.New(dbConns, chartAuthorizer, chartViewer, logger)
	// Register all chart types here. The registry is used to validate chart types in the API
	// and to iterate over all chart types when generating chart data.
	chartSvc.RegisterDataset(&chart.UptimeDataset{})
	chartSvc.RegisterDataset(&chart.CVEDataset{})
	// Create auth middleware for chart bounded context
	chartAuthMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint {
		return auth.AuthenticatedUser(svc, next)
	}
	chartRoutes := chartRoutesFn(chartAuthMiddleware)
	return chartSvc, chartRoutes
}

// initOrgLogoStore builds the OrgLogoStore implementation appropriate for the deployment:
//   - S3 when a software installers bucket is configured (shared bucket, distinct prefix)
//   - otherwise a database-backed store, so custom org logos work without object storage or a writable filesystem.
func initOrgLogoStore(ctx context.Context, s3Config configpkg.S3Config, ds *mysql.Datastore, logger *slog.Logger) fleet.OrgLogoStore {
	if s3Config.SoftwareInstallersBucket != "" {
		store, err := s3.NewOrgLogoStore(s3Config)
		if err != nil {
			initFatal(err, "initializing S3 org logo store")
		}
		logger.InfoContext(ctx, "using S3 org logo store", "bucket", s3Config.SoftwareInstallersBucket)
		return store
	}
	logger.InfoContext(ctx, "using database org logo store")
	return ds.NewOrgLogoStore()
}

func createActivityBoundedContext(svc fleet.Service, ds fleet.Datastore, dbConns *common_mysql.DBConnections, logger *slog.Logger) (activity_api.Service, endpointer.HandlerRoutesFunc) {
	legacyAuthorizer, err := authz.NewAuthorizer()
	if err != nil {
		initFatal(err, "initializing activity authorizer")
	}
	activityAuthorizer := authz.NewAuthorizerAdapter(legacyAuthorizer)
	activityACLAdapter := activityacl.NewFleetServiceAdapter(svc, ds)
	activitySvc, activityRoutesFn := activity_bootstrap.New(
		dbConns,
		activityAuthorizer,
		activityACLAdapter,
		logger,
	)
	// Makes sure that api_only users are subject to endpoint
	// restrictions on activity routes.
	activityAuthMiddleware := func(next endpoint.Endpoint) endpoint.Endpoint {
		return auth.AuthenticatedUser(svc, auth.APIOnlyEndpointCheck(next))
	}
	activityRoutes := activityRoutesFn(activityAuthMiddleware)
	return activitySvc, activityRoutes
}

func printDatabaseNotInitializedError() {
	fmt.Printf("################################################################################\n"+
		"# ERROR:\n"+
		"#   Your Fleet database is not initialized. Fleet cannot start up.\n"+
		"#\n"+
		"#   Run `%s prepare db` to initialize the database.\n"+
		"################################################################################\n",
		os.Args[0])
}

func printMissingMigrationsWarning(w io.Writer, tables []int64, data []int64) {
	fmt.Fprintf(w, "################################################################################\n"+
		"# WARNING:\n"+
		"#   Your Fleet database is missing required migrations. This is likely to cause\n"+
		"#   errors in Fleet.\n"+
		"#\n"+
		"#   Missing migrations: %s.\n"+
		"#\n"+
		"#   Run `%s prepare db` to perform migrations.\n"+
		"#\n"+
		"#   To run the server without performing migrations:\n"+
		"#     - Set environment variable FLEET_UPGRADES_ALLOW_MISSING_MIGRATIONS=1, or,\n"+
		"#     - Set config updates.allow_missing_migrations to true, or,\n"+
		"#     - Use command line argument --upgrades_allow_missing_migrations=true\n"+
		"################################################################################\n",
		tablesAndDataToString(tables, data), os.Args[0])
}

func printFleetv4732FixNeededMessage() {
	fmt.Printf("################################################################################\n"+
		"# WARNING:\n"+
		"#   Your Fleet database has misnumbered migrations introduced in some released\n"+
		"#   v4.73.2 artifacts. Fleet will automatically perform this fix prior to database\n"+
		"#   migrations. Please back up your data before continuing.\n"+
		"#\n"+
		"#   Run `%s prepare db` to perform migrations.\n"+
		"#\n"+
		"#   To run the server without performing migrations:\n"+
		"#     - Set environment variable FLEET_UPGRADES_ALLOW_MISSING_MIGRATIONS=1, or,\n"+
		"#     - Set config updates.allow_missing_migrations to true, or,\n"+
		"#     - Use command line argument --upgrades_allow_missing_migrations=true\n"+
		"################################################################################\n", os.Args[0])
}

func initLicense(config *configpkg.FleetConfig, devLicense, devExpiredLicense bool) (*fleet.LicenseInfo, error) {
	if devLicense {
		// This license key is valid for development only
		config.License.Key = "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJGbGVldCBEZXZpY2UgTWFuYWdlbWVudCBJbmMuIiwiZXhwIjoxNzk4NTk3MDU1LCJzdWIiOiJkZXZlbG9wbWVudCIsImRldmljZXMiOjEwMDAsIm5vdGUiOiJmb3IgZGV2ZWxvcG1lbnQgb25seSIsInRpZXIiOiJwcmVtaXVtIiwiaWF0IjoxNzgyODI5MDU1fQ.SCwrVBV3fIb7JSS5tOLx0EmlyS6m20h34C9WOW1RqlLf009gEldWk2eO3ma8caW5_te4aEbjcvTBDeIkvM7NIA"
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

	logger *slog.Logger
}

func (in *devSQLInterceptor) ConnQueryContext(ctx context.Context, conn driver.QueryerContext, query string, args []driver.NamedValue) (driver.Rows, error) {
	start := time.Now()
	rows, err := conn.QueryContext(ctx, query, args)
	in.logQuery(ctx, start, query, args, err)
	return rows, err
}

func (in *devSQLInterceptor) StmtQueryContext(ctx context.Context, stmt driver.StmtQueryContext, query string, args []driver.NamedValue) (driver.Rows, error) {
	start := time.Now()
	rows, err := stmt.QueryContext(ctx, args)
	in.logQuery(ctx, start, query, args, err)
	return rows, err
}

func (in *devSQLInterceptor) StmtExecContext(ctx context.Context, stmt driver.StmtExecContext, query string, args []driver.NamedValue) (driver.Result, error) {
	start := time.Now()
	result, err := stmt.ExecContext(ctx, args)
	in.logQuery(ctx, start, query, args, err)
	return result, err
}

var spaceRegex = regexp.MustCompile(`\s+`)

func (in *devSQLInterceptor) logQuery(ctx context.Context, start time.Time, query string, args []driver.NamedValue, err error) {
	query = strings.TrimSpace(spaceRegex.ReplaceAllString(query, " "))
	if err != nil {
		in.logger.ErrorContext(ctx, "sql query", "duration", time.Since(start), "query", query, "args", argsToString(args), "err", err)
	} else {
		in.logger.DebugContext(ctx, "sql query", "duration", time.Since(start), "query", query, "args", argsToString(args))
	}
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

func createTestBuckets(ctx context.Context, config *configpkg.FleetConfig, logger *slog.Logger) {
	softwareInstallerStore, err := s3.NewSoftwareInstallerStore(config.S3)
	if err != nil {
		initFatal(err, "initializing S3 software installer store")
	}
	if err := softwareInstallerStore.CreateTestBucket(ctx, config.S3.SoftwareInstallersBucket); err != nil {
		// Don't panic, allow devs to run Fleet without S3 dependency.
		logger.InfoContext(ctx, "failed to create test software installer bucket",
			"err", err,
			"name", config.S3.SoftwareInstallersBucket,
		)
	}
	carveStore, err := s3.NewCarveStore(config.S3, nil)
	if err != nil {
		initFatal(err, "initializing S3 carve store")
	}
	if err := carveStore.CreateTestBucket(ctx, config.S3.CarvesBucket); err != nil {
		// Don't panic, allow devs to run Fleet without S3 dependency.
		logger.InfoContext(ctx, "failed to create test carve bucket",
			"err", err,
			"name", config.S3.CarvesBucket,
		)
	}
}
