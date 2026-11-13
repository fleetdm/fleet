package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	ds_mock "github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

func TestBuildFleetDefaultPolicy(t *testing.T) {
	profile := func(name, rawJSON string) *fleet.MDMAndroidConfigProfile {
		return &fleet.MDMAndroidConfigProfile{ProfileUUID: "g-" + name, Name: name, RawJSON: []byte(rawJSON)}
	}
	// settings returns the policy's JSON without the status reporting settings every policy gets.
	settings := func(t *testing.T, policy *androidmanagement.Policy) map[string]any {
		require.NotNil(t, policy.StatusReportingSettings)
		b, err := json.Marshal(policy)
		require.NoError(t, err)
		var got map[string]any
		require.NoError(t, json.Unmarshal(b, &got))
		delete(got, "statusReportingSettings")
		return got
	}
	decode := func(t *testing.T, s string) map[string]any {
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(s), &m))
		return m
	}

	cases := []struct {
		name        string
		profiles    []*fleet.MDMAndroidConfigProfile
		want        string
		wantSkipped []string
	}{
		{
			name: "no profiles",
			want: `{}`,
		},
		{
			name: "only passcode keys are copied",
			profiles: []*fleet.MDMAndroidConfigProfile{
				profile("pin", `{"passwordPolicies": [{"passwordMinimumLength": 6}], "cameraDisabled": true}`),
			},
			want: `{"passwordPolicies": [{"passwordMinimumLength": 6}]}`,
		},
		{
			name: "the name that sorts last wins a key",
			profiles: []*fleet.MDMAndroidConfigProfile{
				profile("b", `{"passwordPolicies": [{"passwordMinimumLength": 8}]}`),
				profile("a", `{"passwordPolicies": [{"passwordMinimumLength": 4}]}`),
			},
			want: `{"passwordPolicies": [{"passwordMinimumLength": 8}]}`,
		},
		{
			name: "passwordPolicies and passwordRequirements from different profiles are both copied",
			profiles: []*fleet.MDMAndroidConfigProfile{
				profile("a", `{"passwordRequirements": {"passwordMinimumLength": 4}}`),
				profile("b", `{"passwordPolicies": [{"passwordMinimumLength": 8}]}`),
			},
			want: `{"passwordRequirements": {"passwordMinimumLength": 4}, "passwordPolicies": [{"passwordMinimumLength": 8}]}`,
		},
		{
			name: "enforcement rules from the winning passcode profile",
			profiles: []*fleet.MDMAndroidConfigProfile{
				profile("pin", `{"passwordPolicies": [{"passwordMinimumLength": 6}], "policyEnforcementRules": [{"settingName": "passwordPolicies", "blockAction": {"blockAfterDays": 1}}]}`),
			},
			want: `{"passwordPolicies": [{"passwordMinimumLength": 6}], "policyEnforcementRules": [{"settingName": "passwordPolicies", "blockAction": {"blockAfterDays": 1}}]}`,
		},
		{
			name: "enforcement rules won by a non-passcode profile are left out",
			profiles: []*fleet.MDMAndroidConfigProfile{
				profile("a pin", `{"passwordPolicies": [{"passwordMinimumLength": 6}], "policyEnforcementRules": [{"settingName": "passwordPolicies"}]}`),
				profile("b rules", `{"policyEnforcementRules": [{"settingName": "cameraDisabled"}]}`),
			},
			want: `{"passwordPolicies": [{"passwordMinimumLength": 6}]}`,
		},
		{
			name: "a non-passcode profile's rules lose to a later passcode profile",
			profiles: []*fleet.MDMAndroidConfigProfile{
				profile("a rules", `{"policyEnforcementRules": [{"settingName": "cameraDisabled"}]}`),
				profile("b pin", `{"passwordPolicies": [{"passwordMinimumLength": 6}], "policyEnforcementRules": [{"settingName": "passwordPolicies"}]}`),
			},
			want: `{"passwordPolicies": [{"passwordMinimumLength": 6}], "policyEnforcementRules": [{"settingName": "passwordPolicies"}]}`,
		},
		{
			name: "profiles with variables are skipped",
			profiles: []*fleet.MDMAndroidConfigProfile{
				profile("a", `{"passwordPolicies": [{"passwordMinimumLength": 4}]}`),
				profile("b vars", `{"passwordPolicies": [{"passwordMinimumLength": 8}], "name": "$FLEET_VAR_HOST_UUID"}`),
			},
			want:        `{"passwordPolicies": [{"passwordMinimumLength": 4}]}`,
			wantSkipped: []string{"b vars"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			policy, skipped, err := buildFleetDefaultPolicy(c.profiles)
			require.NoError(t, err)
			assert.Equal(t, decode(t, c.want), settings(t, policy))
			assert.Equal(t, c.wantSkipped, skipped)
		})
	}

	t.Run("checksum is stable and follows the content", func(t *testing.T) {
		pinFour := []*fleet.MDMAndroidConfigProfile{profile("a", `{"passwordPolicies": [{"passwordMinimumLength": 4}]}`)}
		pinSix := []*fleet.MDMAndroidConfigProfile{profile("a", `{"passwordPolicies": [{"passwordMinimumLength": 6}]}`)}
		checksum := func(profiles []*fleet.MDMAndroidConfigProfile) []byte {
			policy, _, err := buildFleetDefaultPolicy(profiles)
			require.NoError(t, err)
			sum, err := fleetDefaultPolicyChecksum(policy)
			require.NoError(t, err)
			return sum
		}
		first, second := checksum(pinFour), checksum(pinFour)
		assert.Equal(t, first, second)
		assert.NotEqual(t, first, checksum(pinSix))
		assert.NotEqual(t, checksum(nil), checksum(pinFour))
	})
}

func TestReconcileFleetDefaultPolicies(t *testing.T) {
	const enterpriseName = "enterprises/LC00test"
	teamA, teamB, teamGone := uint(3), uint(5), uint(9)
	pin := func(teamID uint, minLength int) *fleet.MDMAndroidConfigProfile {
		raw, err := json.Marshal(map[string]any{"passwordPolicies": []map[string]any{{"passwordMinimumLength": minLength}}})
		require.NoError(t, err)
		return &fleet.MDMAndroidConfigProfile{Name: "pin", TeamID: &teamID, RawJSON: raw}
	}
	checksumFor := func(profiles ...*fleet.MDMAndroidConfigProfile) []byte {
		policy, _, err := buildFleetDefaultPolicy(profiles)
		require.NoError(t, err)
		sum, err := fleetDefaultPolicyChecksum(policy)
		require.NoError(t, err)
		return sum
	}

	ds := new(ds_mock.Store)
	client := &android_mock.Client{}
	reconciler := &profileReconciler{
		DS:         ds,
		Enterprise: &android.Enterprise{EnterpriseID: "LC00test"},
		Client:     client,
		Logger:     slog.New(slog.DiscardHandler),
	}

	var (
		profiles  []*fleet.MDMAndroidConfigProfile
		checksums map[uint][]byte
		patched        map[string]*androidmanagement.Policy
		patchCount     int
		saved          map[uint][]byte
		savedPrevious  map[uint][]byte
		patchErrs      map[string]error
		rejectPasscode bool
	)
	ds.ListAndroidFleetDefaultPolicyProfilesFunc = func(_ context.Context, teamID *uint) ([]*fleet.MDMAndroidConfigProfile, error) {
		require.Nil(t, teamID)
		return profiles, nil
	}
	ds.GetAndroidFleetDefaultPolicyChecksumsFunc = func(_ context.Context) (map[uint][]byte, error) {
		return checksums, nil
	}
	ds.SetAndroidFleetDefaultPolicyChecksumFunc = func(_ context.Context, teamID uint, checksum, previous []byte) error {
		saved[teamID] = checksum
		savedPrevious[teamID] = previous
		return nil
	}
	client.EnterprisesPoliciesPatchFunc = func(_ context.Context, policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
		require.True(t, opts.ExcludeApps)
		patchCount++
		patched[policyName] = policy
		if rejectPasscode && policy.PasswordPolicies != nil {
			return nil, &googleapi.Error{Code: http.StatusBadRequest}
		}
		return policy, patchErrs[policyName]
	}
	run := func(t *testing.T) {
		patched, saved, savedPrevious, patchCount = map[string]*androidmanagement.Policy{}, map[uint][]byte{}, map[uint][]byte{}, 0
		require.NoError(t, reconciler.reconcileFleetDefaultPolicies(t.Context()))
	}
	policyName := func(teamID uint) string { return enterpriseName + "/policies/" + android.FleetDefaultPolicyID(teamID) }

	t.Run("first run creates the policies of fleets with passcode profiles", func(t *testing.T) {
		profiles = []*fleet.MDMAndroidConfigProfile{pin(0, 4), pin(teamA, 6)}
		checksums, patchErrs = nil, nil
		run(t)
		require.Len(t, patched, 2)
		require.Equal(t, int64(4), patched[policyName(0)].PasswordPolicies[0].PasswordMinimumLength)
		require.Equal(t, int64(6), patched[policyName(teamA)].PasswordPolicies[0].PasswordMinimumLength)
		require.Equal(t, map[uint][]byte{0: checksumFor(pin(0, 4)), teamA: checksumFor(pin(teamA, 6))}, saved)
	})

	t.Run("unchanged policies are not sent again", func(t *testing.T) {
		checksums = map[uint][]byte{0: checksumFor(pin(0, 4)), teamA: checksumFor(pin(teamA, 6))}
		run(t)
		require.Empty(t, patched)
		require.Empty(t, saved)
	})

	t.Run("only the changed fleet is sent", func(t *testing.T) {
		profiles = []*fleet.MDMAndroidConfigProfile{pin(0, 4), pin(teamA, 8)}
		run(t)
		require.Len(t, patched, 1)
		require.Equal(t, int64(8), patched[policyName(teamA)].PasswordPolicies[0].PasswordMinimumLength)
		require.Equal(t, map[uint][]byte{teamA: checksumFor(pin(teamA, 8))}, saved)
		require.Equal(t, checksumFor(pin(teamA, 6)), savedPrevious[teamA], "the checksum is only replaced if it's still the one read")
	})

	t.Run("a policy marked unverified is sent again", func(t *testing.T) {
		checksums = map[uint][]byte{0: checksumFor(pin(0, 4)), teamA: make([]byte, 16)}
		run(t)
		require.Len(t, patched, 1)
		require.Equal(t, int64(8), patched[policyName(teamA)].PasswordPolicies[0].PasswordMinimumLength)
		require.Equal(t, map[uint][]byte{teamA: checksumFor(pin(teamA, 8))}, saved)
		require.Equal(t, make([]byte, 16), savedPrevious[teamA])
	})

	t.Run("a fleet whose passcode profiles are gone goes back to the base settings", func(t *testing.T) {
		profiles = []*fleet.MDMAndroidConfigProfile{pin(0, 4)}
		checksums = map[uint][]byte{0: checksumFor(pin(0, 4)), teamA: checksumFor(pin(teamA, 8))}
		run(t)
		require.Len(t, patched, 1)
		require.Empty(t, patched[policyName(teamA)].PasswordPolicies)
		require.NotNil(t, patched[policyName(teamA)].StatusReportingSettings)
		require.Equal(t, map[uint][]byte{teamA: checksumFor()}, saved)
	})

	t.Run("a fleet with a base policy and no passcode profiles is left alone", func(t *testing.T) {
		checksums = map[uint][]byte{0: checksumFor(pin(0, 4)), teamB: checksumFor()}
		run(t)
		require.Empty(t, patched)
	})

	t.Run("an AMAPI failure for one fleet doesn't stop the others and isn't saved", func(t *testing.T) {
		profiles = []*fleet.MDMAndroidConfigProfile{pin(teamA, 6), pin(teamB, 7)}
		checksums = nil
		patchErrs = map[string]error{policyName(teamA): errors.New("quota")}
		run(t)
		require.Len(t, patched, 2)
		require.Equal(t, map[uint][]byte{teamB: checksumFor(pin(teamB, 7))}, saved)
	})

	t.Run("not modified counts as sent", func(t *testing.T) {
		profiles = []*fleet.MDMAndroidConfigProfile{pin(teamGone, 6)}
		checksums = nil
		patchErrs = map[string]error{policyName(teamGone): &googleapi.Error{Code: http.StatusNotModified}}
		run(t)
		require.Equal(t, map[uint][]byte{teamGone: checksumFor(pin(teamGone, 6))}, saved)
		require.Nil(t, savedPrevious[teamGone], "a fleet without a row is only inserted if still absent")
	})

	t.Run("rejected passcode settings are replaced by the base settings and not retried", func(t *testing.T) {
		profiles = []*fleet.MDMAndroidConfigProfile{pin(teamA, 6)}
		checksums, patchErrs, rejectPasscode = nil, nil, true
		t.Cleanup(func() { rejectPasscode = false })
		run(t)
		require.Equal(t, 2, patchCount)
		require.Empty(t, patched[policyName(teamA)].PasswordPolicies, "the last patch is the base policy")
		require.NotNil(t, patched[policyName(teamA)].StatusReportingSettings)
		require.Equal(t, map[uint][]byte{teamA: checksumFor(pin(teamA, 6))}, saved)

		checksums = saved
		run(t)
		require.Zero(t, patchCount, "nothing is sent again until the profiles change")
	})
}
