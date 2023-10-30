// Command getaccount takes an Apple Business Manager server token in decrypted
// JSON format and calls the Apple BM API to retrieve and print the account
// information.
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
	"os"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	kitlog "github.com/go-kit/kit/log"
	nanodep_client "github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/godep"
)

func main() {
	mysqlAddr := flag.String("mysql", "localhost:3306", "mysql address")
	appleBMToken := flag.String("apple-bm-token", "", "path to (decrypted) Apple BM token")

	flag.Parse()

	if *appleBMToken == "" {
		log.Fatal("must provide Apple BM token")
	}
	tok, err := os.ReadFile(*appleBMToken)
	if err != nil {
		log.Fatal(err)
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
	logger := kitlog.NewLogfmtLogger(os.Stderr)
	opts := []mysql.DBOption{mysql.Logger(logger)}
	mds, err := mysql.New(cfg, clock.C, opts...)
	if err != nil {
		log.Fatal(err)
	}

	var jsonTok nanodep_client.OAuth1Tokens
	if err := json.Unmarshal(tok, &jsonTok); err != nil {
		log.Fatal(err)
	}

	depStorage, err := mds.NewMDMAppleDEPStorage(jsonTok)
	if err != nil {
		log.Fatal(err)
	}

	depClient := godep.NewClient(depStorage, fleethttp.NewClient())

	ctx := context.Background()
	res, err := depClient.AccountDetail(ctx, apple_mdm.DEPName)
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		log.Fatalf("pretty-format body: %v", err)
	}
	fmt.Printf("body: \n%s\n", string(b))
}
