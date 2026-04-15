package fleet

import (
	"time"
)

// HostPetSpecies enumerates the available pet species. Only cat is supported at
// the moment; additional species (dog, dragon, etc.) may be added later.
type HostPetSpecies string

const (
	HostPetSpeciesCat HostPetSpecies = "cat"
)

// IsValidHostPetSpecies reports whether the given species string is one we
// currently support.
func IsValidHostPetSpecies(s string) bool {
	switch HostPetSpecies(s) {
	case HostPetSpeciesCat:
		return true
	}
	return false
}

// HostPetAction enumerates the actions a user can take to interact with their
// pet.
type HostPetAction string

const (
	HostPetActionFeed     HostPetAction = "feed"
	HostPetActionPlay     HostPetAction = "play"
	HostPetActionClean    HostPetAction = "clean"
	HostPetActionMedicine HostPetAction = "medicine"
)

// IsValidHostPetAction reports whether the given action string is supported.
func IsValidHostPetAction(a string) bool {
	switch HostPetAction(a) {
	case HostPetActionFeed, HostPetActionPlay, HostPetActionClean, HostPetActionMedicine:
		return true
	}
	return false
}

// HostPetMood is a derived, human-readable mood string computed from the pet's
// current stats. It is not persisted.
type HostPetMood string

const (
	HostPetMoodHappy   HostPetMood = "happy"
	HostPetMoodContent HostPetMood = "content"
	HostPetMoodSad     HostPetMood = "sad"
	HostPetMoodHungry  HostPetMood = "hungry"
	HostPetMoodDirty   HostPetMood = "dirty"
	HostPetMoodSick    HostPetMood = "sick"
)

// HostPetStatFloor and HostPetStatCeiling bound each stat. Health will never
// drop below the floor so the pet can always be revived.
const (
	HostPetStatFloor   uint8 = 1
	HostPetStatCeiling uint8 = 100
)

// HostPet is a user's virtual pet associated with a single host. Stats are
// stored as a snapshot at last_interacted_at and are decayed on read.
type HostPet struct {
	ID               uint      `json:"id" db:"id"`
	HostID           uint      `json:"host_id" db:"host_id"`
	Name             string    `json:"name" db:"name"`
	Species          string    `json:"species" db:"species"`
	Health           uint8     `json:"health" db:"health"`
	Happiness        uint8     `json:"happiness" db:"happiness"`
	Hunger           uint8     `json:"hunger" db:"hunger"`
	Cleanliness      uint8     `json:"cleanliness" db:"cleanliness"`
	LastInteractedAt time.Time `json:"last_interacted_at" db:"last_interacted_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`

	// Mood is derived from the stats above and is computed by the service layer
	// before returning the pet to the caller.
	Mood HostPetMood `json:"mood" db:"-"`
}

// HostPetAdoption is the payload required to adopt a new pet for a host.
type HostPetAdoption struct {
	Name    string `json:"name"`
	Species string `json:"species"`
}

// HostPetMetrics is the snapshot of host hygiene signals the pet derivation
// reads to compute its stats. Built once per request from real host state and
// (when the demo build tag is enabled) overlaid with a HostPetDemoOverrides
// row.
type HostPetMetrics struct {
	// SeenTime is when Fleet last heard from the host. Drives hunger.
	SeenTime time.Time
	// FailingPolicyCount is how many policies are failing on this host right
	// now. Drives cleanliness.
	FailingPolicyCount uint
	// CriticalVulnCount + HighVulnCount are open CVEs on installed software,
	// bucketed by CVSS severity. Drive health.
	CriticalVulnCount uint
	HighVulnCount     uint
	// DiskEncryptionEnabled is the host's current FileVault / BitLocker /
	// LUKS state. Tri-state because some hosts haven't reported it yet.
	DiskEncryptionEnabled *bool
	// MDMUnenrolled is true when the host's MDM enrollment_status is "Off".
	MDMUnenrolled bool
}

// HostPetDemoOverrides are per-host knobs the demo build can layer on top of
// real host state to drive stat changes without touching the underlying
// hosts/policies/vulnerabilities tables. Reset by deleting the row.
type HostPetDemoOverrides struct {
	HostID               uint       `json:"host_id" db:"host_id"`
	SeenTimeOverride     *time.Time `json:"seen_time_override,omitempty" db:"seen_time_override"`
	TimeOffsetHours      int        `json:"time_offset_hours" db:"time_offset_hours"`
	ExtraFailingPolicies uint       `json:"extra_failing_policies" db:"extra_failing_policies"`
	ExtraCriticalVulns   uint       `json:"extra_critical_vulns" db:"extra_critical_vulns"`
	ExtraHighVulns       uint       `json:"extra_high_vulns" db:"extra_high_vulns"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// Apply overlays the override values onto a real metrics snapshot. Counts
// are added; SeenTimeOverride wholesale replaces SeenTime when set.
// TimeOffsetHours is consumed by the caller (it shifts "now"), not here.
func (o *HostPetDemoOverrides) Apply(m *HostPetMetrics) {
	if o == nil || m == nil {
		return
	}
	if o.SeenTimeOverride != nil {
		m.SeenTime = *o.SeenTimeOverride
	}
	m.FailingPolicyCount += o.ExtraFailingPolicies
	m.CriticalVulnCount += o.ExtraCriticalVulns
	m.HighVulnCount += o.ExtraHighVulns
}

//----------------------------------------------------------------------------//
// Pet stat derivation tuning constants                                       //
//----------------------------------------------------------------------------//

// Targets are the per-stat values that host signals push the pet toward.
// On every read, each stat slides toward its target by HostPetTickStep so that
// transient signals don't snap the pet's display from one extreme to another.
const (
	// Default ("nothing wrong") targets when there are no negative signals.
	HostPetTargetHealthBaseline      uint8 = 90
	HostPetTargetCleanlinessBaseline uint8 = 90
	HostPetTargetHungerBaseline      uint8 = 20 // low hunger = full
	HostPetTargetHappinessBaseline   uint8 = 70

	// HostPetTickStep is the maximum amount any single stat can move per read
	// when sliding toward its target. Keeps the UI from flipping wildly.
	HostPetTickStep uint8 = 8

	// Hunger thresholds keyed off hours since last check-in.
	HostPetHungerHoursFresh = 1.0  // <1h since check-in: pet is fed
	HostPetHungerHoursStale = 6.0  // 1-6h: trending toward "peckish"
	HostPetHungerHoursVeryStale = 24.0 // >24h: starving

	// Hunger targets for each band above.
	HostPetTargetHungerFresh     uint8 = 15
	HostPetTargetHungerStale     uint8 = 50
	HostPetTargetHungerVeryStale uint8 = 80
	HostPetTargetHungerStarving  uint8 = 95

	// Cleanliness drag per failing policy (1 failing policy = -15 cleanliness).
	HostPetCleanlinessPerFailingPolicy uint8 = 15

	// Health drag per vulnerability bucket.
	HostPetHealthPerCriticalVuln uint8 = 10
	HostPetHealthPerHighVuln     uint8 = 3

	// MDM/disk-encryption signals (applied as direct stat overlays, not
	// targets, since they're binary).
	HostPetHealthMDMUnenrolledPenalty   uint8 = 10
	HostPetHappinessDiskEncOnBonus      uint8 = 5
	HostPetHappinessDiskEncOffPenalty   uint8 = 10

	// Self-service event happiness bump (applied event-driven by the install
	// success path, persisted on the pet row, then decayed naturally over
	// time toward the happiness baseline).
	HostPetHappinessSelfServiceBump uint8 = 12
)
