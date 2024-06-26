package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	orbittable "github.com/fleetdm/fleet/v4/orbit/pkg/table"
	"github.com/osquery/osquery-go"
)

var (
	socket   = flag.String("socket", "", "Path to the extensions UNIX domain socket")
	timeout  = flag.Int("timeout", 3, "Seconds to wait for autoloaded extensions")
	interval = flag.Int("interval", 3, "Seconds delay between connectivity checks")
	// verbose must be set because osqueryd will set it on the extension when running in verbose mode.
	_ = flag.Bool("verbose", false, "Enable verbose informational messages")
)

func main() {
	flag.Parse()

	if *socket == "" {
		log.Fatalf(`Usage: %s -socket SOCKET_PATH`, os.Args[0])
	}

	serverTimeout := osquery.ServerTimeout(
		time.Second * time.Duration(*timeout),
	)
	serverPingInterval := osquery.ServerPingInterval(
		time.Second * time.Duration(*interval),
	)

	var server *osquery.ExtensionManagerServer
	backOff := backoff.WithMaxRetries(backoff.NewConstantBackOff(time.Millisecond*200), 25) // retry once per 200ms for 25 times == 5 seconds
	op := func() error {
		s, err := osquery.NewExtensionManagerServer("com.fleetdm.fleetd_tables.osquery_extension.v1", *socket, serverTimeout, serverPingInterval)
		if err != nil {
			return fmt.Errorf("error creating extension: %w", err)
		}
		server = s
		return nil
	}

	err := backoff.Retry(op, backOff)
	if err != nil {
		log.Fatalln(err)
	}

	plugins := orbittable.OrbitDefaultTables()
	opts := orbittable.PluginOpts{
		Socket: *socket,
	}
	platformTables, err := orbittable.PlatformTables(opts)
	if err != nil {
		log.Fatalln(err)
	}
	plugins = append(plugins, platformTables...)
	server.RegisterPlugin(plugins...)
	if err := server.Run(); err != nil {
		log.Fatalln(err)
	}
}
