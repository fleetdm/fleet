package main

import (
	"context"
	"os"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/ee/maintained-apps/inputs/darwin"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func main() {
	ctx := context.Background()
	logger := kitlog.NewJSONLogger(os.Stderr)
	logger = level.NewFilter(logger, level.AllowDebug())
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)

	level.Info(logger).Log("msg", "starting maintained app ingestion")

	// init ingesters for different platforms
	var ingesters []maintained_apps.Ingester
	ingesters = append(ingesters, darwin.NewDarwinIngester(logger))

	for _, i := range ingesters {
		if err := i.IngestApps(ctx); err != nil {
			level.Error(logger).Log("msg", "failed to ingest apps", "error", err)
		}
	}
}
