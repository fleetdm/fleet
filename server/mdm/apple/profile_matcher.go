package apple_mdm

import (
	"context"
	"encoding/hex"
	"strings"
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
	// in distributed scenarios, the external host identifier might come up
	// with a suffix that varies from server to server. To account for that
	// we only take into account the first 36 runes from the identifier.
	// See https://github.com/fleetdm/fleet/issues/12483 for more info.
	preassignKeySuffixMaxLen = 36
)

type profileMatcher struct {
	pool fleet.RedisPool
}

// NewProfileMatcher creates a new MDM profile matcher based on Redis.
func NewProfileMatcher(pool fleet.RedisPool) fleet.ProfileMatcher {
	return &profileMatcher{pool: pool}
}

// PreassignProfile stores the profile associated with the host in Redis for
// later retrieval and matching to a team. Note that to keep this logic fast,
// we avoid accessing the mysql database, so the host is not validated at this
// stage (i.e. checking that it is valid, enrolled in MDM, etc.). It is done in
// the matching stage.
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
	} else {
		// team ID is not relevant at this stage, this is just for validation
		if cp, err := fleet.NewMDMAppleConfigProfile(payload.Profile, nil); err != nil {
			invArg.Append("profile", err.Error())
		} else if err := cp.ValidateUserProvided(false); err != nil {
			invArg.Append("profile", err.Error())
		}
	}
	if invArg.HasErrors() {
		return ctxerr.Wrap(ctx, invArg)
	}

	md5Hash := payload.HexMD5Hash()

	// 2 fields set if the top-level Redis hash key was newly created: host uuid
	// and profile. If a group is provided, then it's 3 fields.
	expectOnCreate := 3
	args := []any{
		// key is the prefix + the external identifier, all of this host's profiles
		// will be stored under that hash, keyed by the md5-hash.
		keyForExternalHostIdentifier(payload.ExternalHostIdentifier),

		// the host uuid must be stored (cannot clash with other fields as they are
		// hex-encoded hashes), will be a no-op if it was already stored (i.e. not
		// the first profile for this host).
		"host_uuid", payload.HostUUID,

		// the profile itself is stored under its md5-hash field, no-op if it
		// already existed.
		md5Hash, payload.Profile,

		md5Hash + "_exclude", payload.Exclude,
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
func (p *profileMatcher) RetrieveProfiles(ctx context.Context, externalHostIdentifier string) (fleet.MDMApplePreassignHostProfiles, error) {
	var hostProfs fleet.MDMApplePreassignHostProfiles

	// Note that we do not configure the Redis connection to read from a replica
	// here as it may be called very soon after the PreassignProfile call and
	// could miss some unreplicated profiles.
	conn := redis.ConfigureDoer(p.pool, p.pool.Get())
	defer conn.Close()

	profs, err := redigo.StringMap(conn.Do("HGETALL", keyForExternalHostIdentifier(externalHostIdentifier)))
	if err != nil {
		return hostProfs, ctxerr.Wrap(ctx, err, "execute redis HGETALL")
	}
	if _, err := conn.Do("UNLINK", keyForExternalHostIdentifier(externalHostIdentifier)); err != nil {
		return hostProfs, ctxerr.Wrap(ctx, err, "execute redis UNLINK")
	}
	_ = conn.Close() // release connection to the pool immediately as we're done with redis

	if hostProfs.HostUUID = profs["host_uuid"]; hostProfs.HostUUID == "" {
		// unknown host/no profiles to assign, not an error but nothing to do
		return hostProfs, nil
	}
	delete(profs, "host_uuid")

	for k, v := range profs {
		if strings.HasSuffix(k, "_group") || v == "" || strings.HasSuffix(k, "_exclude") {
			// only look for profiles' hex hashes, the group information will be
			// retrieved only when a profile is found. Ignore empty values (e.g.
			// empty profile).
			continue
		}

		// if the key is not the group, then it has to be a profile hash, ensure
		// that it is a valid hex-encoded value.
		if _, err := hex.DecodeString(k); err != nil {
			// ignore unknown/invalid fields
			continue
		}

		hostProfs.Profiles = append(hostProfs.Profiles, fleet.MDMApplePreassignProfile{
			Profile:    []byte(v),
			Group:      profs[k+"_group"],
			HexMD5Hash: k,
			Exclude:    profs[k+"_exclude"] == "1",
		})
	}
	return hostProfs, nil
}

func keyForExternalHostIdentifier(externalHostIdentifier string) string {
	return preassignKeyPrefix + firstNRunes(externalHostIdentifier, preassignKeySuffixMaxLen)
}

// firstNRunes grabs the first N runes from the provided string
func firstNRunes(s string, n int) string {
	i := 0
	for j := range s {
		if i == n {
			return s[:j]
		}
		i++
	}
	return s
}
