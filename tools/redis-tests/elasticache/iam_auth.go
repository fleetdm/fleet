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
	userFlag       = flag.String("user", "", "Username for IAM authentication")
	assumeRoleFlag = flag.String("assume-role", "", "STS assume role ARN (optional)")
	externalIDFlag = flag.String("external-id", "", "STS external ID (optional)")
)

func main() {
	flag.Parse()

	if *addrFlag == "" {
		log.Fatal("ElastiCache address is required (-addr flag)")
	}
	if *userFlag == "" {
		log.Fatal("Username is required (-user flag)")
	}

	log.Printf("Connecting to ElastiCache at %s with IAM auth for user %s", *addrFlag, *userFlag)
	if *assumeRoleFlag != "" {
		log.Printf("Using assume role: %s", *assumeRoleFlag)
	}

	pool, err := redis.NewPool(redis.PoolConfig{
		Server:           *addrFlag,
		Username:         *userFlag,
		UseTLS:           true,
		StsAssumeRoleArn: *assumeRoleFlag,
		StsExternalID:    *externalIDFlag,
	})
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
