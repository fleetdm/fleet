package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	kitlog "github.com/go-kit/kit/log"
	"github.com/micromdm/nanodep/client"
	nanodep_client "github.com/micromdm/nanodep/client"
)

func main() {
	mysqlAddr := flag.String("mysql", "localhost:3306", "mysql address")
	appleBMToken := flag.String("apple-bm-token", "", "path to (decrypted) Apple BM token")

	flag.Parse()

	if *appleBMToken == "" {
		log.Fatal("must provide Apple BM token")
	}
	tok, err := ioutil.ReadFile(*appleBMToken)
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

	httpClient := fleethttp.NewClient()
	depTransport := client.NewTransport(httpClient.Transport, httpClient, depStorage, nil)
	depClient := client.NewClient(fleethttp.NewClient(), depTransport)

	ctx := context.Background()
	req, err := client.NewRequestWithContext(ctx, apple_mdm.DEPName, depStorage, "GET", "/account", nil)
	if err != nil {
		log.Fatalf("new request: %v", err)
	}
	res, err := depClient.Do(req)
	if err != nil {
		log.Fatalf("execute request: %v", err)
	}
	defer res.Body.Close()
	fmt.Printf("status: %d\n", res.StatusCode)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("read response body: %v", err)
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, body, "", "  "); err != nil {
		log.Fatalf("pretty-format body: %v", err)
	}
	fmt.Printf("body: \n%s\n", buf.String())
}
