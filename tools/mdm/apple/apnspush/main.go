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
// Usage:
// $ go run ./tools/mdm/apple/apnspush/main.go -mysql localhost:3306 -server-private-key <key> HOST_UUID1 HOST_UUID2 ...
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/buford"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/service"
	kitlog "github.com/go-kit/log"
)

func main() {
	mysqlAddr := flag.String("mysql", "localhost:3306", "mysql address")
	serverPrivateKey := flag.String("server-private-key", "", "fleet server's private key (to decrypt MDM assets)")

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
	logger := kitlog.NewLogfmtLogger(os.Stderr)

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

	pushProviderFactory := buford.NewPushProviderFactory(buford.WithNewClient(func(cert *tls.Certificate) (*http.Client, error) {
		return fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{ // nolint:gosec // complains about TLS min version too low
			Certificates: []tls.Certificate{*cert},
		})), nil
	}))

	nanoMDMLogger := service.NewNanoMDMLogger(kitlog.With(logger, "component", "apple-mdm-push"))
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
