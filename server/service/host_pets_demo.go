//go:build pet_demo

// Package service: pet demo endpoints.
//
// These endpoints exist purely to drive the host_pets feature for live
// demos — they let an admin override what the pet derivation function
// "sees" (failing policies, vulnerability counts, time-since-last-checkin)
// without actually mutating policies, hosts, or vulnerabilities tables.
//
// They are gated by *both* a compile-time build tag and a runtime env
// var so they cannot accidentally reach production:
//
//   1. Compiled in only when the binary is built with `-tags pet_demo`.
//   2. Even then, every handler 404s unless FLEET_ENABLE_PET_DEMO=1 is
//      set on the running server.
//   3. The handlers also require global-admin auth.
//
// Documentation for usage lives in `43625-pet-host-metrics-plan.md`.

package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	eu "github.com/fleetdm/fleet/v4/server/platform/endpointer"
)

// petDemoEnvVar is the runtime gate. Set FLEET_ENABLE_PET_DEMO=1 on a
// pet_demo-tagged binary to actually serve these endpoints.
const petDemoEnvVar = "FLEET_ENABLE_PET_DEMO"

func petDemoEnabled() bool { return os.Getenv(petDemoEnvVar) == "1" }

// registerPetDemoEndpoints wires the demo routes onto the user-authenticated
// endpointer. Called from handler.go unconditionally; the stub version (in
// host_pets_demo_stub.go, default build) does nothing.
func registerPetDemoEndpoints(ue *eu.CommonEndpointer[handlerFunc]) {
	ue.GET("/api/_version_/fleet/hosts/{id:[0-9]+}/pet/demo/overrides",
		getHostPetDemoOverridesEndpoint, getHostPetDemoOverridesRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/pet/demo/overrides",
		upsertHostPetDemoOverridesEndpoint, upsertHostPetDemoOverridesRequest{})
	ue.DELETE("/api/_version_/fleet/hosts/{id:[0-9]+}/pet/demo/overrides",
		deleteHostPetDemoOverridesEndpoint, deleteHostPetDemoOverridesRequest{})
	ue.POST("/api/_version_/fleet/hosts/{id:[0-9]+}/pet/demo/simulate_self_service",
		simulateSelfServiceEndpoint, simulateSelfServiceRequest{})
}

// requirePetDemoAdmin enforces the runtime env-var gate AND a global-admin
// check. Returns nil on success, an error on rejection.
func requirePetDemoAdmin(ctx context.Context, svc *Service) error {
	if !petDemoEnabled() {
		// 404 rather than 403 — when the gate is off the endpoint shouldn't
		// even appear to exist.
		return ctxerr.Wrap(ctx, &notFoundError{})
	}
	// Satisfy the authz middleware with a baseline host check.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}
	if vc.User == nil || vc.User.GlobalRole == nil || *vc.User.GlobalRole != fleet.RoleAdmin {
		return fleet.NewPermissionError("pet demo endpoints require global admin")
	}
	return nil
}

//----------------------------------------------------------------------------//
// GET /pet/demo/overrides                                                    //
//----------------------------------------------------------------------------//

type getHostPetDemoOverridesRequest struct {
	ID uint `url:"id"`
}

type getHostPetDemoOverridesResponse struct {
	HostID    uint                        `json:"host_id"`
	Overrides *fleet.HostPetDemoOverrides `json:"overrides"`
	Err       error                       `json:"error,omitempty"`
}

func (r getHostPetDemoOverridesResponse) Error() error { return r.Err }

func getHostPetDemoOverridesEndpoint(ctx context.Context, request any, sIface fleet.Service) (fleet.Errorer, error) {
	req := request.(*getHostPetDemoOverridesRequest)
	svc, ok := sIface.(*Service)
	if !ok {
		return getHostPetDemoOverridesResponse{Err: errors.New("internal: service is not *Service")}, nil
	}
	if err := requirePetDemoAdmin(ctx, svc); err != nil {
		return getHostPetDemoOverridesResponse{Err: err}, nil
	}
	o, err := svc.ds.GetHostPetDemoOverrides(ctx, req.ID)
	if err != nil {
		return getHostPetDemoOverridesResponse{Err: ctxerr.Wrap(ctx, err, "get host pet demo overrides")}, nil
	}
	return getHostPetDemoOverridesResponse{HostID: req.ID, Overrides: o}, nil
}

//----------------------------------------------------------------------------//
// POST /pet/demo/overrides   (upsert; merges into existing row)              //
//----------------------------------------------------------------------------//

type upsertHostPetDemoOverridesRequest struct {
	ID uint `url:"id"`
	// All fields optional. Pass only what you want to change. Unset fields
	// leave the existing value alone (PATCH-like semantics).
	SeenTimeOverride     *time.Time `json:"seen_time_override,omitempty"`
	TimeOffsetHours      *int       `json:"time_offset_hours,omitempty"`
	ExtraFailingPolicies *uint      `json:"extra_failing_policies,omitempty"`
	ExtraCriticalVulns   *uint      `json:"extra_critical_vulns,omitempty"`
	ExtraHighVulns       *uint      `json:"extra_high_vulns,omitempty"`
	// ClearSeenTimeOverride lets the caller explicitly null out
	// seen_time_override (otherwise sending no field leaves it alone).
	ClearSeenTimeOverride bool `json:"clear_seen_time_override,omitempty"`
}

type upsertHostPetDemoOverridesResponse struct {
	HostID    uint                        `json:"host_id"`
	Overrides *fleet.HostPetDemoOverrides `json:"overrides"`
	Err       error                       `json:"error,omitempty"`
}

func (r upsertHostPetDemoOverridesResponse) Error() error { return r.Err }

func upsertHostPetDemoOverridesEndpoint(ctx context.Context, request any, sIface fleet.Service) (fleet.Errorer, error) {
	req := request.(*upsertHostPetDemoOverridesRequest)
	svc, ok := sIface.(*Service)
	if !ok {
		return upsertHostPetDemoOverridesResponse{Err: errors.New("internal: service is not *Service")}, nil
	}
	if err := requirePetDemoAdmin(ctx, svc); err != nil {
		return upsertHostPetDemoOverridesResponse{Err: err}, nil
	}

	existing, err := svc.ds.GetHostPetDemoOverrides(ctx, req.ID)
	if err != nil {
		return upsertHostPetDemoOverridesResponse{Err: ctxerr.Wrap(ctx, err, "load existing overrides for merge")}, nil
	}
	merged := &fleet.HostPetDemoOverrides{HostID: req.ID}
	if existing != nil {
		*merged = *existing
		merged.HostID = req.ID
	}
	if req.ClearSeenTimeOverride {
		merged.SeenTimeOverride = nil
	} else if req.SeenTimeOverride != nil {
		merged.SeenTimeOverride = req.SeenTimeOverride
	}
	if req.TimeOffsetHours != nil {
		merged.TimeOffsetHours = *req.TimeOffsetHours
	}
	if req.ExtraFailingPolicies != nil {
		merged.ExtraFailingPolicies = *req.ExtraFailingPolicies
	}
	if req.ExtraCriticalVulns != nil {
		merged.ExtraCriticalVulns = *req.ExtraCriticalVulns
	}
	if req.ExtraHighVulns != nil {
		merged.ExtraHighVulns = *req.ExtraHighVulns
	}

	if err := svc.ds.UpsertHostPetDemoOverrides(ctx, merged); err != nil {
		return upsertHostPetDemoOverridesResponse{Err: ctxerr.Wrap(ctx, err, "upsert host pet demo overrides")}, nil
	}
	// Re-read so timestamps are populated.
	o, err := svc.ds.GetHostPetDemoOverrides(ctx, req.ID)
	if err != nil {
		return upsertHostPetDemoOverridesResponse{Err: ctxerr.Wrap(ctx, err, "reload after upsert")}, nil
	}
	return upsertHostPetDemoOverridesResponse{HostID: req.ID, Overrides: o}, nil
}

//----------------------------------------------------------------------------//
// DELETE /pet/demo/overrides   (reset to no overrides)                        //
//----------------------------------------------------------------------------//

type deleteHostPetDemoOverridesRequest struct {
	ID uint `url:"id"`
}

type deleteHostPetDemoOverridesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteHostPetDemoOverridesResponse) Error() error { return r.Err }
func (r deleteHostPetDemoOverridesResponse) Status() int  { return http.StatusNoContent }

func deleteHostPetDemoOverridesEndpoint(ctx context.Context, request any, sIface fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteHostPetDemoOverridesRequest)
	svc, ok := sIface.(*Service)
	if !ok {
		return deleteHostPetDemoOverridesResponse{Err: errors.New("internal: service is not *Service")}, nil
	}
	if err := requirePetDemoAdmin(ctx, svc); err != nil {
		return deleteHostPetDemoOverridesResponse{Err: err}, nil
	}
	if err := svc.ds.DeleteHostPetDemoOverrides(ctx, req.ID); err != nil {
		return deleteHostPetDemoOverridesResponse{Err: ctxerr.Wrap(ctx, err, "delete host pet demo overrides")}, nil
	}
	return deleteHostPetDemoOverridesResponse{}, nil
}

//----------------------------------------------------------------------------//
// POST /pet/demo/simulate_self_service   (one-shot happiness bump)            //
//----------------------------------------------------------------------------//

type simulateSelfServiceRequest struct {
	ID    uint `url:"id"`
	Delta int  `json:"delta"`
}

type simulateSelfServiceResponse struct {
	Err error `json:"error,omitempty"`
}

func (r simulateSelfServiceResponse) Error() error { return r.Err }
func (r simulateSelfServiceResponse) Status() int  { return http.StatusNoContent }

func simulateSelfServiceEndpoint(ctx context.Context, request any, sIface fleet.Service) (fleet.Errorer, error) {
	req := request.(*simulateSelfServiceRequest)
	svc, ok := sIface.(*Service)
	if !ok {
		return simulateSelfServiceResponse{Err: errors.New("internal: service is not *Service")}, nil
	}
	if err := requirePetDemoAdmin(ctx, svc); err != nil {
		return simulateSelfServiceResponse{Err: err}, nil
	}
	delta := req.Delta
	if delta == 0 {
		// Default to the same bump a real install applies. Lets curl-only
		// demos call POST .../simulate_self_service with no body.
		delta = int(fleet.HostPetHappinessSelfServiceBump)
	}
	if err := svc.ds.ApplyHostPetHappinessDelta(ctx, req.ID, delta); err != nil {
		return simulateSelfServiceResponse{Err: ctxerr.Wrap(ctx, err, "apply host pet happiness delta")}, nil
	}
	return simulateSelfServiceResponse{}, nil
}
