//nolint:gocritic // Test tool, not production code
package main

import (
	"flag"
	"log"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

var (
	addrFlag       = flag.String("addr", "", "ElastiCache endpoint address, including port")
	userFlag       = flag.String("user", "", "Username for authentication")
	passwordFlag   = flag.String("pass", "", "Password for authentication")
	useTLS         = flag.Bool("tls", false, "Whether or not to use TLS")
	assumeRoleFlag = flag.String("assume-role", "", "STS assume role ARN (optional)")
	externalIDFlag = flag.String("external-id", "", "STS external ID (optional)")
	regionFlag     = flag.String("region", "", "AWS region")
	cacheNameFlag  = flag.String("cache-name", "", "ElastiCache cluster name")
)

func main() {
	flag.Parse()

	if *addrFlag == "" {
		log.Fatal("ElastiCache address is required (-addr flag)")
	}

	log.Printf("Connecting to ElastiCache at %s with IAM auth for user %s", *addrFlag, *userFlag)
	if *assumeRoleFlag != "" {
		log.Printf("Using assume role: %s", *assumeRoleFlag)
	}

	config := redis.PoolConfig{
		Server: *addrFlag,
		// UseTLS:           true,
		StsAssumeRoleArn: *assumeRoleFlag,
		StsExternalID:    *externalIDFlag,
	}

	if userFlag != nil && *userFlag != "" {
		config.Username = *userFlag
	}
	if passwordFlag != nil && *passwordFlag != "" {
		config.Password = *passwordFlag
	}
	if useTLS != nil && *useTLS {
		config.UseTLS = true
	}
	if regionFlag != nil && *regionFlag != "" {
		config.Region = *regionFlag
	}
	if cacheNameFlag != nil && *cacheNameFlag != "" {
		config.CacheName = *cacheNameFlag
	}

	pool, err := redis.NewPool(config)
	if err != nil {
		log.Fatalf("Failed to create Redis pool: %v", err)
	}
	defer pool.Close()

	// Test basic connection
	conn := pool.Get()
	defer conn.Close()

	// Execute PING command
	reply, err := redigo.String(conn.Do("PING"))
	if err != nil {
		log.Fatalf("PING failed: %v", err)
	}
	log.Printf("✅ PING successful: %s", reply)
}
