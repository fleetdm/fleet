package main

import (
	"flag"
	"fmt"
	stdlog "log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	dephttp "github.com/fleetdm/fleet/v4/server/mdm/nanodep/http"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/http/api"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log/stdlogfmt"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/parse"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/proxy"
)

// overridden by -ldflags -X
var version = "unknown"

const (
	apiUsername = "depserver"

	endpointVersion  = "/version"
	endpointTokens   = "/v1/tokens/" //nolint:gosec
	endpointConfig   = "/v1/config/"
	endpointTokenPKI = "/v1/tokenpki/" //nolint:gosec
	endpointAssigner = "/v1/assigner/"
	endpointProxy    = "/proxy/"
)

func main() {
	var (
		flDebug   = flag.Bool("debug", false, "log debug messages")
		flListen  = flag.String("listen", ":9001", "HTTP listen address")
		flAPIKey  = flag.String("api", "", "API key for API endpoints")
		flVersion = flag.Bool("version", false, "print version")
		flStorage = flag.String("storage", "file", "storage backend")
		flDSN     = flag.String("storage-dsn", "", "storage data source name")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		return
	}

	if *flAPIKey == "" {
		fmt.Fprintf(flag.CommandLine.Output(), "empty API key\n")
		flag.Usage()
		os.Exit(1)
	}

	logger := stdlogfmt.New(stdlog.Default(), *flDebug)

	storage, err := parse.Storage(*flStorage, *flDSN)
	if err != nil {
		logger.Info("msg", "creating storage backend", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.Handle(endpointVersion, dephttp.VersionHandler(version))

	handleStrippedAPI := func(handler http.Handler, endpoint string) {
		handler = http.StripPrefix(endpoint, handler)
		handler = dephttp.BasicAuthMiddleware(handler, apiUsername, *flAPIKey, "depserver")
		mux.Handle(endpoint, handler)
	}

	tokensMux := dephttp.NewMethodMux()
	tokensMux.Handle("PUT", api.StoreAuthTokensHandler(storage, logger.With("handler", "store-auth-tokens")))
	tokensMux.Handle("GET", api.RetrieveAuthTokensHandler(storage, logger.With("handler", "retrieve-auth-tokens")))
	handleStrippedAPI(tokensMux, endpointTokens)

	configMux := dephttp.NewMethodMux()
	configMux.Handle("GET", api.RetrieveConfigHandler(storage, logger.With("handler", "retrieve-config")))
	configMux.Handle("PUT", api.StoreConfigHandler(storage, logger.With("handler", "store-config")))
	handleStrippedAPI(configMux, endpointConfig)

	tokenPKIMux := dephttp.NewMethodMux()
	tokenPKIMux.Handle("GET", api.GetCertTokenPKIHandler(storage, logger.With("handler", "get-token-pki")))
	tokenPKIMux.Handle("PUT", api.DecryptTokenPKIHandler(storage, storage, logger.With("handler", "put-token-pki")))
	handleStrippedAPI(tokenPKIMux, endpointTokenPKI)

	assignerMux := dephttp.NewMethodMux()
	assignerMux.Handle("GET", api.RetrieveAssignerProfileHandler(storage, logger.With("handler", "retrieve-assigner-profile")))
	assignerMux.Handle("PUT", api.StoreAssignerProfileHandler(storage, logger.With("handler", "store-assigner-profile")))
	handleStrippedAPI(assignerMux, endpointAssigner)

	p := proxy.New(
		client.NewTransport(http.DefaultTransport, http.DefaultClient, storage, nil),
		storage,
		logger.With("component", "proxy"),
	)
	var proxyHandler http.Handler = proxy.ProxyDEPNameHandler(p, logger.With("handler", "proxy"))
	proxyHandler = http.StripPrefix(endpointProxy, proxyHandler)
	proxyHandler = DelHeaderMiddleware(proxyHandler, "Authorization")
	proxyHandler = dephttp.BasicAuthMiddleware(proxyHandler, apiUsername, *flAPIKey, "depserver")
	mux.Handle(endpointProxy, proxyHandler)

	// init for newTraceID()
	rand.Seed(time.Now().UnixNano())

	logger.Info("msg", "starting server", "listen", *flListen)
	err = http.ListenAndServe(*flListen, dephttp.TraceLoggingMiddleware(mux, logger.With("handler", "log"), newTraceID)) //nolint:gosec
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
func newTraceID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:gosec
	return fmt.Sprintf("%x", b)
}

// DelHeaderMiddleware deletes header from the HTTP request headers before calling h.
func DelHeaderMiddleware(h http.Handler, header string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del(header)
		h.ServeHTTP(w, r)
	}
}
