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
