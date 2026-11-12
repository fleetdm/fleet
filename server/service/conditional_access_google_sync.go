package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google"
)

const (
	// googleDeletedIntegrationRetention is how long a deleted integration keeps its device
	// rows for a re-add.
	googleDeletedIntegrationRetention = 30 * 24 * time.Hour
	googleSyncBatchSize               = 500
	// googleDeviceRowsDeleteBatch bounds each delete of a previous customer's rows.
	googleDeviceRowsDeleteBatch = 2000
	// googleMarkRetryDelay leaves a just-registered name to the request marking it, and is
	// the first wait after a failed mark.
	googleMarkRetryDelay    = time.Minute
	googleMarkRetryMaxDelay = time.Hour
	// googleBookmarkRetryDelay is how long a bookmark-era host that couldn't be migrated is
	// left alone.
	googleBookmarkRetryDelay = 6 * time.Hour
	// googleAndroidFastAttempts is how many sync runs in a row try an Android host that
	// can't be registered yet. After that only the hourly run tries it.
	googleAndroidFastAttempts = 3
)

// GoogleConditionalAccessSyncer runs the Google conditional access cron jobs.
type GoogleConditionalAccessSyncer struct {
	ds          fleet.Datastore
	clients     *GoogleConditionalAccessClient
	newActivity fleet.NewActivityFunc
	logger      *slog.Logger
	now         func() time.Time
	// bookmarkSkips holds when to try again each bookmark-era host that couldn't be
	// migrated, by host UUID and Google device ID. In memory, so a restart costs one call
	// per host.
	bookmarkSkips map[string]time.Time
}

func NewGoogleConditionalAccessSyncer(
	ds fleet.Datastore, clients *GoogleConditionalAccessClient, newActivity fleet.NewActivityFunc, logger *slog.Logger,
) *GoogleConditionalAccessSyncer {
	return &GoogleConditionalAccessSyncer{
		ds: ds, clients: clients, newActivity: newActivity, logger: logger, now: time.Now, bookmarkSkips: map[string]time.Time{},
	}
}

// googleSyncRun is one run's integration and Cloud Identity client.
type googleSyncRun struct {
	settings    fleet.GoogleConditionalAccessIntegration
	integration *fleet.ConditionalAccessGoogleIntegration
	ci          conditional_access_google.CloudIdentity
}

// Sync unmarks queued names, retries failed marks, migrates bookmark-era hosts, registers
// Android hosts, and cleans up after a customer change or a deletion.
func (s *GoogleConditionalAccessSyncer) Sync(ctx context.Context) error {
	ctx = ctxdb.RequirePrimary(ctx, true)
	integration, err := s.integration(ctx)
	if err != nil || integration == nil {
		return err
	}
	if integration.State == fleet.ConditionalAccessGoogleStateDeleted {
		return s.purgeDeletedIntegration(ctx)
	}
	settings, err := s.settings(ctx)
	if err != nil || settings == nil {
		return err
	}
	if err := s.deleteOtherCustomersDevices(ctx, settings.CustomerID); err != nil {
		return err
	}

	run, err := s.start(ctx, integration, *settings)
	if err != nil || run == nil {
		return err
	}
	resolveAndroidHosts := func(ctx context.Context, run *googleSyncRun) error {
		return s.resolveAndroidHosts(ctx, run, googleAndroidFastAttempts)
	}
	for _, job := range []func(context.Context, *googleSyncRun) error{
		s.unmark, s.retryMarks, s.migrateBookmarkHosts, resolveAndroidHosts,
	} {
		if err := job(ctx, run); err != nil {
			return s.handleDelegationError(ctx, run, err)
		}
	}
	return nil
}

// CheckDevices deletes the names whose Google device is gone and sets stale those whose
// device left the Workspace, so their host registers again on its next open. It also
// tries the Android hosts the sync cron gave up on.
func (s *GoogleConditionalAccessSyncer) CheckDevices(ctx context.Context) error {
	ctx = ctxdb.RequirePrimary(ctx, true)
	integration, err := s.integration(ctx)
	if err != nil || integration == nil || integration.State == fleet.ConditionalAccessGoogleStateDeleted {
		return err
	}
	settings, err := s.settings(ctx)
	if err != nil || settings == nil {
		return err
	}
	run, err := s.start(ctx, integration, *settings)
	if err != nil || run == nil {
		return err
	}

	type deviceCheck struct {
		found bool
		// staleErr is set when the device left the customer's Workspace.
		staleErr error
		// failed skips the device's other names for this run.
		failed bool
	}
	checked := make(map[string]deviceCheck)
	err = s.eachDeviceUser(ctx, run, []fleet.GoogleConditionalAccessDeviceUserState{
		fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserManaged,
	}, false, func(du fleet.GoogleConditionalAccessDeviceUser) error {
		device := googleDeviceOf(du.Name)
		check, ok := checked[device]
		if !ok {
			found, err := run.ci.DeviceExists(ctx, device)
			switch {
			case conditional_access_google.IsAuthError(err):
				return err
			case conditional_access_google.IsStale(err):
				check.staleErr = err
			case err != nil:
				s.logger.WarnContext(ctx, "check google device", "device_user_name", du.Name, "err", err)
				check.failed = true
			}
			check.found = found
			checked[device] = check
		}
		switch {
		case check.failed:
			return nil
		case check.staleErr != nil:
			s.logger.InfoContext(ctx, "google device left the workspace, the host signs in again", "device_user_name", du.Name, "err", check.staleErr)
			if _, err := s.ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, du.Name, du.State,
				fleet.GoogleConditionalAccessDeviceUserStale, new(check.staleErr.Error())); err != nil {
				return ctxerr.Wrap(ctx, err, "set google device user stale")
			}
			return nil
		case check.found:
			return nil
		}
		s.logger.InfoContext(ctx, "google device gone, deleting its device user", "device_user_name", du.Name)
		return s.ds.DeleteGoogleConditionalAccessDeviceUser(ctx, du.Name, du.State, false)
	})
	if err == nil {
		err = s.resolveAndroidHosts(ctx, run, 0)
	}
	if err != nil {
		return s.handleDelegationError(ctx, run, err)
	}
	return nil
}

func (s *GoogleConditionalAccessSyncer) integration(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
	integration, err := s.ds.GetGoogleConditionalAccessIntegration(ctx)
	if fleet.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load google conditional access integration")
	}
	return integration, nil
}

// settings returns the customer ID and admin email from the app config, nil when the
// integration is off.
func (s *GoogleConditionalAccessSyncer) settings(ctx context.Context) (*fleet.GoogleConditionalAccessIntegration, error) {
	appConfig, err := s.ds.AppConfig(ctxdb.BypassCachedMysql(ctx, true))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load app config")
	}
	return appConfig.Integrations.GoogleConditionalAccess, nil
}

// start returns nil when the integration can't call Google: it isn't connected, or its
// key is gone. An integration fleetdm.com never registered is left to the admin's next
// connect, or recovering its delegation would show it connected.
func (s *GoogleConditionalAccessSyncer) start(
	ctx context.Context, integration *fleet.ConditionalAccessGoogleIntegration, settings fleet.GoogleConditionalAccessIntegration,
) (*googleSyncRun, error) {
	if !integration.Registered() || (integration.State != fleet.ConditionalAccessGoogleStateConnected &&
		integration.State != fleet.ConditionalAccessGoogleStateDelegationFailed) {
		return nil, nil
	}
	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey}, nil)
	if fleet.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load google conditional access service account key")
	}
	key := assets[fleet.MDMAssetGoogleConditionalAccessServiceAccountKey].Value
	if len(key) == 0 {
		return nil, nil
	}
	ci, err := s.clients.CloudIdentity(ctx, conditional_access_google.Credentials{
		ServiceAccountKey: key, AdminEmail: settings.AdminEmail, CustomerID: settings.CustomerID,
	})
	if err != nil {
		return nil, s.setDelegationFailed(ctx, integration, err)
	}
	run := &googleSyncRun{settings: settings, integration: integration, ci: ci}
	// One cheap call tells a broken delegation from a broken device.
	if err := ci.CheckDelegation(ctx); err != nil {
		if conditional_access_google.IsAuthError(err) {
			return nil, s.setDelegationFailed(ctx, integration, err)
		}
		return nil, ctxerr.Wrap(ctx, err, "check google delegation")
	}
	if integration.State == fleet.ConditionalAccessGoogleStateDelegationFailed {
		if _, err := s.ds.SetGoogleConditionalAccessIntegrationState(ctx, integration.ProxyServerSecret,
			[]fleet.ConditionalAccessGoogleState{fleet.ConditionalAccessGoogleStateDelegationFailed},
			fleet.ConditionalAccessGoogleStateConnected, nil); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "set google conditional access connected")
		}
	}
	return run, nil
}

func (s *GoogleConditionalAccessSyncer) handleDelegationError(ctx context.Context, run *googleSyncRun, err error) error {
	if conditional_access_google.IsAuthError(err) {
		return s.setDelegationFailed(ctx, run.integration, err)
	}
	return err
}

// setDelegationFailed records the failure on the integration for the admin and returns
// it, so the run is reported as failed.
func (s *GoogleConditionalAccessSyncer) setDelegationFailed(ctx context.Context, integration *fleet.ConditionalAccessGoogleIntegration, cause error) error {
	msg := "Couldn't connect to Google: " + cause.Error()
	// Only the same registration, still connected, is changed, so an admin's concurrent
	// change wins.
	if _, err := s.ds.SetGoogleConditionalAccessIntegrationState(ctx, integration.ProxyServerSecret,
		[]fleet.ConditionalAccessGoogleState{fleet.ConditionalAccessGoogleStateConnected, fleet.ConditionalAccessGoogleStateDelegationFailed},
		fleet.ConditionalAccessGoogleStateDelegationFailed, &msg); err != nil {
		return ctxerr.Wrap(ctx, err, "set google conditional access delegation failed")
	}
	return ctxerr.Wrap(ctx, cause, "google conditional access delegation failed")
}

func (s *GoogleConditionalAccessSyncer) eachDeviceUser(
	ctx context.Context, run *googleSyncRun, states []fleet.GoogleConditionalAccessDeviceUserState, hostExists bool,
	fn func(fleet.GoogleConditionalAccessDeviceUser) error,
) error {
	after := ""
	for {
		dus, err := s.ds.ListGoogleConditionalAccessDeviceUsers(ctx, fleet.GoogleConditionalAccessDeviceUserFilter{
			CustomerID: run.settings.CustomerID, States: states, HostExists: hostExists, AfterName: after, Limit: googleSyncBatchSize,
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list google device users")
		}
		for _, du := range dus {
			if err := fn(du); err != nil {
				return err
			}
		}
		if len(dus) < googleSyncBatchSize {
			return nil
		}
		after = dus[len(dus)-1].Name
	}
}

func (s *GoogleConditionalAccessSyncer) unmark(ctx context.Context, run *googleSyncRun) error {
	unmanaging := fleet.GoogleConditionalAccessDeviceUserUnmanaging
	return s.eachDeviceUser(ctx, run, []fleet.GoogleConditionalAccessDeviceUserState{unmanaging}, false,
		func(du fleet.GoogleConditionalAccessDeviceUser) error {
			err := run.ci.SetClientState(ctx, du.Name, fleet.GoogleConditionalAccessDesiredClientState(du))
			switch {
			case conditional_access_google.IsAuthError(err):
				return err
			case err != nil && !conditional_access_google.IsStale(err):
				msg := err.Error()
				if _, err := s.ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, du.Name, unmanaging, unmanaging, &msg); err != nil {
					return ctxerr.Wrap(ctx, err, "record google unmark error")
				}
				return nil
			}
			// A record Google no longer has needs no unmark.
			return s.ds.DeleteGoogleConditionalAccessDeviceUser(ctx, du.Name, unmanaging, true)
		})
}

func (s *GoogleConditionalAccessSyncer) retryMarks(ctx context.Context, run *googleSyncRun) error {
	return s.eachDeviceUser(ctx, run, []fleet.GoogleConditionalAccessDeviceUserState{fleet.GoogleConditionalAccessDeviceUserPending}, true,
		func(du fleet.GoogleConditionalAccessDeviceUser) error {
			if !googleMarkRetryDue(du, s.now()) {
				return nil
			}
			stale, err := markGoogleDeviceUser(ctx, s.ds, run.ci, du.Name, du.HostUUID)
			switch {
			case conditional_access_google.IsAuthError(err):
				return err
			case stale:
				s.logger.InfoContext(ctx, "google refused a stored device user, the host signs in again", "device_user_name", du.Name, "err", err)
				if _, err := s.ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, du.Name,
					fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserStale, new(err.Error())); err != nil {
					return ctxerr.Wrap(ctx, err, "set google device user stale")
				}
			case err != nil:
				s.logger.WarnContext(ctx, "mark google device user", "device_user_name", du.Name, "err", err)
			}
			return nil
		})
}

// migrateBookmarkHosts stores and marks the DeviceUsers of hosts that only have a Google
// device ID from the bookmark flow, so their users don't have to sign in again.
func (s *GoogleConditionalAccessSyncer) migrateBookmarkHosts(ctx context.Context, run *googleSyncRun) error {
	customerID := run.settings.CustomerID
	now := s.now()
	for k, retryAt := range s.bookmarkSkips {
		if !now.Before(retryAt) {
			delete(s.bookmarkSkips, k)
		}
	}
	var afterHostID uint
	for {
		hosts, err := s.ds.ListGoogleConditionalAccessBookmarkHosts(ctx, customerID, afterHostID, googleSyncBatchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list bookmark-era google hosts")
		}
		for _, h := range hosts {
			if err := s.migrateBookmarkHost(ctx, run, h); err != nil {
				return err
			}
		}
		if len(hosts) < googleSyncBatchSize {
			return nil
		}
		afterHostID = hosts[len(hosts)-1].HostID
	}
}

func (s *GoogleConditionalAccessSyncer) migrateBookmarkHost(ctx context.Context, run *googleSyncRun, h fleet.GoogleConditionalAccessBookmarkHost) error {
	skipKey := h.HostUUID + " " + h.GoogleDeviceID
	if _, ok := s.bookmarkSkips[skipKey]; ok {
		return nil
	}
	// A host that can't be migrated is left to the sign-in flow.
	skip := func() { s.bookmarkSkips[skipKey] = s.now().Add(googleBookmarkRetryDelay) }
	names, err := run.ci.ListDeviceUserNames(ctx, h.GoogleDeviceID)
	switch {
	case conditional_access_google.IsAuthError(err):
		return err
	case conditional_access_google.IsStale(err), err == nil && len(names) == 0:
		skip()
		return nil
	case err != nil:
		s.logger.DebugContext(ctx, "list google device users of bookmark-era host", "host_uuid", h.HostUUID, "err", err)
		return nil
	}
	recorded, err := s.recordNames(ctx, run, h.HostUUID, names)
	if err != nil {
		return err
	}
	if !recorded {
		skip()
		return nil
	}
	return s.markNames(ctx, run, h.HostUUID, names)
}

// recordNames stores the names the cron found for a host. It reports false, and stores
// nothing, when one of them is stored on another host.
func (s *GoogleConditionalAccessSyncer) recordNames(ctx context.Context, run *googleSyncRun, hostUUID string, names []string) (bool, error) {
	err := s.ds.RecordGoogleConditionalAccessRegistration(ctx, hostUUID, run.settings.CustomerID, names)
	if conflict, ok := errors.AsType[*fleet.GoogleConditionalAccessDeviceUserConflictError](err); ok {
		s.logger.WarnContext(ctx, "google device user registered to another host",
			"host_uuid", hostUUID, "device_user_name", conflict.Name)
		return false, nil
	}
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "record google registration")
	}
	return true, nil
}

// markNames marks the names just stored on a host. A failed mark stays pending with its
// error and is retried by a later run.
func (s *GoogleConditionalAccessSyncer) markNames(ctx context.Context, run *googleSyncRun, hostUUID string, names []string) error {
	for _, name := range names {
		if _, err := markGoogleDeviceUser(ctx, s.ds, run.ci, name, hostUUID); err != nil {
			if conditional_access_google.IsAuthError(err) {
				return err
			}
			s.logger.WarnContext(ctx, "mark google device user", "host_uuid", hostUUID, "device_user_name", name, "err", err)
		}
	}
	return nil
}

// resolveAndroidHosts registers company-owned Android hosts without a sign-in: the
// Android Management API device ID Fleet stores for them is the deviceId of their record
// in Google. A maxAttempts above 0 leaves out the hosts that failed that many times.
func (s *GoogleConditionalAccessSyncer) resolveAndroidHosts(ctx context.Context, run *googleSyncRun, maxAttempts int) error {
	hosts, err := s.ds.ListGoogleConditionalAccessUnresolvedAndroidHosts(ctx, run.settings.CustomerID, maxAttempts, googleSyncBatchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list unresolved android google hosts")
	}
	for _, h := range hosts {
		if err := s.resolveAndroidHost(ctx, run, h); err != nil {
			return err
		}
	}
	return nil
}

func (s *GoogleConditionalAccessSyncer) resolveAndroidHost(ctx context.Context, run *googleSyncRun, h fleet.GoogleConditionalAccessAndroidHost) error {
	customerID := run.settings.CustomerID
	// Google creates the record when the managed account first checks in, which can be
	// after the enrollment, so a host that can't be registered is counted and tried again.
	countAttempt := func() error {
		if err := s.ds.CountGoogleConditionalAccessResolutionAttempt(ctx, h.HostUUID, customerID); err != nil {
			return ctxerr.Wrap(ctx, err, "count android google resolution attempt")
		}
		return nil
	}

	deviceIDs, err := run.ci.FindDeviceIDs(ctx, h.AndroidDeviceID)
	if conditional_access_google.IsAuthError(err) {
		return err
	}
	var names []string
	for _, deviceID := range deviceIDs {
		deviceNames, listErr := run.ci.ListDeviceUserNames(ctx, deviceID)
		if conditional_access_google.IsAuthError(listErr) {
			return listErr
		}
		// A record deleted since the search has no names to add.
		if listErr != nil && !conditional_access_google.IsStale(listErr) {
			err = listErr
			break
		}
		names = append(names, deviceNames...)
	}
	if err != nil {
		s.logger.DebugContext(ctx, "find google device users of android host", "host_uuid", h.HostUUID, "err", err)
		return countAttempt()
	}
	if names = dedupeDeviceUserNames(names); len(names) == 0 {
		return countAttempt()
	}

	recorded, err := s.recordNames(ctx, run, h.HostUUID, names)
	if err != nil {
		return err
	}
	if !recorded {
		return countAttempt()
	}
	s.logAndroidRegistration(ctx, h, customerID)
	return s.markNames(ctx, run, h.HostUUID, names)
}

// logAndroidRegistration logs the activity of a registration. A failure doesn't stop the
// marking, which is what lets the host in.
func (s *GoogleConditionalAccessSyncer) logAndroidRegistration(ctx context.Context, h fleet.GoogleConditionalAccessAndroidHost, customerID string) {
	host, err := s.ds.HostLite(ctx, h.HostID)
	if err != nil {
		s.logger.ErrorContext(ctx, "get android host for google registration activity", "host_uuid", h.HostUUID, "err", err)
		return
	}
	googleDeviceID, err := s.ds.GetHostGoogleDeviceID(ctx, h.HostUUID, customerID)
	if err != nil {
		s.logger.ErrorContext(ctx, "get android host's google device id", "host_uuid", h.HostUUID, "err", err)
		return
	}
	if err := s.newActivity(ctx, nil, fleet.ActivityTypeSetHostGoogleDeviceID{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		GoogleDeviceID:  googleDeviceID,
		Automated:       true,
	}); err != nil {
		s.logger.ErrorContext(ctx, "create activity for android google registration", "host_uuid", h.HostUUID, "err", err)
	}
}

func (s *GoogleConditionalAccessSyncer) deleteOtherCustomersDevices(ctx context.Context, customerID string) error {
	for {
		n, err := s.ds.DeleteGoogleConditionalAccessDevicesExcept(ctx, customerID, googleDeviceRowsDeleteBatch)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete previous customer's google device rows")
		}
		if n < googleDeviceRowsDeleteBatch {
			return nil
		}
	}
}

// purgeDeletedIntegration deletes the device rows and the row of an integration deleted
// long enough ago. Its registration was dropped by the deletion, and Google is left as it
// is.
func (s *GoogleConditionalAccessSyncer) purgeDeletedIntegration(ctx context.Context) error {
	deletedBefore := s.now().Add(-googleDeletedIntegrationRetention)
	for {
		more, err := s.ds.PurgeDeletedGoogleConditionalAccessIntegration(ctx, deletedBefore, googleDeviceRowsDeleteBatch)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "purge deleted google conditional access integration")
		}
		if !more {
			return nil
		}
	}
}

// googleMarkRetryDue reports whether the cron should mark a pending name. A name just
// registered is left to the request for a minute. A failed one waits as long as it has
// been registered, so each failure doubles the wait, up to an hour.
func googleMarkRetryDue(du fleet.GoogleConditionalAccessDeviceUser, now time.Time) bool {
	wait := googleMarkRetryDelay
	if du.ErrorMessage != nil {
		wait = min(max(du.UpdatedAt.Sub(du.CreatedAt), googleMarkRetryDelay), googleMarkRetryMaxDelay)
	}
	return now.Sub(du.UpdatedAt) >= wait
}

func googleDeviceOf(deviceUserName string) string {
	device, _, _ := strings.Cut(strings.TrimPrefix(deviceUserName, "devices/"), "/")
	return device
}
