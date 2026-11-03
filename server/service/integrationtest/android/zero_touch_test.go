package android

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

const zeroTouchConfigPath = "/api/v1/fleet/android_enterprise/zero_touch_configuration"

func TestAndroidZeroTouchPremium(t *testing.T) {
	s := SetUpSuite(t, "integrationtest.AndroidZeroTouchPremium", WithEEAndroidService(fleet.TierPremium))

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"AndroidNotConfigured", testZeroTouchAndroidNotConfigured},
		{"DestinationFleets", testZeroTouchDestinationFleets},
		{"NonAdminsRejected", testZeroTouchNonAdminsRejected},
		{"DeletedFleet", testZeroTouchDeletedFleet},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Truncating everything would also drop the users and sessions the suite logs in with.
			defer mysqltest.TruncateTables(t, s.DS, "android_zero_touch_tokens", "android_enterprises", "teams")
			c.fn(t, s)
		})
	}
}

func TestAndroidZeroTouchFree(t *testing.T) {
	s := SetUpSuite(t, "integrationtest.AndroidZeroTouchFree", WithEEAndroidService(fleet.TierFree))
	amapiCalls := setUpZeroTouch(t, s)

	fleetA, err := s.DS.NewTeam(t.Context(), &fleet.Team{Name: "zero-touch-free"})
	require.NoError(t, err)

	s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusPaymentRequired)
	s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusPaymentRequired, "fleet_id", fmt.Sprint(fleetA.ID))
	assert.Empty(t, *amapiCalls)
}

// setUpZeroTouch turns on Android MDM, creates an enterprise, and records the
// additionalData of every AMAPI enrollment token Fleet creates.
func setUpZeroTouch(t *testing.T, s *Suite) *[]string {
	t.Helper()

	appCfg, err := s.DS.AppConfig(t.Context())
	require.NoError(t, err)
	appCfg.MDM.AndroidEnabledAndConfigured = true
	require.NoError(t, s.DS.SaveAppConfig(t.Context(), appCfg))
	t.Cleanup(func() {
		appCfg.MDM.AndroidEnabledAndConfigured = false
		// t.Context() is already canceled when cleanups run.
		require.NoError(t, s.DS.SaveAppConfig(context.Background(), appCfg))
	})

	enterpriseID, err := s.DS.CreateEnterprise(t.Context(), s.Users["admin1"].ID)
	require.NoError(t, err)
	require.NoError(t, s.DS.UpdateEnterprise(t.Context(), &android.EnterpriseDetails{
		ID:           enterpriseID,
		EnterpriseID: "LC00zerotouch",
	}))

	var additionalData []string
	s.AndroidProxy.EnterprisesEnrollmentTokensCreateFunc = func(_ context.Context, enterpriseName string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error) {
		assert.Equal(t, "enterprises/LC00zerotouch", enterpriseName)
		additionalData = append(additionalData, token.AdditionalData)
		n := len(additionalData)
		return &androidmanagement.EnrollmentToken{
			Name:                fmt.Sprintf("enterprises/LC00zerotouch/enrollmentTokens/%d", n),
			Value:               fmt.Sprintf("zero-touch-token-%d", n),
			ExpirationTimestamp: time.Now().Add(100 * 365 * 24 * time.Hour).Format(time.RFC3339),
		}, nil
	}
	return &additionalData
}

func getZeroTouchToken(t *testing.T, s *Suite, queryParams ...string) string {
	t.Helper()
	var resp map[string]map[string]string
	s.DoJSON(t, "GET", zeroTouchConfigPath, nil, http.StatusOK, &resp, queryParams...)
	bundle, ok := resp["android.app.extra.PROVISIONING_ADMIN_EXTRAS_BUNDLE"]
	require.True(t, ok, "response is missing the admin extras bundle: %v", resp)
	return bundle["com.google.android.apps.work.clouddpc.EXTRA_ENROLLMENT_TOKEN"]
}

func countZeroTouchTokens(t *testing.T, s *Suite) int {
	t.Helper()
	var count int
	mysqltest.ExecAdhocSQL(t, s.DS, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &count, `SELECT COUNT(*) FROM android_zero_touch_tokens`)
	})
	return count
}

func testZeroTouchAndroidNotConfigured(t *testing.T, s *Suite) {
	fleetA, err := s.DS.NewTeam(t.Context(), &fleet.Team{Name: "zero-touch-no-android"})
	require.NoError(t, err)

	s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusNotFound)
	s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusNotFound, "fleet_id", fmt.Sprint(fleetA.ID))
}

func testZeroTouchDestinationFleets(t *testing.T, s *Suite) {
	amapiCalls := setUpZeroTouch(t, s)

	fleetA, err := s.DS.NewTeam(t.Context(), &fleet.Team{Name: "zero-touch-a"})
	require.NoError(t, err)
	fleetB, err := s.DS.NewTeam(t.Context(), &fleet.Team{Name: "zero-touch-b"})
	require.NoError(t, err)
	fleetAID := strconv.FormatUint(uint64(fleetA.ID), 10)
	fleetBID := strconv.FormatUint(uint64(fleetB.ID), 10)

	// Unassigned, with no param and with fleet_id=0, shares one token.
	unassignedToken := getZeroTouchToken(t, s)
	require.Len(t, *amapiCalls, 1)
	assert.JSONEq(t, `{"fleet_id":null}`, (*amapiCalls)[0])
	assert.Equal(t, unassignedToken, getZeroTouchToken(t, s, "fleet_id", "0"))

	fleetAToken := getZeroTouchToken(t, s, "fleet_id", fleetAID)
	require.Len(t, *amapiCalls, 2)
	assert.JSONEq(t, fmt.Sprintf(`{"fleet_id":%d}`, fleetA.ID), (*amapiCalls)[1])
	assert.NotEqual(t, unassignedToken, fleetAToken)

	// Repeat requests, including via the deprecated team_id name, reuse the stored token.
	assert.Equal(t, fleetAToken, getZeroTouchToken(t, s, "fleet_id", fleetAID))
	assert.Equal(t, fleetAToken, getZeroTouchToken(t, s, "team_id", fleetAID))
	require.Len(t, *amapiCalls, 2)

	fleetBToken := getZeroTouchToken(t, s, "fleet_id", fleetBID)
	require.Len(t, *amapiCalls, 3)
	assert.JSONEq(t, fmt.Sprintf(`{"fleet_id":%d}`, fleetB.ID), (*amapiCalls)[2])
	assert.NotEqual(t, fleetAToken, fleetBToken)

	assert.Equal(t, 3, countZeroTouchTokens(t, s))

	stored, err := s.DS.GetZeroTouchEnrollmentToken(t.Context(), &fleetA.ID)
	require.NoError(t, err)
	assert.Equal(t, fleetAToken, stored.TokenValue)

	// A fleet that doesn't exist is a 404 and never reaches AMAPI.
	s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusNotFound, "fleet_id", "999999")
	s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusBadRequest, "fleet_id", "abc")
	s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusBadRequest, "fleet_id", fleetAID, "team_id", fleetBID)
	assert.Len(t, *amapiCalls, 3)
	assert.Equal(t, 3, countZeroTouchTokens(t, s))
}

func testZeroTouchNonAdminsRejected(t *testing.T, s *Suite) {
	amapiCalls := setUpZeroTouch(t, s)

	fleetA, err := s.DS.NewTeam(t.Context(), &fleet.Team{Name: "zero-touch-authz"})
	require.NoError(t, err)

	adminToken := s.Token
	t.Cleanup(func() { s.Token = adminToken })

	for _, email := range []string{service.TestMaintainerUserEmail, service.TestObserverUserEmail} {
		t.Run(email, func(t *testing.T) {
			s.Token = s.GetTestToken(t, email, test.GoodPassword)
			s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusForbidden)
			s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusForbidden, "fleet_id", fmt.Sprint(fleetA.ID))
			// A missing fleet is still 403, so non-admins can't probe which fleets exist.
			s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusForbidden, "fleet_id", "999999")
		})
	}

	assert.Empty(t, *amapiCalls)
	assert.Zero(t, countZeroTouchTokens(t, s))
}

func testZeroTouchDeletedFleet(t *testing.T, s *Suite) {
	amapiCalls := setUpZeroTouch(t, s)

	deletedFleet, err := s.DS.NewTeam(t.Context(), &fleet.Team{Name: "zero-touch-deleted"})
	require.NoError(t, err)
	keptFleet, err := s.DS.NewTeam(t.Context(), &fleet.Team{Name: "zero-touch-kept"})
	require.NoError(t, err)

	getZeroTouchToken(t, s, "fleet_id", fmt.Sprint(deletedFleet.ID))
	keptToken := getZeroTouchToken(t, s, "fleet_id", fmt.Sprint(keptFleet.ID))
	require.Equal(t, 2, countZeroTouchTokens(t, s))

	require.NoError(t, s.DS.DeleteTeam(t.Context(), deletedFleet.ID))

	assert.Equal(t, 1, countZeroTouchTokens(t, s))
	_, err = s.DS.GetZeroTouchEnrollmentToken(t.Context(), &deletedFleet.ID)
	assert.True(t, fleet.IsNotFound(err))

	s.Do(t, "GET", zeroTouchConfigPath, nil, http.StatusNotFound, "fleet_id", fmt.Sprint(deletedFleet.ID))
	assert.Equal(t, keptToken, getZeroTouchToken(t, s, "fleet_id", fmt.Sprint(keptFleet.ID)))
	assert.Len(t, *amapiCalls, 2)
}
