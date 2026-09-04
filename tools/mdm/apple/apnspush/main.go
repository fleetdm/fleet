// Command apnspush takes a mysql database connection information and the fleet
// server private key (to decrypt MDM assets) and sends a push notification to
// a host identified by UUID (the host doesn't have to exist in Fleet, but for
// the notification to do anything it should have been enrolled in Fleet MDM,
// even if the host itself is now deleted from Fleet).
//
// Was implemented to force deleted iDevices to check-in sooner for
// https://github.com/fleetdm/fleet/issues/22941
// and can still be useful for debugging purposes.
//
// Usage (through the full Fleet push stack — nanomdm push service + nanopush):
//
//	go run ./tools/mdm/apple/apnspush/main.go -mysql localhost:3306 -server-private-key <key> HOST_UUID1 HOST_UUID2 ...
//
// Direct mode: resolves the device token and push magic from the DB, then
// sends a raw request to APNS itself — no nanomdm push provider — and dumps
// Apple's raw response (status, headers, body) so the actual APNS behavior
// can be inspected:
//
//	go run ./tools/mdm/apple/apnspush/main.go -direct -mysql localhost:3306 -server-private-key <key> HOST_UUID1 ...
//	... -direct -url https://api.development.push.apple.com ...  # APNS sandbox
//	... -direct -url http://localhost:8378 ...                   # mock APNS server (cmd/apple-apns-mock)
//	... -direct -expiration 0 ...                                # send apns-expiration: 0 (deliver-now-or-discard)
//	... -direct -fake ANY_ID ...                                 # push a UUID with no nano_enrollments row, deriving
//	                                                             # the mdmtest token/magic scheme from the argument
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/nanopush"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/service"
)

func main() {
	mysqlAddr := flag.String("mysql", "localhost:3306", "mysql address")
	serverPrivateKey := flag.String("server-private-key", "", "fleet server's private key (to decrypt MDM assets)")
	direct := flag.Bool("direct", false, "send raw requests directly to APNS, bypassing the nanomdm push service and provider, and dump the raw responses")
	apnsURL := flag.String("url", "https://api.push.apple.com", "APNS base URL for -direct mode (e.g. https://api.development.push.apple.com for the sandbox, or a mock server)")
	expiration := flag.Int64("expiration", -1, "apns-expiration header value (unix seconds) for -direct mode; -1 omits the header, 0 means deliver-now-or-discard")
	fake := flag.Bool("fake", false, "for -direct mode: UUIDs not found in nano_enrollments are pushed anyway, deriving the mdmtest scheme (token=hex(\"token\"+UUID), magic=\"pushmagic\"+UUID) instead of being skipped")
	pushType := flag.String("type", "", "apns-push-type header value for -direct mode; if empty, the header is omitted.")

	flag.Parse()
	hostUUIDs := flag.Args()

	if *serverPrivateKey == "" {
		log.Fatal("must provide -server-private-key")
	}
	if len(hostUUIDs) == 0 {
		log.Fatal("must provide at least one target host uuid")
	}

	if len(*serverPrivateKey) > 32 {
		// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
		// infra setups generate keys that are longer than 32 bytes.
		truncatedServerPrivateKey := (*serverPrivateKey)[:32]
		serverPrivateKey = &truncatedServerPrivateKey
	}

	// this matches the development config in /cmd/fleet/main.go
	cfg := config.MysqlConfig{
		Protocol:        "tcp",
		Address:         *mysqlAddr,
		Database:        "fleet",
		Username:        "fleet",
		Password:        "insecure",
		MaxOpenConns:    50,
		MaxIdleConns:    50,
		ConnMaxLifetime: 0,
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	opts := []mysql.DBOption{
		mysql.Logger(logger),
		mysql.WithFleetConfig(&config.FleetConfig{
			Server: config.ServerConfig{PrivateKey: *serverPrivateKey},
		}),
	}
	mds, err := mysql.New(cfg, clock.C, opts...)
	if err != nil {
		log.Fatal(err)
	}

	mdmStorage, err := mds.NewMDMAppleMDMStorage()
	if err != nil {
		log.Fatalf("initialize mdm apple MySQL storage: %v", err)
	}

	if *direct {
		if err := pushDirect(context.Background(), mdmStorage, *apnsURL, *expiration, *fake, *pushType, hostUUIDs); err != nil {
			log.Fatal(err)
		}
		return
	}

	pushProviderFactory := nanopush.NewFactory(
		nanopush.WithNewClient(func(cert *tls.Certificate) (*http.Client, error) {
			return fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
				Certificates: []tls.Certificate{*cert},
				MinVersion:   tls.VersionTLS12, // Apple APNs requires TLS 1.2+
			})), nil
		}),
		// same default expiration the Fleet server uses (mdm.apple_apns_push_expiration)
		nanopush.WithExpiration(30*24*time.Hour),
	)

	nanoMDMLogger := service.NewNanoMDMLogger(logger.With("component", "apple-mdm-push"))
	pusher := nanomdm_pushsvc.New(mdmStorage, mdmStorage, pushProviderFactory, nanoMDMLogger)
	res, err := pusher.Push(context.Background(), hostUUIDs)
	if err != nil {
		log.Fatalf("send push notification: %v", err)
	}

	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		log.Fatalf("json-marshal response: %v", err)
	}
	log.Printf("response: %s", string(b))
}

// pushDirect resolves each host UUID to its device token and push magic via
// nano_enrollments, then sends the same request Fleet sends today —
// POST /3/device/<token> with body {"mdm":"<magic>"} — with the APNS
// certificate from mdm_config_assets as the TLS client certificate, and
// prints the raw response.
func pushDirect(ctx context.Context, mdmStorage *mysql.NanoMDMStorage, baseURL string, expiration int64, fake bool, pushType string, hostUUIDs []string) error {
	cert, _, err := mdmStorage.RetrievePushCert(ctx, "")
	if err != nil {
		return fmt.Errorf("retrieve APNS certificate from DB: %w", err)
	}

	pushInfos, err := mdmStorage.RetrievePushInfo(ctx, hostUUIDs)
	if err != nil {
		return fmt.Errorf("retrieve push info from DB: %w", err)
	}

	// Same client the full-stack path above builds, so -direct exercises the
	// transport Fleet actually uses. HTTP/2 comes from ALPN (fleethttp's
	// transport inherits ForceAttemptHTTP2 from http.DefaultTransport); the
	// response's Proto is printed below, so a downgrade is visible.
	client := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		Certificates: []tls.Certificate{*cert},
		MinVersion:   tls.VersionTLS12, // Apple APNs requires TLS 1.2+
	}))

	var failed int
	for _, uuid := range hostUUIDs {
		pushInfo := pushInfos[uuid]
		var provenance string
		switch {
		case pushInfo != nil:
			provenance = "nano_enrollments"
		case fake:
			// Same deterministic scheme mdmtest clients use in TokenUpdate
			// (pkg/mdm/mdmtest/apple.go), so fake pushes line up with what
			// simulated devices derive for themselves.
			pushInfo = &mdm.Push{
				PushMagic: "pushmagic" + uuid,
				Token:     []byte("token" + uuid),
				Topic:     "com.apple.mgmt.External." + uuid,
			}
			provenance = "not in nano_enrollments — derived fake (mdmtest scheme)"
		default:
			fmt.Printf("%s: no push info in nano_enrollments (never enrolled, or enrollment deleted); use -fake to push a derived token anyway\n", uuid)
			failed++
			continue
		}
		token := pushInfo.Token.String()
		fmt.Printf("%s: (%s)\n  topic: %s\n  token: %s\n  magic: %s\n", uuid, provenance, pushInfo.Topic, token, pushInfo.PushMagic)

		body, err := json.Marshal(map[string]string{"mdm": pushInfo.PushMagic})
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/3/device/"+token, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if expiration >= 0 {
			req.Header.Set("apns-expiration", strconv.FormatInt(expiration, 10))
		}
		if pushType != "" {
			req.Header.Set("apns-push-type", pushType)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("  request failed: %v\n", err)
			failed++
			continue
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		resp.Body.Close()

		fmt.Printf("  response: %s %s\n", resp.Proto, resp.Status)
		keys := make([]string, 0, len(resp.Header))
		for k := range resp.Header {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("    %s: %s\n", k, strings.Join(resp.Header.Values(k), ", "))
		}
		if len(respBody) > 0 {
			fmt.Printf("    body: %s\n", respBody)
		}
		if resp.StatusCode != http.StatusOK {
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d pushes failed", failed, len(hostUUIDs))
	}
	return nil
}
