package apple_mdm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

func TestPreassignProfile(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		ctx := context.Background()
		matcher := NewProfileMatcher(pool)

		// preassign a profile
		p1 := fleet.MDMApplePreassignProfilePayload{
			ExternalHostIdentifier: "abcd",
			HostUUID:               "1234",
			Profile:                generateProfile("p1", "p1", "Configuration", "p1"),
			Group:                  "g1",
		}
		err := matcher.PreassignProfile(ctx, p1)
		require.NoError(t, err)

		// should've set a TTL on the key
		conn := redis.ConfigureDoer(pool, pool.Get())
		defer conn.Close()

		ttl1, err := redigo.Int(conn.Do("TTL", keyForExternalHostIdentifier("abcd")))
		require.NoError(t, err)
		require.NotZero(t, ttl1)
		require.LessOrEqual(t, ttl1, int(preassignKeyExpiration.Seconds()))

		// sleep a second to see the existing ttl go down
		time.Sleep(time.Second)

		// preassign another profile on the same host
		p2 := fleet.MDMApplePreassignProfilePayload{
			ExternalHostIdentifier: "abcd",
			HostUUID:               "1234",
			Profile:                generateProfile("p2", "p2", "Configuration", "p2"),
			Group:                  "g2",
		}
		err = matcher.PreassignProfile(ctx, p2)
		require.NoError(t, err)

		// key already existed, so it did not reset the ttl
		ttl2, err := redigo.Int(conn.Do("TTL", keyForExternalHostIdentifier("abcd")))
		require.NoError(t, err)
		require.NotZero(t, ttl2)
		require.Less(t, ttl2, ttl1)

		// preassign another profile on the same host, without a group
		p3 := fleet.MDMApplePreassignProfilePayload{
			ExternalHostIdentifier: "abcd",
			HostUUID:               "1234",
			Profile:                generateProfile("p3", "p3", "Configuration", "p3"),
			Exclude:                true,
		}
		err = matcher.PreassignProfile(ctx, p3)
		require.NoError(t, err)

		// key already existed, so it did not reset the ttl
		ttl3, err := redigo.Int(conn.Do("TTL", keyForExternalHostIdentifier("abcd")))
		require.NoError(t, err)
		require.NotZero(t, ttl3)
		require.Less(t, ttl3, ttl1)

		// preassign the same profile on the same host, no change
		err = matcher.PreassignProfile(ctx, p3)
		require.NoError(t, err)

		// key already existed, so it did not reset the ttl
		ttl4, err := redigo.Int(conn.Do("TTL", keyForExternalHostIdentifier("abcd")))
		require.NoError(t, err)
		require.NotZero(t, ttl4)
		require.Less(t, ttl4, ttl1)

		// preassign a profile on a different host
		p4 := fleet.MDMApplePreassignProfilePayload{
			ExternalHostIdentifier: "efgh",
			HostUUID:               "5678",
			Profile:                generateProfile("p4", "p4", "Configuration", "p4"),
			Group:                  "g4",
		}
		err = matcher.PreassignProfile(ctx, p4)
		require.NoError(t, err)

		// original host's key unchanged
		ttl5, err := redigo.Int(conn.Do("TTL", keyForExternalHostIdentifier("abcd")))
		require.NoError(t, err)
		require.NotZero(t, ttl5)
		require.Less(t, ttl5, ttl1)

		// new host's ttl is set
		ttl1, err = redigo.Int(conn.Do("TTL", keyForExternalHostIdentifier("efgh")))
		require.NoError(t, err)
		require.NotZero(t, ttl1)
		require.LessOrEqual(t, ttl1, int(preassignKeyExpiration.Seconds()))

		// stored 3 profiles in original host
		profs, err := redigo.StringMap(conn.Do("HGETALL", keyForExternalHostIdentifier("abcd")))
		require.NoError(t, err)
		require.Equal(t, map[string]string{
			"host_uuid":                  "1234",
			p1.HexMD5Hash():              string(p1.Profile),
			p1.HexMD5Hash() + "_group":   "g1",
			p1.HexMD5Hash() + "_exclude": "0",
			p2.HexMD5Hash():              string(p2.Profile),
			p2.HexMD5Hash() + "_group":   "g2",
			p2.HexMD5Hash() + "_exclude": "0",
			p3.HexMD5Hash():              string(p3.Profile),
			p3.HexMD5Hash() + "_exclude": "1",
		}, profs)

		// stored 1 profile in new host
		profs, err = redigo.StringMap(conn.Do("HGETALL", keyForExternalHostIdentifier("efgh")))
		require.NoError(t, err)
		require.Equal(t, map[string]string{
			"host_uuid":                  "5678",
			p4.HexMD5Hash():              string(p4.Profile),
			p4.HexMD5Hash() + "_group":   "g4",
			p4.HexMD5Hash() + "_exclude": "0",
		}, profs)
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, preassignKeyPrefix, false, false, false)
		runTest(t, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, preassignKeyPrefix, true, true, false)
		runTest(t, pool)
	})
}

func TestRetrieveProfiles(t *testing.T) {
	runTest := func(t *testing.T, pool fleet.RedisPool) {
		ctx := context.Background()
		matcher := NewProfileMatcher(pool)

		// preassign a profile with a group
		p1 := fleet.MDMApplePreassignProfilePayload{
			ExternalHostIdentifier: "abcd",
			HostUUID:               "1234",
			Profile:                generateProfile("p1", "p1", "Configuration", "p1"),
			Group:                  "g1",
		}
		err := matcher.PreassignProfile(ctx, p1)
		require.NoError(t, err)

		// preassign a profile without a group
		p2 := fleet.MDMApplePreassignProfilePayload{
			ExternalHostIdentifier: "abcd",
			HostUUID:               "1234",
			Profile:                generateProfile("p2", "p2", "Configuration", "p2"),
		}
		err = matcher.PreassignProfile(ctx, p2)
		require.NoError(t, err)

		// retrieve from unknown external host identifier
		profs, err := matcher.RetrieveProfiles(ctx, "efgh")
		require.NoError(t, err)
		require.Empty(t, profs.HostUUID)
		require.Empty(t, profs.Profiles)

		// retrieve from valid external host identifier
		profs, err = matcher.RetrieveProfiles(ctx, "abcd")
		require.NoError(t, err)
		require.Equal(t, "1234", profs.HostUUID)
		require.ElementsMatch(t, []fleet.MDMApplePreassignProfile{
			{Profile: p1.Profile, Group: p1.Group, HexMD5Hash: p1.HexMD5Hash()},
			{Profile: p2.Profile, Group: "", HexMD5Hash: p2.HexMD5Hash()},
		}, profs.Profiles)

		// after retrieval, the key is deleted
		profs, err = matcher.RetrieveProfiles(ctx, "abcd")
		require.NoError(t, err)
		require.Empty(t, profs.HostUUID)
		require.Empty(t, profs.Profiles)

		// preassign to a host and generate invalid data
		p3 := fleet.MDMApplePreassignProfilePayload{
			ExternalHostIdentifier: "xyz",
			HostUUID:               "5678",
			Profile:                generateProfile("p3", "p3", "Configuration", "p3"),
		}
		err = matcher.PreassignProfile(ctx, p3)
		require.NoError(t, err)

		conn := redis.ConfigureDoer(pool, pool.Get())
		defer conn.Close()
		_, err = conn.Do("HSET", keyForExternalHostIdentifier("xyz"), "123ABC", "", "not-hex", "foo")
		require.NoError(t, err)

		// retrieves only the valid data for that host
		profs, err = matcher.RetrieveProfiles(ctx, "xyz")
		require.NoError(t, err)
		require.Equal(t, "5678", profs.HostUUID)
		require.ElementsMatch(t, []fleet.MDMApplePreassignProfile{
			{Profile: p3.Profile, Group: "", HexMD5Hash: p3.HexMD5Hash()},
		}, profs.Profiles)
	}

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, preassignKeyPrefix, false, false, false)
		runTest(t, pool)
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, preassignKeyPrefix, true, true, false)
		runTest(t, pool)
	})
}

func TestPreassignProfileValidation(t *testing.T) {
	ctx := context.Background()
	pool := redistest.SetupRedis(t, preassignKeyPrefix, false, false, false)
	matcher := NewProfileMatcher(pool)

	cases := []struct {
		desc    string
		payload fleet.MDMApplePreassignProfilePayload
		err     string
	}{
		{
			"empty external host identifier",
			fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "",
				HostUUID:               "1234",
				Profile:                generateProfile("p1", "p1", "Configuration", "p1"),
			},
			"external_host_identifier required",
		},
		{
			"empty host uuid",
			fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "abcd",
				HostUUID:               "",
				Profile:                generateProfile("p1", "p1", "Configuration", "p1"),
			},
			"host_uuid required",
		},
		{
			"empty profile",
			fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "abcd",
				HostUUID:               "1234",
				Profile:                nil,
			},
			"profile required",
		},
		{
			"invalid profile",
			fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "abcd",
				HostUUID:               "1234",
				Profile:                []byte(`abcd`),
			},
			"signed profiles are not supported",
		},
		{
			"invalid profile type",
			fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "abcd",
				HostUUID:               "1234",
				Profile:                generateProfile("p1", "p1", "abcd", "p1"),
			},
			"invalid PayloadType: abcd",
		},
		{
			"empty payload identifier",
			fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "abcd",
				HostUUID:               "1234",
				Profile:                generateProfile("p1", "", "Configuration", "p1"),
			},
			"empty PayloadIdentifier in profile",
		},
		{
			"empty payload name",
			fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "abcd",
				HostUUID:               "1234",
				Profile:                generateProfile("", "p1", "Configuration", "p1"),
			},
			"empty PayloadDisplayName in profile",
		},
		{
			"invalid payload identifier",
			fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "abcd",
				HostUUID:               "1234",
				Profile:                generateProfile("p1", mobileconfig.FleetFileVaultPayloadIdentifier, "Configuration", "p1"),
			},
			"payload identifier com.fleetdm.fleet.mdm.filevault is not allowed",
		},
		{
			"invalid payload name",
			fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "abcd",
				HostUUID:               "1234",
				Profile:                generateProfile(mdm.FleetFileVaultProfileName, "p1", "Configuration", "p1"),
			},
			"payload display name Disk encryption is not allowed",
		},
		{
			"valid",
			fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: "abcd",
				HostUUID:               "1234",
				Profile:                generateProfile("p1", "p1", "Configuration", "p1"),
				Group:                  "g1",
			},
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := matcher.PreassignProfile(ctx, c.payload)
			if c.err != "" {
				require.ErrorContains(t, err, c.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestKeyForExternalHostIdentifier(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"abcd", "abcd"},
		{"6f36ab2c-1a40-429b-9c9d-07c9029f4aa8", "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8"},
		{"6f36ab2c-1a40-429b-9c9d-07c9029f4aa8-puppetcompiler06.test.example.com", "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8"},
	}

	for _, c := range cases {
		got := keyForExternalHostIdentifier(c.in)
		require.Equal(t, preassignKeyPrefix+c.want, got)
	}
}

func generateProfile(name, ident, typ, uuid string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>%s</string>
	<key>PayloadIdentifier</key>
	<string>%s</string>
	<key>PayloadType</key>
	<string>%s</string>
	<key>PayloadUUID</key>
	<string>%s</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`, name, ident, typ, uuid))
}
