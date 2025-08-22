//nolint:gocritic // Test tool, not production code
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

var (
	endpointFlag   = flag.String("endpoint", "", "RDS endpoint address (without port)")
	portFlag       = flag.String("port", "3306", "Database port")
	userFlag       = flag.String("user", "fleet_iam_user", "Username for IAM authentication")
	dbNameFlag     = flag.String("db", "fleet", "Database name")
	assumeRoleFlag = flag.String("assume-role", "", "STS assume role ARN (optional)")
	externalIDFlag = flag.String("external-id", "", "STS external ID (optional)")
)

func main() {
	flag.Parse()

	if *endpointFlag == "" {
		log.Fatal("RDS endpoint is required (-endpoint flag)")
	}
	if *userFlag == "" {
		log.Fatal("Username is required (-user flag)")
	}

	logger := level.NewFilter(kitlog.NewLogfmtLogger(os.Stderr), level.AllowDebug())

	// Configure MySQL connection with IAM auth
	mysqlConfig := &config.MysqlConfig{
		Protocol:         "tcp",
		Address:          fmt.Sprintf("%s:%s", *endpointFlag, *portFlag),
		Username:         *userFlag,
		Database:         *dbNameFlag,
		StsAssumeRoleArn: *assumeRoleFlag,
		StsExternalID:    *externalIDFlag,
	}

	dbOpts := &common_mysql.DBOptions{
		MaxAttempts: 3,
		Logger:      logger,
	}

	log.Printf("Connecting to RDS at %s:%s with IAM auth for user %s", *endpointFlag, *portFlag, *userFlag)
	if *assumeRoleFlag != "" {
		log.Printf("Using assume role: %s", *assumeRoleFlag)
	}

	log.Println("📋 Testing connection with IAM token...")
	db, err := common_mysql.NewDB(mysqlConfig, dbOpts, "")
	if err != nil {
		log.Printf("❌ Connection failed: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := testConnection(db); err != nil {
		log.Printf("❌ Test failed: %v", err)
		os.Exit(1)
	}

	log.Println("✅ Connection successful!")
}

func testConnection(db *sqlx.DB) error {
	ctx := context.Background()

	// Execute test query
	var version string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		return fmt.Errorf("failed to query version: %w", err)
	}
	log.Printf("  Database version: %s", version)

	return nil
}
