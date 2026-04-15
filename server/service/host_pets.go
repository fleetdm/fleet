package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	// Name validation.
	maxPetNameLen = 32
	minPetNameLen = 1

	// Happiness decays toward its target by this many points per hour of
	// elapsed time since the pet was last persisted (last_interacted_at).
	// At 2/h, a +12 self-service bump above target dissipates in ~6 hours.
	happinessDecayPerHour = 2

	// Cap simulated elapsed time when computing happiness decay so a pet
	// that hasn't been read in a year doesn't snap from a "fresh bump"
	// straight to the target on the next read — caller still sees a
	// gradual trend if they refresh frequently after a long gap.
	maxHappinessDecayWindow = 7 * 24 * time.Hour
)

//----------------------------------------------------------------------------//
// Device endpoint: GET /device/{token}/pet                                   //
//----------------------------------------------------------------------------//

type getDevicePetRequest struct {
	Token string `url:"token"`
}

func (r *getDevicePetRequest) deviceAuthToken() string { return r.Token }

type getDevicePetResponse struct {
	Err error         `json:"error,omitempty"`
	Pet *fleet.HostPet `json:"pet"`
}

func (r getDevicePetResponse) Error() error { return r.Err }

func getDevicePetEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return getDevicePetResponse{Err: err}, nil
	}

	pet, err := svc.GetDevicePet(ctx, host)
	if err != nil {
		return getDevicePetResponse{Err: err}, nil
	}
	return getDevicePetResponse{Pet: pet}, nil
}

//----------------------------------------------------------------------------//
// Device endpoint: POST /device/{token}/pet (adopt)                          //
//----------------------------------------------------------------------------//

type adoptDevicePetRequest struct {
	Token   string `url:"token"`
	Name    string `json:"name"`
	Species string `json:"species"`
}

func (r *adoptDevicePetRequest) deviceAuthToken() string { return r.Token }

type adoptDevicePetResponse struct {
	Err error         `json:"error,omitempty"`
	Pet *fleet.HostPet `json:"pet"`
}

func (r adoptDevicePetResponse) Error() error { return r.Err }

func adoptDevicePetEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*adoptDevicePetRequest)
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return adoptDevicePetResponse{Err: err}, nil
	}

	pet, err := svc.AdoptDevicePet(ctx, host, req.Name, req.Species)
	if err != nil {
		return adoptDevicePetResponse{Err: err}, nil
	}
	return adoptDevicePetResponse{Pet: pet}, nil
}

//----------------------------------------------------------------------------//
// Device endpoint: POST /device/{token}/pet/action                           //
//----------------------------------------------------------------------------//

type applyDevicePetActionRequest struct {
	Token  string `url:"token"`
	Action string `json:"action"`
}

func (r *applyDevicePetActionRequest) deviceAuthToken() string { return r.Token }

type applyDevicePetActionResponse struct {
	Err error         `json:"error,omitempty"`
	Pet *fleet.HostPet `json:"pet"`
}

func (r applyDevicePetActionResponse) Error() error { return r.Err }

func applyDevicePetActionEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyDevicePetActionRequest)
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return applyDevicePetActionResponse{Err: err}, nil
	}

	pet, err := svc.ApplyDevicePetAction(ctx, host, fleet.HostPetAction(req.Action))
	if err != nil {
		return applyDevicePetActionResponse{Err: err}, nil
	}
	return applyDevicePetActionResponse{Pet: pet}, nil
}

//----------------------------------------------------------------------------//
// Service methods                                                            //
//----------------------------------------------------------------------------//

// GetDevicePet returns the pet for the given host (or nil if unadopted),
// applying decay and device-hygiene signals.
func (svc *Service) GetDevicePet(ctx context.Context, host *fleet.Host) (*fleet.HostPet, error) {
	// skipauth: The device is already authenticated via deviceAuthToken middleware
	// and we only operate on its own pet. There is no cross-host access.
	svc.authz.SkipAuthorization(ctx)

	pet, err := svc.ds.GetHostPet(ctx, host.ID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get host pet")
	}

	metrics, now := svc.gatherHostMetrics(ctx, host)
	applyHostMetricsToPet(pet, metrics, now)
	return pet, nil
}

// AdoptDevicePet creates a new pet for the host. Returns a conflict if the host
// already has a pet.
func (svc *Service) AdoptDevicePet(ctx context.Context, host *fleet.Host, name, species string) (*fleet.HostPet, error) {
	svc.authz.SkipAuthorization(ctx)

	name = strings.TrimSpace(name)
	if species == "" {
		species = string(fleet.HostPetSpeciesCat)
	}

	invalid := &fleet.InvalidArgumentError{}
	if utf8.RuneCountInString(name) < minPetNameLen {
		invalid.Append("name", "cannot be empty")
	}
	if utf8.RuneCountInString(name) > maxPetNameLen {
		invalid.Append("name", "must be 32 characters or fewer")
	}
	if !fleet.IsValidHostPetSpecies(species) {
		invalid.Append("species", "must be a supported species (cat)")
	}
	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}

	pet, err := svc.ds.CreateHostPet(ctx, host.ID, name, species)
	if err != nil {
		var existsErr fleet.AlreadyExistsError
		if errors.As(err, &existsErr) {
			return nil, ctxerr.Wrap(ctx, &fleet.ConflictError{Message: "this host already has a pet"})
		}
		return nil, ctxerr.Wrap(ctx, err, "create host pet")
	}

	metrics, now := svc.gatherHostMetrics(ctx, host)
	applyHostMetricsToPet(pet, metrics, now)
	return pet, nil
}

// ApplyDevicePetAction is deprecated. Pet stats are now driven entirely by
// host hygiene signals (check-ins, failing policies, vulnerabilities, MDM
// posture, disk encryption, and self-service install events) — there's no
// longer any user-driven action that mutates them. The endpoint stays
// registered so older Fleet Desktop clients don't 404, but it always
// returns 410 Gone.
func (svc *Service) ApplyDevicePetAction(ctx context.Context, _ *fleet.Host, _ fleet.HostPetAction) (*fleet.HostPet, error) {
	svc.authz.SkipAuthorization(ctx)
	return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
		Message:     "pet actions have been removed; the pet now reacts to your device's hygiene automatically",
		InternalErr: errors.New("ApplyDevicePetAction is deprecated"),
	})
}

//----------------------------------------------------------------------------//
// Internal: host-metrics derivation                                          //
//----------------------------------------------------------------------------//

// gatherHostMetrics builds the HostPetMetrics snapshot the derivation reads,
// pulling failing policies + open vulns from the datastore and overlaying any
// demo overrides. Returns the metrics and the "now" timestamp the derivation
// should use (real now + any demo time offset).
func (svc *Service) gatherHostMetrics(ctx context.Context, host *fleet.Host) (fleet.HostPetMetrics, time.Time) {
	now := svc.clock.Now()
	m := fleet.HostPetMetrics{
		SeenTime:              host.SeenTime,
		DiskEncryptionEnabled: host.DiskEncryptionEnabled,
		MDMUnenrolled:         host.MDM.EnrollmentStatus != nil && *host.MDM.EnrollmentStatus == "Off",
	}

	// Failing policies. ListPoliciesForHost may return an error on hosts with
	// no membership rows; we treat that as zero failing policies rather than
	// failing the whole request, matching the previous behaviour.
	if policies, err := svc.ds.ListPoliciesForHost(ctx, host); err == nil {
		for _, p := range policies {
			if p.Response == "fail" {
				m.FailingPolicyCount++
			}
		}
	}

	// Open vulns by severity. Same forgiving treatment — a transient vuln
	// query failure shouldn't blank the pet.
	if crit, high, err := svc.ds.CountOpenHostVulnsBySeverity(ctx, host.ID); err == nil {
		m.CriticalVulnCount = crit
		m.HighVulnCount = high
	}

	// Demo overlay. GetHostPetDemoOverrides returns (nil, nil) when no row
	// exists, so this is a no-op on real deployments.
	if overrides, err := svc.ds.GetHostPetDemoOverrides(ctx, host.ID); err == nil && overrides != nil {
		overrides.Apply(&m)
		if overrides.TimeOffsetHours != 0 {
			now = now.Add(time.Duration(overrides.TimeOffsetHours) * time.Hour)
		}
	}

	return m, now
}

// applyHostMetricsToPet derives the pet's display stats from a host metrics
// snapshot. Pure function over (pet, metrics, now) so it's table-testable
// without any mocks. Mutates pet in place; does NOT persist.
//
// Hunger / cleanliness / health snap to whatever the host metrics say —
// these stats are the pet's *display* of the host's posture, not an
// accumulator the user can grind. Happiness is the one persisted stat: it
// decays over time toward a target derived from disk encryption posture, and
// is bumped above target by event-driven signals (currently: successful
// self-service installs / uninstalls).
func applyHostMetricsToPet(pet *fleet.HostPet, m fleet.HostPetMetrics, now time.Time) {
	pet.Hunger = hungerFromMetrics(m, now)
	pet.Cleanliness = cleanlinessFromMetrics(m)
	pet.Health = healthFromMetrics(m)

	target := happinessTargetFromMetrics(m)
	pet.Happiness = decayedHappiness(pet.Happiness, target, now.Sub(pet.LastInteractedAt))

	pet.Mood = computeMood(pet)
}

// hungerFromMetrics: time-since-check-in → hunger band. Higher hunger means
// the pet is hungrier (i.e. the host hasn't checked in recently).
func hungerFromMetrics(m fleet.HostPetMetrics, now time.Time) uint8 {
	if m.SeenTime.IsZero() {
		// Brand-new host that's never checked in. Don't punish — return
		// the baseline so the pet looks normal until the first check-in.
		return fleet.HostPetTargetHungerBaseline
	}
	hours := now.Sub(m.SeenTime).Hours()
	switch {
	case hours < fleet.HostPetHungerHoursFresh:
		return fleet.HostPetTargetHungerFresh
	case hours < fleet.HostPetHungerHoursStale:
		return fleet.HostPetTargetHungerStale
	case hours < fleet.HostPetHungerHoursVeryStale:
		return fleet.HostPetTargetHungerVeryStale
	default:
		return fleet.HostPetTargetHungerStarving
	}
}

// cleanlinessFromMetrics: each failing policy drags cleanliness down from the
// baseline. All-passing → baseline.
func cleanlinessFromMetrics(m fleet.HostPetMetrics) uint8 {
	drag := int(m.FailingPolicyCount) * int(fleet.HostPetCleanlinessPerFailingPolicy)
	return clampStat(int(fleet.HostPetTargetCleanlinessBaseline) - drag)
}

// healthFromMetrics: critical/high vulns drain health hardest. MDM unenrolled
// adds a small additional penalty.
func healthFromMetrics(m fleet.HostPetMetrics) uint8 {
	v := int(fleet.HostPetTargetHealthBaseline)
	v -= int(m.CriticalVulnCount) * int(fleet.HostPetHealthPerCriticalVuln)
	v -= int(m.HighVulnCount) * int(fleet.HostPetHealthPerHighVuln)
	if m.MDMUnenrolled {
		v -= int(fleet.HostPetHealthMDMUnenrolledPenalty)
	}
	return clampStat(v)
}

// happinessTargetFromMetrics: the floor that persisted happiness decays
// toward. Disk encryption is the only host-state signal that moves the
// target right now — self-service events bump happiness above it.
func happinessTargetFromMetrics(m fleet.HostPetMetrics) uint8 {
	v := int(fleet.HostPetTargetHappinessBaseline)
	if m.DiskEncryptionEnabled != nil {
		if *m.DiskEncryptionEnabled {
			v += int(fleet.HostPetHappinessDiskEncOnBonus)
		} else {
			v -= int(fleet.HostPetHappinessDiskEncOffPenalty)
		}
	}
	return clampStat(v)
}

// decayedHappiness slides current toward target by happinessDecayPerHour
// per hour of elapsed time, capped at maxHappinessDecayWindow so a long
// idle stretch doesn't snap straight to target.
func decayedHappiness(current, target uint8, elapsed time.Duration) uint8 {
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > maxHappinessDecayWindow {
		elapsed = maxHappinessDecayWindow
	}
	step := int(elapsed.Hours() * float64(happinessDecayPerHour))
	if step <= 0 {
		return current
	}
	if current < target {
		next := int(current) + step
		if next > int(target) {
			next = int(target)
		}
		return clampStat(next)
	}
	if current > target {
		next := int(current) - step
		if next < int(target) {
			next = int(target)
		}
		return clampStat(next)
	}
	return current
}

// clampStat constrains v to [HostPetStatFloor, HostPetStatCeiling].
func clampStat(v int) uint8 {
	if v < int(fleet.HostPetStatFloor) {
		v = int(fleet.HostPetStatFloor)
	}
	if v > int(fleet.HostPetStatCeiling) {
		v = int(fleet.HostPetStatCeiling)
	}
	return uint8(v) //nolint:gosec // clamped above
}

// computeMood returns a single word that best summarises the pet's state.
// Priority: sick (health) > hungry > dirty > sad > content > happy.
func computeMood(pet *fleet.HostPet) fleet.HostPetMood {
	switch {
	case pet.Health < 30:
		return fleet.HostPetMoodSick
	case pet.Hunger > 75:
		return fleet.HostPetMoodHungry
	case pet.Cleanliness < 25:
		return fleet.HostPetMoodDirty
	case pet.Happiness < 35:
		return fleet.HostPetMoodSad
	case pet.Happiness > 75 && pet.Health > 75 && pet.Cleanliness > 60:
		return fleet.HostPetMoodHappy
	default:
		return fleet.HostPetMoodContent
	}
}

