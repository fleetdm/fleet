package main

import "github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate/log"

func init() {
	log.Options.SetWithCaller()
	log.Options.SetWithLevel()
	log.Level = log.LevelDebug
}
