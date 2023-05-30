package apple_mdm

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type profileMatcher struct {
	pool fleet.RedisPool
}

// NewProfileMatcher creates a new MDM profile matcher based on Redis.
func NewProfileMatcher(pool fleet.RedisPool) fleet.ProfileMatcher {
	return &profileMatcher{pool: pool}
}

// PreassignProfile stores the profile associated with the host in Redis for
// later retrieval and matching to a team.
func (p *profileMatcher) PreassignProfile(ctx context.Context, payload fleet.MDMApplePreassignProfilePayload) error {
	conn := p.pool.Get()
	defer conn.Close()
	// TODO: store the profile keyed by payload.ExternalHostIdentifier
	return nil
}

// RetrieveProfiles retrieves the profiles preassigned to this host for
// matching with a team and assignment.
func (p *profileMatcher) RetrieveProfiles(ctx context.Context, externalHostIdentifier string) error {
	conn := p.pool.Get()
	defer conn.Close()
	// TODO: find all profiles matching the host identifier
	// TODO: cleanup all retrieved profiles?
	return nil
}
