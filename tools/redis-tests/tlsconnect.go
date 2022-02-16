// Command tlsconnect provides a way to manually test TLS connection to Redis.
// A TLS-enabled redis server must be running locally, this is to verify that
// the configuration get properly passed down to the pool creation.
//
// To run a TLS redis server:
//     * Build redis from source with `make BUILD_TLS=yes` (https://redis.io/topics/encryption)
//     * Generate certificates and keys with `./utils/gen-test-certs.sh`
//       (the generated files will be under ./tests/tls/)
//     * Run `./src/redis-server --tls-port 7379 --port 0 --tls-ca-cert-file
//       ./tests/tls/ca.crt --tls-cert-file ./tests/tls/redis.crt --tls-key-file
//       ./tests/tls/redis.key`
//     * Run this command to test connection, e.g.:
//       `go run ./tools/redis-tests/tlsconnect.go -- -addr localhost:7379 -cacert ./tests/tls/ca.crt
//        -cert ./tests/tls/redis.crt -key ./tests/tls/redis.key PING` -skip
package main

import (
	"flag"
	"log"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

var (
	addrFlag       = flag.String("addr", "", "Redis TLS address, including port")
	certFlag       = flag.String("cert", "", "Redis TLS certificate file")
	keyFlag        = flag.String("key", "", "Redis TLS key file")
	cacertFlag     = flag.String("cacert", "", "Redis TLS CA certificate file")
	serverFlag     = flag.String("server", "", "Redis TLS certificate server name for verification")
	skipVerifyFlag = flag.Bool("skip", false, "Skip verify of TLS certificate")
)

func main() {
	flag.Parse()

	pool, err := redis.NewPool(redis.PoolConfig{
		Server:        *addrFlag,
		UseTLS:        true,
		TLSCA:         *cacertFlag,
		TLSCert:       *certFlag,
		TLSKey:        *keyFlag,
		TLSServerName: *serverFlag,
		TLSSkipVerify: *skipVerifyFlag,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	log.Println("pool created successfully")

	conn := pool.Get()
	defer conn.Close()

	args := flag.Args()
	if len(args) == 0 {
		args = append(args, "PING")
	}

	var rargs redigo.Args
	rargs = rargs.AddFlat(args[1:])
	v, err := conn.Do(args[0], rargs...)
	log.Printf("command result: %v ; %v", v, err)
}
