package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/test"
	fleet_httptest "github.com/fleetdm/fleet/v4/server/test/httptest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestOsqueryDoesNotChangeHostUUID() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)

	mac, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	otherMac, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	windows, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	otherWindows, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	for _, hosts := range [][2]*fleet.Host{{mac, otherMac}, {windows, otherWindows}} {
		host, other := hosts[0], hosts[1]
		t.Run(host.Platform, func(t *testing.T) {
			s.postNoAuth(t, "/api/osquery/distributed/write", SubmitDistributedQueryResultsRequest{
				NodeKey: *host.NodeKey,
				Results: map[string][]map[string]string{
					"fleet_detail_query_system_info": {{"uuid": other.UUID, "hardware_serial": host.HardwareSerial}},
				},
				Statuses: map[string]fleet.OsqueryStatus{"fleet_detail_query_system_info": 0},
			}, http.StatusOK)

			stored, err := s.ds.Host(ctx, host.ID)
			require.NoError(t, err)
			assert.Equal(t, host.UUID, stored.UUID)
			s.requireOnlyHostWithUUID(t, other)
		})
	}
}

func (s *integrationMDMTestSuite) TestEnrollWithUUIDOfAnotherMDMHost() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	secrets := []struct{ name, secret string }{{"fleet secret", uuid.NewString()}, {"global secret", uuid.NewString()}}
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, &team.ID, []*fleet.EnrollSecret{{Secret: secrets[0].secret, TeamID: &team.ID}}))
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: secrets[1].secret}}))

	mac, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	windows, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	const rejectedActivity = "host_enrollment_rejected"
	for _, host := range []*fleet.Host{mac, windows} {
		// Host IDs restart with each test database, but the activity rate limit lives in Redis.
		require.NoError(t, s.keyValueStore.Delete(ctx, fmt.Sprintf("%s:host:%d:%s",
			enrollmentRejectedKeyPrefix, host.ID, fleet.EnrollmentRejectedUUIDOfAnotherMDMHost)))

		osquerySerial := mdmtest.RandSerialNumber()
		for _, secret := range secrets {
			t.Run(host.Platform+" "+secret.name, func(t *testing.T) {
				t.Run("osquery enroll", func(t *testing.T) {
					s.postNoAuth(t, "/api/osquery/enroll", contract.EnrollOsqueryAgentRequest{
						EnrollSecret:   secret.secret,
						HostIdentifier: uuid.NewString(),
						HostDetails: map[string]map[string]string{
							"os_version":  {"platform": host.Platform},
							"system_info": {"uuid": host.UUID, "hardware_serial": osquerySerial},
						},
					}, http.StatusUnauthorized)

					s.requireOnlyHostWithUUID(t, host)
				})

				t.Run("osquery enroll, then report the UUID", func(t *testing.T) {
					nodeKey := s.enrollOsqueryWithoutUUID(t, secret.secret, uuid.NewString(), host.Platform)

					s.postNoAuth(t, "/api/osquery/distributed/write", SubmitDistributedQueryResultsRequest{
						NodeKey: nodeKey,
						Results: map[string][]map[string]string{
							"fleet_detail_query_system_info": {{"uuid": host.UUID, "hardware_serial": mdmtest.RandSerialNumber()}},
						},
						Statuses: map[string]fleet.OsqueryStatus{"fleet_detail_query_system_info": 0},
					}, http.StatusOK)

					s.requireOnlyHostWithUUID(t, host)
				})

				t.Run("orbit enroll", func(t *testing.T) {
					s.postNoAuth(t, "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
						EnrollSecret:      secret.secret,
						HardwareUUID:      host.UUID,
						HardwareSerial:    mdmtest.RandSerialNumber(),
						Platform:          host.Platform,
						OsqueryIdentifier: uuid.NewString(),
					}, http.StatusUnauthorized)

					s.requireOnlyHostWithUUID(t, host)
				})

				t.Run("orbit enroll onto a host without a UUID", func(t *testing.T) {
					osqueryID := uuid.NewString()
					s.enrollOsqueryWithoutUUID(t, secret.secret, osqueryID, host.Platform)

					s.postNoAuth(t, "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
						EnrollSecret:      secret.secret,
						HardwareUUID:      host.UUID,
						HardwareSerial:    mdmtest.RandSerialNumber(),
						Platform:          host.Platform,
						OsqueryIdentifier: osqueryID,
					}, http.StatusUnauthorized)

					s.requireOnlyHostWithUUID(t, host)
				})
			})
		}

		s.lastHostActivityMatches(host.ID, rejectedActivity, fmt.Sprintf(`{
			"host_id": %d,
			"host_display_name": %q,
			"host_serial": %q,
			"host_uuid": %q,
			"platform": %q,
			"enrollment_plane": "osquery",
			"reason": "uuid_of_another_mdm_host"
		}`, host.ID, host.DisplayName(), osquerySerial, host.UUID, host.Platform), 0)
		require.Equal(t, 1, s.countHostActivitiesOfType(host.ID, rejectedActivity))
	}
}

func (s *integrationMDMTestSuite) TestEnrollmentNextToMDMHostUUIDStillAllowed() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	secret := uuid.NewString()
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, &team.ID, []*fleet.EnrollSecret{{Secret: secret, TeamID: &team.ID}}))

	newMDMHosts := func(t *testing.T) []*fleet.Host {
		mac, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
		windows, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
		return []*fleet.Host{mac, windows}
	}

	t.Run("re-enrolls as itself", func(t *testing.T) {
		for _, host := range newMDMHosts(t) {
			s.postNoAuth(t, "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
				EnrollSecret:      secret,
				HardwareUUID:      host.UUID,
				HardwareSerial:    host.HardwareSerial,
				Platform:          host.Platform,
				OsqueryIdentifier: *host.OsqueryHostID,
			}, http.StatusOK)
			s.postNoAuth(t, "/api/osquery/enroll", contract.EnrollOsqueryAgentRequest{
				EnrollSecret:   secret,
				HostIdentifier: *host.OsqueryHostID,
				HostDetails: map[string]map[string]string{
					"os_version":  {"platform": host.Platform},
					"system_info": {"uuid": host.UUID, "hardware_serial": host.HardwareSerial},
				},
			}, http.StatusOK)

			s.requireOnlyHostWithUUID(t, host)
		}
	})

	t.Run("hosts already sharing the UUID re-enroll", func(t *testing.T) {
		for _, host := range newMDMHosts(t) {
			duplicate, err := s.ds.NewHost(ctx, &fleet.Host{
				DetailUpdatedAt: time.Now(),
				LabelUpdatedAt:  time.Now(),
				PolicyUpdatedAt: time.Now(),
				SeenTime:        time.Now(),
				OsqueryHostID:   new(uuid.NewString()),
				NodeKey:         new(uuid.NewString()),
				UUID:            host.UUID,
				Platform:        host.Platform,
			})
			require.NoError(t, err)

			for _, osqueryID := range []string{*host.OsqueryHostID, *duplicate.OsqueryHostID} {
				s.postNoAuth(t, "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
					EnrollSecret:      secret,
					HardwareUUID:      host.UUID,
					HardwareSerial:    host.HardwareSerial,
					Platform:          host.Platform,
					OsqueryIdentifier: osqueryID,
				}, http.StatusOK)
			}
		}
	})

	t.Run("a deleted host comes back", func(t *testing.T) {
		for _, host := range newMDMHosts(t) {
			s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK)

			s.postNoAuth(t, "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
				EnrollSecret:   secret,
				HardwareUUID:   host.UUID,
				HardwareSerial: host.HardwareSerial,
				Platform:       host.Platform,
			}, http.StatusOK)

			hosts, err := s.ds.ListHostsLiteByUUIDs(ctx, fleet.TeamFilter{User: test.UserAdmin}, []string{host.UUID})
			require.NoError(t, err)
			require.Len(t, hosts, 1)
		}
	})

	t.Run("a host without a UUID reports one", func(t *testing.T) {
		nodeKey := s.enrollOsqueryWithoutUUID(t, secret, uuid.NewString(), "darwin")
		hostUUID := strings.ToUpper(uuid.NewString())

		s.postNoAuth(t, "/api/osquery/distributed/write", SubmitDistributedQueryResultsRequest{
			NodeKey: nodeKey,
			Results: map[string][]map[string]string{
				"fleet_detail_query_system_info": {{"uuid": hostUUID, "hardware_serial": mdmtest.RandSerialNumber()}},
			},
			Statuses: map[string]fleet.OsqueryStatus{"fleet_detail_query_system_info": 0},
		}, http.StatusOK)

		host, err := s.ds.LoadHostByNodeKey(ctx, nodeKey)
		require.NoError(t, err)
		require.Equal(t, hostUUID, host.UUID)
	})
}

// postNoAuth sends an unauthenticated POST and requires wantStatus. Unlike the suite's request helpers, a mismatch fails
// only t, so the remaining subtests still run.
func (s *integrationMDMTestSuite) postNoAuth(t *testing.T, path string, params any, wantStatus int) *http.Response {
	body, err := json.Marshal(params)
	require.NoError(t, err)
	res := fleet_httptest.DoHTTPReq(t, fleethttp.NewClient(), decodeJSON, "POST", body, s.server.URL+path, nil, wantStatus)
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func (s *integrationMDMTestSuite) enrollOsqueryWithoutUUID(t *testing.T, secret, osqueryID, platform string) string {
	res := s.postNoAuth(t, "/api/osquery/enroll", contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   secret,
		HostIdentifier: osqueryID,
		HostDetails:    map[string]map[string]string{"os_version": {"platform": platform}},
	}, http.StatusOK)
	var enrollResp contract.EnrollOsqueryAgentResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&enrollResp))
	require.NotEmpty(t, enrollResp.NodeKey)
	return enrollResp.NodeKey
}

func (s *integrationMDMTestSuite) requireOnlyHostWithUUID(t *testing.T, host *fleet.Host) {
	hosts, err := s.ds.ListHostsLiteByUUIDs(t.Context(), fleet.TeamFilter{User: test.UserAdmin}, []string{host.UUID})
	require.NoError(t, err)
	require.Len(t, hosts, 1)
	require.Equal(t, host.ID, hosts[0].ID)
}
