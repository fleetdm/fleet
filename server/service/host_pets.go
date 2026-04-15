package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

//----------------------------------------------------------------------------//
// Pet stat decay / signal tuning constants                                   //
//----------------------------------------------------------------------------//

const (
	// Passive decay per hour of neglect (applied on every read).
	decayHungerPerHour      = 3 // hunger rises (higher = hungrier)
	decayCleanlinessPerHour = 2 // cleanliness drops
	decayHappinessPerHour   = 1 // happiness drops slowly
	// Health does not decay on its own; it drains from sustained bad stats.

	// Action effects (applied on POST /pet/action).
	actionFeedHungerDelta      = -30 // Feed: lower hunger
	actionFeedHappinessDelta   = 5
	actionPlayHappinessDelta   = 30
	actionPlayHungerDelta      = 10 // playing makes you hungry
	actionCleanCleanlinessDelta = 40
	actionCleanHappinessDelta  = 5
	actionMedicineHealthDelta  = 30

	// Device-hygiene signal effects (applied on every read, once per read).
	signalFailingPolicyHealthPerPolicy     = -5
	signalDiskEncryptionOffHappinessDelta  = -10
	signalMdmUnenrolledHealthDelta         = -10
	signalAllPoliciesPassingHealthDelta    = 2
	signalDiskEncryptionOnHappinessDelta   = 2

	// Name validation.
	maxPetNameLen = 32
	minPetNameLen = 1
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

	svc.applyDecayAndSignals(ctx, pet, host)
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

	svc.applyDecayAndSignals(ctx, pet, host)
	return pet, nil
}

// ApplyDevicePetAction applies a single action to the host's pet (feed, play,
// clean, medicine) and returns the updated pet. Passive decay and device-hygiene
// signals are applied first.
func (svc *Service) ApplyDevicePetAction(ctx context.Context, host *fleet.Host, action fleet.HostPetAction) (*fleet.HostPet, error) {
	svc.authz.SkipAuthorization(ctx)

	if !fleet.IsValidHostPetAction(string(action)) {
		invalid := fleet.NewInvalidArgumentError("action", "must be one of feed, play, clean, medicine")
		return nil, ctxerr.Wrap(ctx, invalid)
	}

	pet, err := svc.ds.GetHostPet(ctx, host.ID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{Message: "no pet found for this host - adopt one first"})
		}
		return nil, ctxerr.Wrap(ctx, err, "get host pet")
	}

	// Apply decay and device signals before the action so each action is
	// measured against the current state of the pet, not a stale snapshot.
	svc.applyDecayAndSignals(ctx, pet, host)

	// Apply the action's deltas.
	switch action {
	case fleet.HostPetActionFeed:
		// Over-feeding: if hunger is already low, the pet gets stuffed and sad.
		if pet.Hunger < 20 {
			pet.Happiness = adjustStat(pet.Happiness, -10)
			pet.Health = adjustStat(pet.Health, -2)
		} else {
			pet.Happiness = adjustStat(pet.Happiness, actionFeedHappinessDelta)
		}
		pet.Hunger = adjustStat(pet.Hunger, actionFeedHungerDelta)

	case fleet.HostPetActionPlay:
		pet.Happiness = adjustStat(pet.Happiness, actionPlayHappinessDelta)
		pet.Hunger = adjustStat(pet.Hunger, actionPlayHungerDelta)

	case fleet.HostPetActionClean:
		pet.Cleanliness = adjustStat(pet.Cleanliness, actionCleanCleanlinessDelta)
		pet.Happiness = adjustStat(pet.Happiness, actionCleanHappinessDelta)

	case fleet.HostPetActionMedicine:
		pet.Health = adjustStat(pet.Health, actionMedicineHealthDelta)
	}

	// An action counts as an interaction — reset the decay clock.
	pet.LastInteractedAt = svc.clock.Now()

	if err := svc.ds.SaveHostPet(ctx, pet); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "save host pet")
	}

	// Recompute derived mood for the response.
	pet.Mood = computeMood(pet)
	return pet, nil
}

//----------------------------------------------------------------------------//
// Internal: decay + signal application                                       //
//----------------------------------------------------------------------------//

// applyDecayAndSignals mutates the pet in-place by applying:
//   1. Passive decay based on time elapsed since last_interacted_at.
//   2. Device-hygiene signals (failing policies, disk encryption, MDM).
//   3. Cross-stat feedback: if hunger/cleanliness are maxed, health drains.
//
// This intentionally does *not* persist the pet — we only write to the DB on
// explicit actions. This keeps GETs cheap and avoids the user's browser silently
// grinding down their pet by refreshing.
func (svc *Service) applyDecayAndSignals(ctx context.Context, pet *fleet.HostPet, host *fleet.Host) {
	now := svc.clock.Now()
	hours := now.Sub(pet.LastInteractedAt).Hours()
	if hours < 0 {
		hours = 0
	}

	// 1. Passive decay (fractional-friendly via int truncation — fine for a pet).
	pet.Hunger = adjustStat(pet.Hunger, int(hours*float64(decayHungerPerHour)))
	pet.Cleanliness = adjustStat(pet.Cleanliness, -int(hours*float64(decayCleanlinessPerHour)))
	pet.Happiness = adjustStat(pet.Happiness, -int(hours*float64(decayHappinessPerHour)))

	// 2. Device-hygiene signals. These are "what the pet sees right now", not a
	//    running accumulator — they're applied every read but the result is
	//    bounded by the stat clamps, so they don't compound unbounded.
	policies, err := svc.ds.ListPoliciesForHost(ctx, host)
	if err == nil && len(policies) > 0 {
		failing := 0
		for _, p := range policies {
			if p.Response == "fail" {
				failing++
			}
		}
		if failing == 0 {
			pet.Health = adjustStat(pet.Health, signalAllPoliciesPassingHealthDelta)
		} else {
			pet.Health = adjustStat(pet.Health, failing*signalFailingPolicyHealthPerPolicy)
		}
	}

	if host.DiskEncryptionEnabled != nil {
		if *host.DiskEncryptionEnabled {
			pet.Happiness = adjustStat(pet.Happiness, signalDiskEncryptionOnHappinessDelta)
		} else {
			pet.Happiness = adjustStat(pet.Happiness, signalDiskEncryptionOffHappinessDelta)
		}
	}

	if host.MDM.EnrollmentStatus != nil && *host.MDM.EnrollmentStatus == "Off" {
		pet.Health = adjustStat(pet.Health, signalMdmUnenrolledHealthDelta)
	}

	// 3. Cross-stat feedback: a starving, filthy pet loses health.
	if pet.Hunger >= 90 {
		pet.Health = adjustStat(pet.Health, -5)
	}
	if pet.Cleanliness <= 10 {
		pet.Health = adjustStat(pet.Health, -5)
	}

	// 4. Derived mood.
	pet.Mood = computeMood(pet)
}

// adjustStat clamps stat ± delta to [HostPetStatFloor, HostPetStatCeiling].
// Using int for delta keeps callers ergonomic (negative deltas, int8 overflow
// sidestepped) while the stored value stays uint8.
func adjustStat(stat uint8, delta int) uint8 {
	v := int(stat) + delta
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

