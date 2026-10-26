package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestApplyProfilesBatchErrorNamesProfile(t *testing.T) {
	t.Parallel()

	// 422 response as returned by POST /api/latest/fleet/mdm/profiles/batch when
	// one profile in the batch fails validation.
	newClient := func(t *testing.T, body map[string]any) *Client {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(body)
		}))
		t.Cleanup(srv.Close)
		client, err := NewClient(srv.URL, true, "", "")
		require.NoError(t, err)
		client.SetToken("test-token")
		return client
	}

	profiles := []fleet.MDMProfileBatchPayload{
		{Name: "screen-lock", Contents: []byte("<Replace/>")},
		{Name: "bitlocker", Contents: []byte("<Replace/>")},
	}
	invalidBody := map[string]any{
		"message": "Validation Failed",
		"errors":  []map[string]string{{"name": "profiles[bitlocker]", "reason": "Couldn't add. The configuration profile can't include BitLocker settings."}},
	}

	t.Run("team profiles name the failing profile", func(t *testing.T) {
		client := newClient(t, invalidBody)
		err := client.ApplyTeamProfiles("Workstations", profiles, fleet.ApplyTeamSpecOptions{})
		require.ErrorContains(t, err, `profile "bitlocker"`)
		require.ErrorContains(t, err, "can't include BitLocker settings")
		// the original error stays in the chain for status-code checks
		var scErr *StatusCodeErr
		require.ErrorAs(t, err, &scErr)
		require.Equal(t, http.StatusUnprocessableEntity, scErr.Code)
	})

	t.Run("no-team profiles name the failing profile", func(t *testing.T) {
		client := newClient(t, invalidBody)
		err := client.ApplyNoTeamProfiles(profiles, fleet.ApplySpecOptions{}, nil)
		require.ErrorContains(t, err, `profile "bitlocker"`)
	})

	t.Run("errors not tied to a profile pass through unchanged", func(t *testing.T) {
		client := newClient(t, map[string]any{
			"message": "Validation Failed",
			"errors":  []map[string]string{{"name": "mdm", "reason": "cannot set custom settings: Windows MDM is not configured"}},
		})
		err := client.ApplyNoTeamProfiles(profiles, fleet.ApplySpecOptions{}, nil)
		require.Error(t, err)
		require.NotContains(t, err.Error(), `profile "`)
		require.ErrorContains(t, err, "Windows MDM is not configured")
	})
}

func TestRewrapProfileBatchNameErr(t *testing.T) {
	t.Parallel()

	profiles := []fleet.MDMProfileBatchPayload{{Name: "screen-lock"}, {Name: "Passcode policy"}}

	statusErr := func(name string) error {
		return &StatusCodeErr{Code: http.StatusUnprocessableEntity, Body: "Validation Failed: boom", Name: name}
	}

	require.NoError(t, rewrapProfileBatchNameErr(nil, profiles))

	plain := errors.New("boom")
	require.Equal(t, plain, rewrapProfileBatchNameErr(plain, profiles))

	cases := []struct {
		desc     string
		name     string
		wantWrap string // empty means unchanged
	}{
		{"windows/android style", "profiles[screen-lock]", `profile "screen-lock"`},
		{"apple style bare name", "Passcode policy", `profile "Passcode policy"`},
		{"bare name with spaces in brackets", "profiles[Passcode policy]", `profile "Passcode policy"`},
		{"no name", "", ""},
		{"endpoint field", "mdm", ""},
		{"whole-list field", "profiles", ""},
		{"unknown profile", "profiles[other]", ""},
		{"missing closing bracket", "profiles[screen-lock", ""},
		{"platform sub-list index", "windowsProfiles[0]", ""},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			in := statusErr(c.name)
			out := rewrapProfileBatchNameErr(in, profiles)
			if c.wantWrap == "" {
				require.Equal(t, in, out)
			} else {
				require.ErrorContains(t, out, c.wantWrap)
				require.ErrorIs(t, out, in)
			}
		})
	}
}

func TestApplyProfilesSendsDryRunAssumptions(t *testing.T) {
	t.Parallel()

	newClient := func(t *testing.T, query *url.Values) *Client {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*query = r.URL.Query()
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(srv.Close)
		client, err := NewClient(srv.URL, true, "", "")
		require.NoError(t, err)
		client.SetToken("test-token")
		return client
	}
	profiles := []fleet.MDMProfileBatchPayload{{Name: "p", Contents: []byte("<Replace/>")}}

	for _, v := range []bool{true, false} {
		assumptions := &fleet.TeamSpecsDryRunAssumptions{EnableProfileValidation: optjson.SetBool(v)}
		want := strconv.FormatBool(v)

		t.Run("team "+want, func(t *testing.T) {
			var q url.Values
			err := newClient(t, &q).ApplyTeamProfiles("Workstations", profiles,
				fleet.ApplyTeamSpecOptions{ApplySpecOptions: fleet.ApplySpecOptions{DryRun: true}, DryRunAssumptions: assumptions})
			require.NoError(t, err)
			require.Equal(t, want, q.Get("assume_profile_validation"))
		})

		t.Run("no team "+want, func(t *testing.T) {
			var q url.Values
			err := newClient(t, &q).ApplyNoTeamProfiles(profiles, fleet.ApplySpecOptions{DryRun: true}, assumptions)
			require.NoError(t, err)
			require.Equal(t, want, q.Get("assume_profile_validation"))
		})
	}

	t.Run("not sent without assumption", func(t *testing.T) {
		var q url.Values
		require.NoError(t, newClient(t, &q).ApplyNoTeamProfiles(profiles, fleet.ApplySpecOptions{}, nil))
		require.False(t, q.Has("assume_profile_validation"))
		require.NoError(t, newClient(t, &q).ApplyTeamProfiles("Workstations", profiles, fleet.ApplyTeamSpecOptions{}))
		require.False(t, q.Has("assume_profile_validation"))
	})

	t.Run("no team keeps assume_enabled only when true", func(t *testing.T) {
		var q url.Values
		err := newClient(t, &q).ApplyNoTeamProfiles(profiles, fleet.ApplySpecOptions{},
			&fleet.TeamSpecsDryRunAssumptions{WindowsEnabledAndConfigured: optjson.SetBool(true)})
		require.NoError(t, err)
		require.Equal(t, "true", q.Get("assume_enabled"))

		err = newClient(t, &q).ApplyNoTeamProfiles(profiles, fleet.ApplySpecOptions{},
			&fleet.TeamSpecsDryRunAssumptions{WindowsEnabledAndConfigured: optjson.SetBool(false)})
		require.NoError(t, err)
		require.False(t, q.Has("assume_enabled"))
	})
}
