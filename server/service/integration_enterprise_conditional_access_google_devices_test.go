package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const googleTestCustomer = "C02cajzvg"

// googleDeviceTest drives the device side of Google conditional access against a
// connected integration.
type googleDeviceTest struct {
	*googleCATest
	syncer *GoogleConditionalAccessSyncer
	// cronActivities are the activities the sync cron logged.
	cronActivities []fleet.ActivityDetails
}

func (s *integrationEnterpriseTestSuite) newGoogleDeviceTest() *googleDeviceTest {
	g := &googleDeviceTest{googleCATest: s.newGoogleCATest()}
	g.syncer = NewGoogleConditionalAccessSyncer(s.ds, &GoogleConditionalAccessClient{
		Relay:         g.client,
		CloudIdentity: g.client.cloudIdentity,
	}, func(_ context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		require.Nil(g.t, user)
		g.cronActivities = append(g.cronActivities, activity)
		return nil
	}, slog.New(slog.DiscardHandler))
	// Removed through the API so the server's cached app config doesn't keep the
	// setting into the next test.
	g.t.Cleanup(func() {
		s.token = s.getTestAdminToken()
		if g.getConfig().Integrations.GoogleConditionalAccess != nil {
			s.DoJSON("DELETE", "/api/latest/fleet/conditional_access/google", nil, http.StatusOK, &deleteGoogleConditionalAccessResponse{})
		}
	})
	return g
}

func (g *googleDeviceTest) connectIntegration() {
	key := newGoogleCATestKey(g.t, "sa@example.iam.gserviceaccount.com")
	g.connect(googleTestCustomer, "admin@example.com", key, http.StatusOK)
	g.client.reset()
}

// newIOSHost creates an iOS or iPadOS host enrolled in MDM; automatic makes it company-owned.
func (g *googleDeviceTest) newIOSHost(platform string, automatic bool) *fleet.Host {
	t := g.t
	host := test.NewHost(t, g.s.ds, platform+"-"+uuid.NewString(), "", uuid.NewString(), uuid.NewString(), time.Now(), test.WithPlatform(platform))
	cleanupCtx := context.WithoutCancel(t.Context())
	t.Cleanup(func() { _ = g.s.ds.DeleteHost(cleanupCtx, host.ID) })
	personal := fleet.PersonalEnrollmentTypeNone
	if !automatic {
		personal = fleet.PersonalEnrollmentTypeAccountDriven
	}
	require.NoError(t, g.s.ds.SetOrUpdateMDMData(t.Context(), host.ID, false, true, g.s.server.URL, automatic, fleet.WellKnownMDMFleet, "", personal))
	return host
}

// newAndroidHost enrolls an Android host the way Android MDM does, so eligibility is
// checked against the flags enrollment writes.
func (g *googleDeviceTest) newAndroidHost(companyOwned bool) *fleet.Host {
	t := g.t
	hostUUID := uuid.NewString()
	ah := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname: "android-" + hostUUID, ComputerName: "android-" + hostUUID, Platform: "android",
			OSVersion: "Android 14", HardwareSerial: uuid.NewString(), UUID: hostUUID,
		},
		Device: &android.Device{DeviceID: strings.ReplaceAll(uuid.NewString(), "-", ""), AppliedPolicyID: new("1")},
	}
	ah.SetNodeKey(hostUUID)
	created, err := g.s.ds.NewAndroidHost(t.Context(), ah, companyOwned)
	require.NoError(t, err)
	host := created.Host
	cleanupCtx := context.WithoutCancel(t.Context())
	t.Cleanup(func() { _ = g.s.ds.DeleteHost(cleanupCtx, host.ID) })
	return host
}

func (g *googleDeviceTest) androidDeviceID(host *fleet.Host) string {
	var id string
	mysqltest.ExecAdhocSQL(g.t, g.s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(g.t.Context(), q, &id, `SELECT device_id FROM android_devices WHERE host_id = ?`, host.ID)
	})
	return id
}

func (g *googleDeviceTest) iosStatus(host *fleet.Host) fleet.GoogleConditionalAccessDeviceStatus {
	var resp fleet.GoogleConditionalAccessDeviceStatus
	res := g.s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+host.UUID+"/conditional_access/google", nil, http.StatusOK)
	require.NoError(g.t, json.NewDecoder(res.Body).Decode(&resp))
	return resp
}

// signIn returns the token in the sign-in URL the status route hands out.
func (g *googleDeviceTest) signIn(host *fleet.Host) string {
	st := g.iosStatus(host)
	require.NotEmpty(g.t, st.SignInURL, "%+v", st)
	u, err := url.Parse(st.SignInURL)
	require.NoError(g.t, err)
	require.Equal(g.t, "/api/v1/google-conditional-access/sign-in", u.Path)
	require.Equal(g.t, googleTestCustomer, u.Query().Get("googleCustomerId"))
	require.Equal(g.t, host.UUID, u.Query().Get("hostUuid"))
	require.NotEmpty(g.t, u.Query().Get("token"))
	return u.Query().Get("token")
}

func (g *googleDeviceTest) register(secret string, reg fleet.GoogleConditionalAccessRegistration, wantStatus int) fleet.GoogleConditionalAccessRegisterResult {
	body, err := json.Marshal(reg)
	require.NoError(g.t, err)
	res := g.s.DoRawWithHeaders("POST", "/api/latest/fleet/google_conditional_access/register", body, wantStatus,
		map[string]string{"Authorization": "Bearer " + secret})
	var resp struct {
		Result fleet.GoogleConditionalAccessRegisterResult `json:"result"`
	}
	b, err := io.ReadAll(res.Body)
	require.NoError(g.t, err)
	require.NoError(g.t, json.Unmarshal(b, &resp), string(b))
	return resp.Result
}

func (g *googleDeviceTest) registerNames(token string, names ...string) fleet.GoogleConditionalAccessRegisterResult {
	return g.register("secret-"+googleTestCustomer, fleet.GoogleConditionalAccessRegistration{
		Token: token, Names: names, Customer: googleTestCustomer, Email: "user@example.com",
	}, http.StatusOK)
}

func (g *googleDeviceTest) states(host *fleet.Host) map[string]fleet.GoogleConditionalAccessDeviceUserState {
	states := map[string]fleet.GoogleConditionalAccessDeviceUserState{}
	mysqltest.ExecAdhocSQL(g.t, g.s.ds, func(q sqlx.ExtContext) error {
		var rows []fleet.GoogleConditionalAccessDeviceUser
		if err := sqlx.SelectContext(g.t.Context(), q, &rows, `SELECT device_user_name, state FROM google_conditional_access_device_users WHERE host_uuid = ?`, host.UUID); err != nil {
			return err
		}
		for _, r := range rows {
			states[r.Name] = r.State
		}
		return nil
	})
	return states
}

func (g *googleDeviceTest) device(host *fleet.Host) *fleet.GoogleConditionalAccessDevice {
	dev, err := g.s.ds.GetGoogleConditionalAccessDevice(g.t.Context(), host.UUID, googleTestCustomer)
	require.NoError(g.t, err)
	require.NotNil(g.t, dev)
	return dev
}

func (g *googleDeviceTest) hasDevice(host *fleet.Host) bool {
	_, err := g.s.ds.GetGoogleConditionalAccessDevice(g.t.Context(), host.UUID, googleTestCustomer)
	if fleet.IsNotFound(err) {
		return false
	}
	require.NoError(g.t, err)
	return true
}

// googleDeviceID returns the Google device ID host details show, "" when there's none.
func (g *googleDeviceTest) googleDeviceID(host *fleet.Host) string {
	var resp getHostResponse
	g.s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &resp)
	require.NotNil(g.t, resp.Host)
	if resp.Host.GoogleDeviceID == nil {
		return ""
	}
	return *resp.Host.GoogleDeviceID
}

// googleDeviceIDActivity is the details of the activity both ways of binding a host log.
func googleDeviceIDActivity(host *fleet.Host, googleDeviceID string) string {
	return fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "google_device_id": %q}`, host.ID, host.DisplayName(), googleDeviceID)
}

// googleSignInActivity is the details of the activity a sign-in logs, which also names
// the Google account that signed in.
func googleSignInActivity(host *fleet.Host, googleDeviceID string) string {
	return fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "google_device_id": %q, "google_email": "user@example.com"}`,
		host.ID, host.DisplayName(), googleDeviceID)
}

func (g *googleDeviceTest) sync() {
	require.NoError(g.t, g.syncer.Sync(g.t.Context()))
}

// syncAfter runs the sync cron as if d had passed.
func (g *googleDeviceTest) syncAfter(d time.Duration) {
	g.syncer.now = func() time.Time { return time.Now().Add(d) }
	defer func() { g.syncer.now = time.Now }()
	g.sync()
}

func (g *googleDeviceTest) execSQL(stmt string, args ...any) {
	mysqltest.ExecAdhocSQL(g.t, g.s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(g.t.Context(), stmt, args...)
		return err
	})
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessGoogleDeviceRegistration() {
	t := s.T()
	g := s.newGoogleDeviceTest()

	ade := g.newIOSHost("ios", true)
	byod := g.newIOSHost("ios", false)
	mac := createHostAndDeviceToken(t, s.ds, "google-ca-mac-token")
	cleanupCtx := context.WithoutCancel(t.Context())
	t.Cleanup(func() { _ = s.ds.DeleteHost(cleanupCtx, mac.ID) })

	// Not connected: nothing is enabled and Google isn't called.
	require.Equal(t, fleet.GoogleConditionalAccessDeviceStatus{}, g.iosStatus(ade))
	require.Empty(t, g.googleDeviceID(ade))
	g.sync()
	require.Zero(t, g.client.devCalls().googleCalls)
	g.register("anything", fleet.GoogleConditionalAccessRegistration{Token: "t"}, http.StatusUnauthorized)

	g.connectIntegration()

	// A company-owned iPhone is handed a sign-in URL; a personal one and a Mac aren't.
	token := g.signIn(ade)
	st := g.iosStatus(byod)
	require.Equal(t, fleet.GoogleConditionalAccessDeviceStatus{Enabled: true}, st)
	var macStatus fleet.GoogleConditionalAccessDeviceStatus
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/google-ca-mac-token/conditional_access/google", nil, http.StatusOK)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&macStatus))
	require.Equal(t, fleet.GoogleConditionalAccessDeviceStatus{}, macStatus)
	require.Empty(t, g.googleDeviceID(ade))

	// A wrong secret is refused and leaves the token usable.
	g.register("wrong", fleet.GoogleConditionalAccessRegistration{Token: token, Names: []string{"devices/x/deviceUsers/y"}}, http.StatusUnauthorized)
	g.register("", fleet.GoogleConditionalAccessRegistration{Token: token}, http.StatusUnauthorized)

	// Registering two names stores and marks both, and binds the host to their device.
	activitiesBefore := s.countActivitiesOfType(fleet.ActivityTypeSetHostGoogleDeviceID{}.ActivityName())
	names := []string{"devices/ade/deviceUsers/1", "devices/ade/deviceUsers/2"}
	require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, g.registerNames(token, names...))
	require.Equal(t, []string{names[0] + " " + ade.UUID, names[1] + " " + ade.UUID}, g.client.devCalls().managed)
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		names[0]: fleet.GoogleConditionalAccessDeviceUserManaged, names[1]: fleet.GoogleConditionalAccessDeviceUserManaged,
	}, g.states(ade))
	require.Equal(t, activitiesBefore+1, s.countActivitiesOfType(fleet.ActivityTypeSetHostGoogleDeviceID{}.ActivityName()))
	activityID := s.lastActivityOfTypeMatches(fleet.ActivityTypeSetHostGoogleDeviceID{}.ActivityName(), googleSignInActivity(ade, "ade"), 0)
	// The end user signed in, not a Fleet user.
	var fleetInitiated bool
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &fleetInitiated, `SELECT fleet_initiated FROM activity_past WHERE id = ?`, activityID)
	})
	require.True(t, fleetInitiated)
	require.Equal(t, "ade", g.googleDeviceID(ade))

	// A registered host gets no URL and no attempt is counted.
	require.Equal(t, fleet.GoogleConditionalAccessDeviceStatus{Enabled: true, Eligible: true, Registered: true, Managed: true}, g.iosStatus(ade))
	require.Zero(t, g.device(ade).RegistrationAttempts)

	// The token can't be replayed.
	require.Equal(t, fleet.GoogleConditionalAccessRegisterNotAllowed, g.registerNames(token, "devices/other/deviceUsers/1"))

	// A host unenrolled after the URL was handed out is refused.
	unenrolled := g.newIOSHost("ipados", true)
	token = g.signIn(unenrolled)
	require.NoError(t, s.ds.SetOrUpdateMDMData(t.Context(), unenrolled.ID, false, false, "", false, "", "", fleet.PersonalEnrollmentTypeNone))
	require.Equal(t, fleet.GoogleConditionalAccessRegisterNotAllowed, g.registerNames(token, "devices/u/deviceUsers/1"))
	require.Empty(t, g.states(unenrolled))

	// So is a host deleted after the URL was handed out.
	gone := g.newIOSHost("ios", true)
	token = g.signIn(gone)
	require.NoError(t, s.ds.DeleteHost(t.Context(), gone.ID))
	require.Equal(t, fleet.GoogleConditionalAccessRegisterNotAllowed, g.registerNames(token, "devices/gone/deviceUsers/1"))
	require.False(t, g.hasDevice(gone))

	// A name stored on another host is refused and nothing is stored.
	second := g.newIOSHost("ios", true)
	token = g.signIn(second)
	require.Equal(t, fleet.GoogleConditionalAccessRegisterNotAllowed, g.registerNames(token, "devices/second/deviceUsers/1", names[1]))
	require.Empty(t, g.states(second))

	// An expired token is refused.
	token = g.signIn(second)
	g.execSQL(`UPDATE google_conditional_access_devices SET registration_token_expires_at = NOW(6) - INTERVAL 1 SECOND WHERE host_uuid = ?`, second.UUID)
	require.Equal(t, fleet.GoogleConditionalAccessRegisterNotAllowed, g.registerNames(token, "devices/second/deviceUsers/1"))
	require.Empty(t, g.states(second))

	// A sign-in to another Workspace is refused.
	token = g.signIn(second)
	require.Equal(t, fleet.GoogleConditionalAccessRegisterNotAllowed, g.register("secret-"+googleTestCustomer,
		fleet.GoogleConditionalAccessRegistration{Token: token, Names: []string{"devices/second/deviceUsers/1"}, Customer: "C09other"}, http.StatusOK))
	require.Empty(t, g.states(second))

	// No names: nothing is stored and the attempt counter is reset.
	token = g.signIn(second)
	require.NotZero(t, g.device(second).RegistrationAttempts)
	require.Equal(t, fleet.GoogleConditionalAccessRegisterNotFound, g.registerNames(token))
	require.Empty(t, g.states(second))
	require.Zero(t, g.device(second).RegistrationAttempts)

	// A failed mark still registers the host; the cron marks the name on its next run.
	failing := "devices/second/deviceUsers/fails"
	g.client.setDev(func(d *googleDeviceMock) {
		d.markErrs = map[string]error{failing: errors.New("Google returned HTTP 500")}
	})
	token = g.signIn(second)
	require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, g.registerNames(token, "devices/second/deviceUsers/ok", failing))
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/second/deviceUsers/ok": fleet.GoogleConditionalAccessDeviceUserManaged,
		failing:                         fleet.GoogleConditionalAccessDeviceUserPending,
	}, g.states(second))
	st = g.iosStatus(second)
	require.True(t, st.Registered)
	require.False(t, st.Managed)
	require.Empty(t, st.SignInURL)
	require.Equal(t, "second", g.googleDeviceID(second))
	g.client.setDev(func(d *googleDeviceMock) { d.markErrs = nil })
	g.sync()
	require.Equal(t, fleet.GoogleConditionalAccessDeviceUserPending, g.states(second)[failing], "retried before the backoff")
	g.syncAfter(2 * time.Minute)
	require.Equal(t, fleet.GoogleConditionalAccessDeviceUserManaged, g.states(second)[failing])
	require.True(t, g.iosStatus(second).Managed)

	// Names under two device records are all stored and marked; the host shows the
	// first one's device.
	wiped := g.newIOSHost("ipados", true)
	token = g.signIn(wiped)
	require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, g.registerNames(token, "devices/wiped-old/deviceUsers/1", "devices/wiped-new/deviceUsers/1"))
	require.Len(t, g.states(wiped), 2)
	require.Equal(t, "wiped-old", g.googleDeviceID(wiped))

	// A sign-in on a host whose names went stale replaces its device ID.
	g.execSQL(`UPDATE google_conditional_access_device_users SET state = 'stale' WHERE host_uuid = ?`, wiped.UUID)
	token = g.signIn(wiped)
	require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, g.registerNames(token, "devices/wiped-newest/deviceUsers/1"))
	require.Equal(t, "wiped-newest", g.googleDeviceID(wiped))
	s.lastActivityOfTypeMatches(fleet.ActivityTypeSetHostGoogleDeviceID{}.ActivityName(), googleSignInActivity(wiped, "wiped-newest"), 0)
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/wiped-newest/deviceUsers/1": fleet.GoogleConditionalAccessDeviceUserManaged,
	}, g.states(wiped))
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessGoogleDeviceRetryCap() {
	t := s.T()
	g := s.newGoogleDeviceTest()
	g.connectIntegration()
	host := g.newIOSHost("ios", true)

	for range fleet.GoogleConditionalAccessMaxRegistrationAttempts {
		g.signIn(host)
	}
	st := g.iosStatus(host)
	require.Equal(t, fleet.GoogleConditionalAccessDeviceStatus{Enabled: true, Eligible: true, RetryBlocked: true}, st)

	// Once the first attempt is older than the window, a URL is handed out again.
	g.execSQL(`UPDATE google_conditional_access_devices SET first_registration_attempt_at = NOW(6) - INTERVAL 25 HOUR WHERE host_uuid = ?`, host.UUID)
	g.signIn(host)
	g.signIn(host)
	token := g.signIn(host)
	require.True(t, g.iosStatus(host).RetryBlocked)

	// A register call resets the counter, whatever its result. The last token is
	// still valid while blocked.
	require.Equal(t, fleet.GoogleConditionalAccessRegisterNotFound, g.registerNames(token))
	g.signIn(host)
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessGoogleUnmark() {
	t := s.T()
	g := s.newGoogleDeviceTest()
	g.connectIntegration()

	register := func(host *fleet.Host, names ...string) {
		require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, g.registerNames(g.signIn(host), names...))
	}

	// The admin removal queues the names, and the cron unmarks and removes them.
	removed := g.newIOSHost("ios", true)
	register(removed, "devices/removed/deviceUsers/1", "devices/removed/deviceUsers/2")
	removedPath := fmt.Sprintf("/api/latest/fleet/hosts/%d/google_device_id", removed.ID)
	s.Do("DELETE", removedPath, nil, http.StatusNoContent)
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/removed/deviceUsers/1": fleet.GoogleConditionalAccessDeviceUserUnmanaging,
		"devices/removed/deviceUsers/2": fleet.GoogleConditionalAccessDeviceUserUnmanaging,
	}, g.states(removed))
	s.lastActivityOfTypeMatches(fleet.ActivityTypeDeletedHostGoogleDeviceID{}.ActivityName(), googleDeviceIDActivity(removed, "removed"), 0)
	// The ID is gone at once and the host can register again right away.
	require.Empty(t, g.googleDeviceID(removed))
	require.False(t, g.iosStatus(removed).Registered)
	// Removing it again logs nothing.
	before := s.countActivitiesOfType(fleet.ActivityTypeDeletedHostGoogleDeviceID{}.ActivityName())
	s.Do("DELETE", removedPath, nil, http.StatusNoContent)
	require.Equal(t, before, s.countActivitiesOfType(fleet.ActivityTypeDeletedHostGoogleDeviceID{}.ActivityName()))
	g.sync()
	_, _, _, unmanaged := g.client.calls()
	require.ElementsMatch(t, []string{"devices/removed/deviceUsers/1", "devices/removed/deviceUsers/2"}, unmanaged)
	require.Empty(t, g.states(removed))
	require.False(t, g.hasDevice(removed))

	// Whoever can write the host can remove its registration.
	newUser := func(name string, role string) *fleet.User {
		u := &fleet.User{Name: name, Email: name + "@example.com", GlobalRole: &role}
		require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
		_, err := s.ds.NewUser(t.Context(), u)
		require.NoError(t, err)
		return u
	}
	maintainer := newUser("gca-device-maintainer", fleet.RoleMaintainer)
	s.setTokenForTest(t, maintainer.Email, test.GoodPassword)
	s.Do("DELETE", removedPath, nil, http.StatusNoContent)
	observer := newUser("gca-device-observer", fleet.RoleObserver)
	s.setTokenForTest(t, observer.Email, test.GoodPassword)
	s.Do("DELETE", removedPath, nil, http.StatusForbidden)
	s.token = s.getTestAdminToken()

	// Deleting the host queues the names; a record Google no longer has counts as
	// unmarked, and a failed unmark is retried.
	deleted := g.newIOSHost("ios", true)
	register(deleted, "devices/deleted/deviceUsers/gone", "devices/deleted/deviceUsers/flaky")
	g.client.reset()
	g.client.setDev(func(d *googleDeviceMock) {
		d.unmarkErrs = map[string]error{
			"devices/deleted/deviceUsers/gone":  googleResourceError{status: http.StatusNotFound},
			"devices/deleted/deviceUsers/flaky": errors.New("Google returned HTTP 503"),
		}
	})
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", deleted.ID), nil, http.StatusOK)
	g.sync()
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/deleted/deviceUsers/flaky": fleet.GoogleConditionalAccessDeviceUserUnmanaging,
	}, g.states(deleted))
	g.client.setDev(func(d *googleDeviceMock) { d.unmarkErrs = nil })
	g.sync()
	require.Empty(t, g.states(deleted))
	require.False(t, g.hasDevice(deleted))

	// A deleted host's failed mark is never retried.
	failed := g.newIOSHost("ios", true)
	g.client.setDev(func(d *googleDeviceMock) {
		d.markErrs = map[string]error{"devices/failed/deviceUsers/1": errors.New("Google returned HTTP 500")}
	})
	register(failed, "devices/failed/deviceUsers/1")
	g.client.setDev(func(d *googleDeviceMock) { d.markErrs = nil })
	require.NoError(t, s.ds.DeleteHost(t.Context(), failed.ID))
	g.client.reset()
	g.sync()
	require.Empty(t, g.client.devCalls().managed)
	_, _, _, unmanaged = g.client.calls()
	require.Equal(t, []string{"devices/failed/deviceUsers/1"}, unmanaged)
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessGoogleStaleNames() {
	t := s.T()
	g := s.newGoogleDeviceTest()
	g.connectIntegration()

	// A 404 on a retried mark sets the name stale: the host signs in again and nothing is
	// queued for unmark.
	host := g.newIOSHost("ios", true)
	name := "devices/stale/deviceUsers/1"
	g.client.setDev(func(d *googleDeviceMock) { d.markErrs = map[string]error{name: errors.New("Google returned HTTP 500")} })
	require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, g.registerNames(g.signIn(host), name))
	g.client.setDev(func(d *googleDeviceMock) {
		d.markErrs = map[string]error{name: googleResourceError{status: http.StatusNotFound}}
	})
	g.syncAfter(2 * time.Minute)
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{name: fleet.GoogleConditionalAccessDeviceUserStale}, g.states(host))
	g.signIn(host)
	_, _, _, unmanaged := g.client.calls()
	require.Empty(t, unmanaged)

	// The hourly check deletes a name whose device is gone, sets stale a name whose device
	// left the Workspace, and leaves the others.
	two := g.newIOSHost("ios", true)
	g.client.setDev(func(d *googleDeviceMock) { d.markErrs = nil })
	require.Equal(t, fleet.GoogleConditionalAccessRegisterDone,
		g.registerNames(g.signIn(two), "devices/gone/deviceUsers/1", "devices/left/deviceUsers/1", "devices/here/deviceUsers/1"))
	g.client.setDev(func(d *googleDeviceMock) {
		d.goneDevices = map[string]bool{"gone": true}
		d.leftDevices = map[string]bool{"left": true}
	})
	require.Equal(t, "gone", g.googleDeviceID(two))
	require.NoError(t, g.syncer.CheckDevices(t.Context()))
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/left/deviceUsers/1": fleet.GoogleConditionalAccessDeviceUserStale,
		"devices/here/deviceUsers/1": fleet.GoogleConditionalAccessDeviceUserManaged,
	}, g.states(two))
	require.True(t, g.iosStatus(two).Registered)
	// The host's device ID moves off the deleted record to one that is still marked.
	require.Equal(t, "here", g.googleDeviceID(two))

	// When the last record is deleted the ID is cleared and the host signs in again.
	g.client.setDev(func(d *googleDeviceMock) { d.goneDevices = map[string]bool{"here": true} })
	require.NoError(t, g.syncer.CheckDevices(t.Context()))
	require.Empty(t, g.googleDeviceID(two))
	require.False(t, g.iosStatus(two).Registered)
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessGoogleDelegationFailure() {
	t := s.T()
	g := s.newGoogleDeviceTest()
	g.connectIntegration()

	// A refused delegation fails the run and is shown on the integration until a run
	// succeeds again.
	g.client.checkErr = googleAuthError{}
	require.Error(t, g.syncer.Sync(t.Context()))
	row, err := s.ds.GetGoogleConditionalAccessIntegration(t.Context())
	require.NoError(t, err)
	require.Equal(t, fleet.ConditionalAccessGoogleStateDelegationFailed, row.State)
	require.NotNil(t, row.ErrorMessage)
	g.client.checkErr = nil
	g.sync()
	row, err = s.ds.GetGoogleConditionalAccessIntegration(t.Context())
	require.NoError(t, err)
	require.Equal(t, fleet.ConditionalAccessGoogleStateConnected, row.State)
	require.Nil(t, row.ErrorMessage)

	// A recovered delegation doesn't connect an integration fleetdm.com never registered.
	row.ProxyServerSecret, row.State, row.ErrorMessage = "", fleet.ConditionalAccessGoogleStateDelegationFailed, new("refused")
	require.NoError(t, s.ds.SaveGoogleConditionalAccessIntegration(t.Context(), row))
	g.sync()
	row, err = s.ds.GetGoogleConditionalAccessIntegration(t.Context())
	require.NoError(t, err)
	require.Equal(t, fleet.ConditionalAccessGoogleStateDelegationFailed, row.State)
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessGoogleBookmarkMigration() {
	t := s.T()
	g := s.newGoogleDeviceTest()
	g.connectIntegration()

	migrated := g.newIOSHost("ios", true)
	gone := g.newIOSHost("ios", true)
	personal := g.newIOSHost("ios", false)
	// Rows as the migration from the bookmark flow's table leaves them: no customer, no names.
	for host, deviceID := range map[*fleet.Host]string{migrated: "bm-device", gone: "bm-gone", personal: "bm-personal"} {
		g.execSQL(`INSERT INTO google_conditional_access_devices (host_uuid, customer_id, google_device_id) VALUES (?, '', ?)`, host.UUID, deviceID)
		require.Equal(t, deviceID, g.googleDeviceID(host))
	}
	g.client.setDev(func(d *googleDeviceMock) {
		d.deviceUsers = map[string][]string{
			"bm-device":   {"devices/bm-device/deviceUsers/1", "devices/bm-device/deviceUsers/2"},
			"bm-personal": {"devices/bm-personal/deviceUsers/1"},
		}
		d.goneDevices = map[string]bool{"bm-gone": true}
	})

	// Before the migration pass the host signs in as usual.
	g.signIn(migrated)
	g.sync()
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/bm-device/deviceUsers/1": fleet.GoogleConditionalAccessDeviceUserManaged,
		"devices/bm-device/deviceUsers/2": fleet.GoogleConditionalAccessDeviceUserManaged,
	}, g.states(migrated))
	require.Equal(t, fleet.GoogleConditionalAccessDeviceStatus{Enabled: true, Eligible: true, Registered: true, Managed: true}, g.iosStatus(migrated))
	require.Equal(t, "bm-device", g.googleDeviceID(migrated))
	require.Empty(t, g.states(gone))
	require.NotEmpty(t, g.iosStatus(gone).SignInURL)
	// A personal host is never migrated, and keeps its device ID until an admin removes it.
	require.Empty(t, g.states(personal))
	require.Equal(t, "bm-personal", g.googleDeviceID(personal))
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/google_device_id", personal.ID), nil, http.StatusNoContent)
	require.Empty(t, g.googleDeviceID(personal))
	require.False(t, g.hasDevice(personal))

	// Neither a migrated host nor one whose device is gone is looked up again.
	g.client.setDev(func(d *googleDeviceMock) { d.googleCalls = 0 })
	g.sync()
	require.Zero(t, g.client.devCalls().googleCalls)

	// An admin removal drops the device ID with the names, so the sync doesn't migrate the host again.
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/google_device_id", migrated.ID), nil, http.StatusNoContent)
	require.Empty(t, g.googleDeviceID(migrated))
	s.lastActivityOfTypeMatches(fleet.ActivityTypeDeletedHostGoogleDeviceID{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "google_device_id": "bm-device"}`, migrated.ID, migrated.DisplayName()), 0)
	g.sync()
	require.Empty(t, g.states(migrated))
	require.False(t, g.iosStatus(migrated).Registered)
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessGoogleAndroidResolution() {
	t := s.T()
	g := s.newGoogleDeviceTest()
	managed := fleet.GoogleConditionalAccessDeviceUserManaged

	host := g.newAndroidHost(true)
	personal := g.newAndroidHost(false)
	hostAndroidID := g.androidDeviceID(host)

	// Not connected: Google isn't called.
	g.sync()
	require.Zero(t, g.client.devCalls().googleCalls)

	// Once connected, the cron registers the company-owned host with no sign-in. A
	// re-enrolled device can leave a second record with the same deviceId.
	g.connectIntegration()
	g.client.setDev(func(d *googleDeviceMock) {
		d.androidRecords = map[string][]string{hostAndroidID: {"and-live", "and-leftover"}}
		d.deviceUsers = map[string][]string{
			"and-live":     {"devices/and-live/deviceUsers/1", "devices/and-live/deviceUsers/2"},
			"and-leftover": {"devices/and-leftover/deviceUsers/1"},
		}
	})
	g.sync()
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/and-live/deviceUsers/1": managed, "devices/and-live/deviceUsers/2": managed, "devices/and-leftover/deviceUsers/1": managed,
	}, g.states(host))
	require.Equal(t, "and-live", g.googleDeviceID(host))
	require.ElementsMatch(t, []string{
		"devices/and-live/deviceUsers/1 " + host.UUID, "devices/and-live/deviceUsers/2 " + host.UUID, "devices/and-leftover/deviceUsers/1 " + host.UUID,
	}, g.client.devCalls().managed)
	require.Equal(t, []string{hostAndroidID}, g.client.devCalls().finds)
	// The activity is automated and names no Google account.
	require.Len(t, g.cronActivities, 1)
	require.Equal(t, fleet.ActivityTypeSetHostGoogleDeviceID{
		HostID: host.ID, HostDisplayName: host.DisplayName(), GoogleDeviceID: "and-live", Automated: true,
	}, g.cronActivities[0])
	require.True(t, g.cronActivities[0].(fleet.ActivityTypeSetHostGoogleDeviceID).WasFromAutomation())
	// A personal host is never resolved.
	require.False(t, g.hasDevice(personal))

	// A registered host isn't searched for again.
	g.sync()
	require.Len(t, g.client.devCalls().finds, 1)

	// A host Google has no record of yet is tried on three runs in a row, then only by
	// the hourly run.
	late := g.newAndroidHost(true)
	lateAndroidID := g.androidDeviceID(late)
	for range 4 {
		g.sync()
	}
	require.Equal(t, []string{hostAndroidID, lateAndroidID, lateAndroidID, lateAndroidID}, g.client.devCalls().finds)
	require.Empty(t, g.states(late))
	require.NoError(t, g.syncer.CheckDevices(t.Context()))
	require.Len(t, g.client.devCalls().finds, 5)

	// A record with a DeviceUser stored on another host leaves the host unregistered.
	g.client.setDev(func(d *googleDeviceMock) { d.androidRecords[lateAndroidID] = []string{"and-live"} })
	require.NoError(t, g.syncer.CheckDevices(t.Context()))
	require.Empty(t, g.states(late))
	require.Empty(t, g.googleDeviceID(late))
	require.Len(t, g.cronActivities, 1)

	// Once its own record appears, the hourly run registers it.
	g.client.setDev(func(d *googleDeviceMock) {
		d.androidRecords[lateAndroidID] = []string{"and-late"}
		d.deviceUsers["and-late"] = []string{"devices/and-late/deviceUsers/1"}
	})
	require.NoError(t, g.syncer.CheckDevices(t.Context()))
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{"devices/and-late/deviceUsers/1": managed}, g.states(late))
	require.Equal(t, "and-late", g.googleDeviceID(late))
	require.Len(t, g.cronActivities, 2)
	require.Zero(t, g.device(late).RegistrationAttempts)

	// A bookmark-era device ID whose record is gone is replaced by the resolved one.
	bookmark := g.newAndroidHost(true)
	g.execSQL(`INSERT INTO google_conditional_access_devices (host_uuid, customer_id, google_device_id) VALUES (?, '', 'bm-dead')`, bookmark.UUID)
	g.client.setDev(func(d *googleDeviceMock) {
		d.goneDevices = map[string]bool{"bm-dead": true}
		d.androidRecords[g.androidDeviceID(bookmark)] = []string{"and-bookmark"}
		d.deviceUsers["and-bookmark"] = []string{"devices/and-bookmark/deviceUsers/1"}
	})
	g.sync()
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{"devices/and-bookmark/deviceUsers/1": managed}, g.states(bookmark))
	require.Equal(t, "and-bookmark", g.googleDeviceID(bookmark))

	// Unenrolling queues the unmark, and the host isn't resolved again.
	_, err := s.ds.SetAndroidHostUnenrolled(t.Context(), host.ID)
	require.NoError(t, err)
	finds := len(g.client.devCalls().finds)
	g.sync()
	require.Empty(t, g.states(host))
	require.Len(t, g.client.devCalls().finds, finds)
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessGoogleCleanup() {
	t := s.T()
	g := s.newGoogleDeviceTest()
	g.connectIntegration()

	// After a customer change the previous customer's rows don't count, the cron deletes
	// them, and a host registering under the new customer takes its row over.
	kept := g.newIOSHost("ios", true)
	takeover := g.newIOSHost("ios", true)
	require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, g.registerNames(g.signIn(kept), "devices/kept/deviceUsers/1"))
	require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, g.registerNames(g.signIn(takeover), "devices/takeover/deviceUsers/1"))
	g.execSQL(`UPDATE google_conditional_access_devices SET customer_id = 'C03previous' WHERE host_uuid IN (?, ?)`, kept.UUID, takeover.UUID)
	require.False(t, g.iosStatus(takeover).Registered)
	require.Empty(t, g.googleDeviceID(kept))
	require.Equal(t, fleet.GoogleConditionalAccessRegisterDone, g.registerNames(g.signIn(takeover), "devices/takeover/deviceUsers/2"))
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/takeover/deviceUsers/2": fleet.GoogleConditionalAccessDeviceUserManaged,
	}, g.states(takeover))
	g.sync()
	require.Empty(t, g.states(kept))
	require.True(t, g.iosStatus(takeover).Registered)

	// DELETE drops the registration, and the cron purges the device rows and the row 30
	// days later, with no Google or fleetdm.com call.
	s.DoJSON("DELETE", "/api/latest/fleet/conditional_access/google", nil, http.StatusOK, &deleteGoogleConditionalAccessResponse{})
	require.Equal(t, fleet.GoogleConditionalAccessDeviceStatus{}, g.iosStatus(takeover))
	// The device is still marked in Google, so host details keep showing its ID.
	require.Equal(t, "takeover", g.googleDeviceID(takeover))
	g.register("secret-"+googleTestCustomer, fleet.GoogleConditionalAccessRegistration{Token: "t"}, http.StatusUnauthorized)
	g.client.reset()
	g.sync()
	require.True(t, g.hasDevice(takeover))
	_, err := s.ds.GetGoogleConditionalAccessIntegration(t.Context())
	require.NoError(t, err)

	g.syncAfter(31 * 24 * time.Hour)
	checks, registers, deregisters, _ := g.client.calls()
	require.Empty(t, checks)
	require.Empty(t, registers)
	require.Empty(t, deregisters)
	_, err = s.ds.GetGoogleConditionalAccessIntegration(t.Context())
	require.True(t, fleet.IsNotFound(err))
	require.False(t, g.hasDevice(takeover))
	require.Zero(t, g.client.devCalls().googleCalls)
}
