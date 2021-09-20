package main

import (
	"flag"
	"log"
	"time"

	"github.com/kolide/osquery-go"
	"github.com/kolide/osquery-go/plugin/table"
	"github.com/macadmins/osquery-extension/tables/mdm"
)

const (
	orbitExtension = "com.fleetdm.orbit.osquery_extension.v1"
)

// TODO: move darwin specific tables to _darwin.go
func platformTables() []osquery.OsqueryPlugin {
	var plugins []osquery.OsqueryPlugin
	plugins = append(plugins, table.NewPlugin("mdm", mdm.MDMInfoColumns(), mdm.MDMInfoGenerate))
	return plugins
}

func main() {
	var (
		flSocketPath = flag.String("socket", "", "")
		flTimeout    = flag.Int("timeout", 0, "")
		_            = flag.Int("interval", 0, "")
		_            = flag.Bool("verbose", false, "")
	)
	flag.Parse()

	server, err := osquery.NewExtensionManagerServer(
		orbitExtension,
		*flSocketPath,
		osquery.ServerTimeout(time.Duration(*flTimeout)*time.Second),
	)

	if err != nil {
		log.Fatal(err)
	}

	var plugins []osquery.OsqueryPlugin
	for _, t := range platformTables() {
		plugins = append(plugins, t)
	}
	server.RegisterPlugin(plugins...)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
