package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	testUsername = "fleet"
	testPassword = "insecure"
	testAddress  = "localhost:3306"
	testDatabase = "fleet"
)

func main() {
	flagKey := flag.String("key", "", "Key used to encrypt the assets")
	flagDir := flag.String("dir", "", "Directory to put the exported assets")
	flagDBUser := flag.String("db-user", testUsername, "Username used to connect to the MySQL instance")
	flagDBPass := flag.String("db-password", testPassword, "Password used to connect to the MySQL instance")
	flagDBAddress := flag.String("db-address", testAddress, "Address used to connect to the MySQL instance")
	flagDBName := flag.String("db-name", testDatabase, "Name of the database with the asset information in the MySQL instance")
	flag.Parse()

	if *flagKey == "" {
		log.Fatal("-key flag is required")
	}

	if *flagDir != "" {
		if err := os.MkdirAll(*flagDir, os.ModePerm); err != nil {
			log.Fatal("ensuring directory: ", err)
		}
	}

	db, err := sql.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s)/?multiStatements=true", testUsername, testPassword, testAddress),
	)
	if err != nil {
		log.Fatal("opening MySQL connection:", err)
	}
	defer db.Close()

	mysqlCfg := config.MysqlConfig{
		Username: *flagDBUser,
		Password: *flagDBPass,
		Address:  *flagDBAddress,
		Database: *flagDBName,
	}
	ds, err := mysql.New(
		mysqlCfg,
		clock.NewMockClock(),
		mysql.LimitAttempts(1),
		mysql.WithFleetConfig(&config.FleetConfig{
			Server: config.ServerConfig{
				PrivateKey: *flagKey,
			},
		}),
	)
	if err != nil {
		log.Fatal("creating datastore instance:", err)
	}
	defer ds.Close()

	ctx := context.Background()
	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
		fleet.MDMAssetCAKey,
		fleet.MDMAssetAPNSKey,
		fleet.MDMAssetAPNSCert,
		fleet.MDMAssetABMCert,
		fleet.MDMAssetABMKey,
		fleet.MDMAssetABMToken,
		fleet.MDMAssetSCEPChallenge,
		fleet.MDMAssetVPPToken,
	})
	if err != nil && !errors.Is(err, mysql.ErrPartialResult) {
		log.Fatal("retrieving assets from db:", err)
	}

	for _, asset := range assets {
		path := filepath.Join(*flagDir, string(asset.Name))
		switch {
		case strings.Contains(path, "_key"):
			path = path + ".key"
		case strings.Contains(path, "_cert"):
			path = path + ".crt"
		}
		if err := os.WriteFile(path, asset.Value, 0600); err != nil {
			log.Fatal("writing asset:", err)
		}

		log.Printf("wrote %s in %s", asset.Name, path)
	}
}
