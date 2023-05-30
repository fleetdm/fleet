package apple_mdm

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

const (
	preassignKeyPrefix = "mdm:preassign:"

	// must be reasonably longer than the expected time to make all
	// PreassignProfile calls and the final matcher call.
	preassignKeyExpiration = 1 * time.Hour
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
	var invArg fleet.InvalidArgumentError
	if payload.ExternalHostIdentifier == "" {
		invArg.Append("external_host_identifier", "required")
	}
	if payload.HostUUID == "" {
		invArg.Append("host_uuid", "required")
	}
	if len(payload.Profile) == 0 {
		invArg.Append("profile", "required")
	}

	// team ID is not relevant at this stage, this is just for validation
	if cp, err := fleet.NewMDMAppleConfigProfile(payload.Profile, nil); err != nil {
		invArg.Append("profile", err.Error())
	} else if err := cp.ValidateUserProvided(); err != nil {
		invArg.Append("profile", err.Error())
	}
	if invArg.HasErrors() {
		return ctxerr.Wrap(ctx, invArg)
	}

	md5Hash := payload.HexMD5Hash()

	// 2 fields set if the top-level Redis hash key was newly created: host uuid
	// and profile. If a group is provided, then it's 3 fields.
	expectOnCreate := 2
	args := []any{
		// key is the prefix + the external identifier, all of this host's profiles
		// will be stored under that hash, keyed by the md5-hash.
		preassignKeyPrefix + payload.ExternalHostIdentifier,

		// the host uuid must be stored (cannot clash with other fields as they are
		// hex-encoded hashes), will be a no-op if it was already stored (i.e. not
		// the first profile for this host).
		"host_uuid", payload.HostUUID,

		// the profile itself is stored under its md5-hash field, no-op if it
		// already existed.
		md5Hash, payload.Profile,
	}
	if payload.Group != "" {
		args = append(args, md5Hash+"_group", payload.Group)
		expectOnCreate++
	}

	conn := redis.ConfigureDoer(p.pool, p.pool.Get())
	defer conn.Close()

	res, err := redigo.Int(conn.Do("HSET", args...))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "execute redis HSET")
	}
	if res >= expectOnCreate {
		// the key was created, set a TTL
		if _, err := conn.Do("EXPIRE", args[0], preassignKeyExpiration.Seconds()); err != nil {
			return ctxerr.Wrap(ctx, err, "execute redis EXPIRE")
		}
	}
	return nil
}

// RetrieveProfiles retrieves the profiles preassigned to this host for
// matching with a team and assignment.
func (p *profileMatcher) RetrieveProfiles(ctx context.Context, externalHostIdentifier string) error {
	// Note that we do not configure the Redis connection to read from a replica
	// here as it may be called very soon after the PreassignProfile call and
	// could miss some unreplicated profiles.
	conn := redis.ConfigureDoer(p.pool, p.pool.Get())
	defer conn.Close()
	// TODO: find all profiles matching the host identifier
	// TODO: cleanup all retrieved profiles?
	return nil
}
