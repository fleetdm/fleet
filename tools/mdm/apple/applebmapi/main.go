// Command applebmapi takes an Apple Business server token in decrypted
// JSON format and calls the Apple BM API to retrieve and print the account
// information or the specified enrollment profile.
//
// Was implemented to test out https://github.com/fleetdm/fleet/issues/7515#issuecomment-1330889768,
// and can still be useful for debugging purposes.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
)

func main() {
	mysqlAddr := flag.String("mysql", "localhost:3306", "mysql address")
	serverPrivateKey := flag.String("key", "", "fleet server's private key (to decrypt MDM assets)")
	profileUUID := flag.String("profile-uuid", "", "the Apple profile UUID to retrieve")
	serialNum := flag.String("serial-number", "", "serial number of a device to get the device details")
	command := flag.String("command", "", "the supported command to execute, if not providing profile-uuid or serial-number.")
	orgName := flag.String("org-name", "", "organization name of the token")

	flag.Parse()

	if *serverPrivateKey == "" {
		log.Fatal("must provide -key")
	}
	if *orgName == "" {
		log.Fatal("must provide -org-name")
	}
	if (*profileUUID != "" && *serialNum != "") ||
		(*profileUUID != "" && *command != "") ||
		(*serialNum != "" && *command != "") {
		log.Fatal("only one of -profile-uuid, -serial-number, or -command must be provided")
	}

	if len(*serverPrivateKey) > 32 {
		// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
		// infra setups generate keys that are longer than 32 bytes.
		truncatedServerPrivateKey := (*serverPrivateKey)[:32]
		serverPrivateKey = &truncatedServerPrivateKey
	}

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
			Server: config.ServerConfig{
				PrivateKey: *serverPrivateKey,
			},
		}),
	}
	mds, err := mysql.New(cfg, clock.C, opts...)
	if err != nil {
		log.Fatal(err)
	}

	depStorage, err := mds.NewMDMAppleDEPStorage()
	if err != nil {
		log.Fatal(err)
	}
	depClient := godep.NewClient(depStorage, fleethttp.NewClient())

	ctx := context.Background()
	var res any
	switch {
	case *profileUUID != "":
		res, err = depClient.GetProfile(ctx, *orgName, *profileUUID)
	case *serialNum != "":
		res, err = depClient.GetDeviceDetails(ctx, *orgName, *serialNum)
	case *command == "adue":
		res, err = depClient.FetchAccountDrivenEnrollmentServiceDiscovery(ctx, *orgName)
	default:
		res, err = depClient.AccountDetail(ctx, *orgName)
	}
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		log.Fatalf("pretty-format body: %v", err)
	}
	fmt.Printf("body: \n%s\n", string(b))
}
