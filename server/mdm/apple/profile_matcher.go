package apple_mdm

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type profileMatcher struct {
	pool fleet.RedisPool
}

func NewProfileMatcher(pool fleet.RedisPool) *profileMatcher {
	return &profileMatcher{pool: pool}
}

func (p *profileMatcher) PreassignProfile(payload fleet.MDMApplePreassignProfilePayload) error {
	conn := p.pool.Get()
	defer conn.Close()
	// TODO: store the profile keyed by payload.ExternalHostIdentifier
	return nil
}

func (p *profileMatcher) RetrieveProfiles(hostReference string) error {
	conn := p.pool.Get()
	defer conn.Close()
	// TODO: find all profiles matching the reference
	// TODO: cleanup all retrieved profiles?
	return nil
}
