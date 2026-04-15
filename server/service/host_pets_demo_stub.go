//go:build !pet_demo

package service

import (
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// registerPetDemoEndpoints is a no-op in production builds. The full
// implementation lives in host_pets_demo.go and is only compiled when the
// `pet_demo` build tag is set, AND requires the FLEET_ENABLE_PET_DEMO=1 env
// var to actually serve. See `43625-pet-host-metrics-plan.md`.
func registerPetDemoEndpoints(_ *eu.CommonEndpointer[handlerFunc]) {}
