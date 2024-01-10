package main

import (
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/certverify"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cli"
	mdmhttp "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http"
	httpapi "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http/api"
	httpmdm "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/http/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log/stdlogfmt"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/buford"
	pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/certauth"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/dump"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/microwebhook"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/multi"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/service/nanomdm"
)

// overridden by -ldflags -X
var version = "unknown"

const (
	endpointMDM     = "/mdm"
	endpointCheckin = "/checkin"

	endpointAPIPushCert  = "/v1/pushcert"
	endpointAPIPush      = "/v1/push/"
	endpointAPIEnqueue   = "/v1/enqueue/"
	endpointAPIMigration = "/migration"
	endpointAPIVersion   = "/version"
)

func main() {
	cliStorage := cli.NewStorage()
	flag.Var(&cliStorage.Storage, "storage", "name of storage backend")
	flag.Var(&cliStorage.DSN, "storage-dsn", "data source name (e.g. connection string or path)")
	flag.Var(&cliStorage.DSN, "dsn", "data source name; deprecated: use -storage-dsn")
	flag.Var(&cliStorage.Options, "storage-options", "storage backend options")
	var (
		flListen     = flag.String("listen", ":9000", "HTTP listen address")
		flAPIKey     = flag.String("api", "", "API key for API endpoints")
		flVersion    = flag.Bool("version", false, "print version")
		flRootsPath  = flag.String("ca", "", "path to CA cert for verification")
		flWebhook    = flag.String("webhook-url", "", "URL to send requests to")
		flCertHeader = flag.String("cert-header", "", "HTTP header containing URL-escaped TLS client certificate")
		flDebug      = flag.Bool("debug", false, "log debug messages")
		flDump       = flag.Bool("dump", false, "dump MDM requests and responses to stdout")
		flDisableMDM = flag.Bool("disable-mdm", false, "disable MDM HTTP endpoint")
		flCheckin    = flag.Bool("checkin", false, "enable separate HTTP endpoint for MDM check-ins")
		flMigration  = flag.Bool("migration", false, "HTTP endpoint for enrollment migrations")
		flRetro      = flag.Bool("retro", false, "Allow retroactive certificate-authorization association")
		flDMURLPfx   = flag.String("dm", "", "URL to send Declarative Management requests to")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		return
	}

	if *flDisableMDM && *flAPIKey == "" {
		stdlog.Fatal("nothing for server to do")
	}

	logger := stdlogfmt.New(stdlogfmt.WithDebugFlag(*flDebug))

	if *flRootsPath == "" {
		stdlog.Fatal("must supply CA cert path flag")
	}
	caPEM, err := ioutil.ReadFile(*flRootsPath)
	if err != nil {
		stdlog.Fatal(err)
	}
	verifier, err := certverify.NewPoolVerifier(caPEM, x509.ExtKeyUsageClientAuth)
	if err != nil {
		stdlog.Fatal(err)
	}

	mdmStorage, err := cliStorage.Parse(logger)
	if err != nil {
		stdlog.Fatal(err)
	}

	// create 'core' MDM service
	nanoOpts := []nanomdm.Option{nanomdm.WithLogger(logger.With("service", "nanomdm"))}
	if *flDMURLPfx != "" {
		logger.Debug("msg", "declarative management setup", "url", *flDMURLPfx)
		dm, err := nanomdm.NewDeclarativeManagementHTTPCaller(*flDMURLPfx, http.DefaultClient)
		if err != nil {
			stdlog.Fatal(err)
		}
		nanoOpts = append(nanoOpts, nanomdm.WithDeclarativeManagement(dm))
	}
	nano := nanomdm.New(mdmStorage, nanoOpts...)

	mux := http.NewServeMux()

	if !*flDisableMDM {
		var mdmService service.CheckinAndCommandService = nano
		if *flWebhook != "" {
			webhookService := microwebhook.New(*flWebhook, mdmStorage)
			mdmService = multi.New(logger.With("service", "multi"), mdmService, webhookService)
		}
		certAuthOpts := []certauth.Option{certauth.WithLogger(logger.With("service", "certauth"))}
		if *flRetro {
			certAuthOpts = append(certAuthOpts, certauth.WithAllowRetroactive())
		}
		mdmService = certauth.New(mdmService, mdmStorage, certAuthOpts...)
		if *flDump {
			mdmService = dump.New(mdmService, os.Stdout)
		}

		// register 'core' MDM HTTP handler
		var mdmHandler http.Handler
		if *flCheckin {
			// if we use the check-in handler then only handle commands
			mdmHandler = httpmdm.CommandAndReportResultsHandler(mdmService, logger.With("handler", "command"))
		} else {
			// if we don't use a check-in handler then do both
			mdmHandler = httpmdm.CheckinAndCommandHandler(mdmService, logger.With("handler", "checkin-command"))
		}
		mdmHandler = httpmdm.CertVerifyMiddleware(mdmHandler, verifier, logger.With("handler", "cert-verify"))
		if *flCertHeader != "" {
			mdmHandler = httpmdm.CertExtractPEMHeaderMiddleware(mdmHandler, *flCertHeader, logger.With("handler", "cert-extract"))
		} else {
			mdmHandler = httpmdm.CertExtractMdmSignatureMiddleware(mdmHandler, logger.With("handler", "cert-extract"))
		}
		mux.Handle(endpointMDM, mdmHandler)

		if *flCheckin {
			// if we specified a separate check-in handler, set it up
			var checkinHandler http.Handler
			checkinHandler = httpmdm.CheckinHandler(mdmService, logger.With("handler", "checkin"))
			checkinHandler = httpmdm.CertVerifyMiddleware(checkinHandler, verifier, logger.With("handler", "cert-verify"))
			if *flCertHeader != "" {
				checkinHandler = httpmdm.CertExtractPEMHeaderMiddleware(checkinHandler, *flCertHeader, logger.With("handler", "cert-extract"))
			} else {
				checkinHandler = httpmdm.CertExtractMdmSignatureMiddleware(checkinHandler, logger.With("handler", "cert-extract"))
			}
			mux.Handle(endpointCheckin, checkinHandler)
		}
	}

	if *flAPIKey != "" {
		const apiUsername = "nanomdm"

		// create our push provider and push service
		pushProviderFactory := buford.NewPushProviderFactory()
		pushService := pushsvc.New(mdmStorage, mdmStorage, pushProviderFactory, logger.With("service", "push"))

		// register API handler for push cert storage/upload.
		var pushCertHandler http.Handler
		pushCertHandler = httpapi.StorePushCertHandler(mdmStorage, logger.With("handler", "store-cert"))
		pushCertHandler = mdmhttp.BasicAuthMiddleware(pushCertHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle(endpointAPIPushCert, pushCertHandler)

		// register API handler for push notifications.
		// we strip the prefix to use the path as an id.
		var pushHandler http.Handler
		pushHandler = httpapi.PushHandler(pushService, logger.With("handler", "push"))
		pushHandler = http.StripPrefix(endpointAPIPush, pushHandler)
		pushHandler = mdmhttp.BasicAuthMiddleware(pushHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle(endpointAPIPush, pushHandler)

		// register API handler for new command queueing.
		// we strip the prefix to use the path as an id.
		var enqueueHandler http.Handler
		enqueueHandler = httpapi.RawCommandEnqueueHandler(mdmStorage, pushService, logger.With("handler", "enqueue"))
		enqueueHandler = http.StripPrefix(endpointAPIEnqueue, enqueueHandler)
		enqueueHandler = mdmhttp.BasicAuthMiddleware(enqueueHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle(endpointAPIEnqueue, enqueueHandler)

		if *flMigration {
			// setup a "migration" handler that takes Check-In messages
			// without bothering with certificate auth or other
			// middleware.
			//
			// if the source MDM can put together enough of an
			// authenticate and tokenupdate message to effectively
			// generate "enrollments" then this effively allows us to
			// migrate MDM enrollments between servers.
			var migHandler http.Handler
			migHandler = httpmdm.CheckinHandler(nano, logger.With("handler", "migration"))
			migHandler = mdmhttp.BasicAuthMiddleware(migHandler, apiUsername, *flAPIKey, "nanomdm")
			mux.Handle(endpointAPIMigration, migHandler)
		}
	}

	mux.HandleFunc(endpointAPIVersion, mdmhttp.VersionHandler(version))

	rand.Seed(time.Now().UnixNano())

	logger.Info("msg", "starting server", "listen", *flListen)
	err = http.ListenAndServe(*flListen, mdmhttp.TraceLoggingMiddleware(mux, logger.With("handler", "log"), newTraceID)) //nolint:gosec
	logs := []interface{}{"msg", "server shutdown"}
	if err != nil {
		logs = append(logs, "err", err)
	}
	logger.Info(logs...)
}

// newTraceID generates a new HTTP trace ID for context logging.
// Currently this just makes a random string. This would be better
// served by e.g. https://github.com/oklog/ulid or something like
// https://opentelemetry.io/ someday.
func newTraceID(_ *http.Request) string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:gosec
	return fmt.Sprintf("%x", b)
}
