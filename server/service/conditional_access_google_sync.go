package service

import (
	"context"
	"errors"
	"log/slog"
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
	// can't be registered yet. After that only the hourly RetryAndroidHosts tries it.
	googleAndroidFastAttempts = 3
)

// errGoogleRunSuperseded stops a run whose integration was removed or replaced since it
// started, so it doesn't keep writing to Google with the old credentials.
var errGoogleRunSuperseded = errors.New("google conditional access integration changed during the run")

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

// googleSyncRun is one run's customer, integration and Cloud Identity client.
type googleSyncRun struct {
	customerID  string
	integration *fleet.ConditionalAccessGoogleIntegration
	ci          conditional_access_google.CloudIdentity
	// delegationChecked is set once the delegation check passed in this run.
	delegationChecked bool
}

// Sync unmarks queued names and those of unenrolled hosts, retries failed marks, migrates
// bookmark-era hosts, registers Android hosts, and cleans up after a customer change or a
// deletion.
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
	if err := s.queueUnmarkOfUnenrolledHosts(ctx, settings.CustomerID); err != nil {
		return err
	}

	run, err := s.start(ctx, integration, *settings)
	if err != nil || run == nil {
		return err
	}
	resolveAndroidHosts := func(ctx context.Context, run *googleSyncRun) error {
		return s.resolveAndroidHosts(ctx, run, 0, googleAndroidFastAttempts)
	}
	for i, job := range []func(context.Context, *googleSyncRun) error{
		s.unmark, s.retryMarks, s.migrateBookmarkHosts, resolveAndroidHosts,
	} {
		err := s.checkCurrent(ctx, run, i > 0)
		if err == nil {
			err = job(ctx, run)
		}
		if err != nil {
			return s.handleRunError(ctx, run, err)
		}
	}
	return nil
}

// RetryAndroidHosts tries the Android hosts the sync cron stopped trying, which the sync
// cron never lists, so the two don't work on the same host.
func (s *GoogleConditionalAccessSyncer) RetryAndroidHosts(ctx context.Context) error {
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
	if err := s.resolveAndroidHosts(ctx, run, googleAndroidFastAttempts, 0); err != nil {
		return s.handleRunError(ctx, run, err)
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
	run := &googleSyncRun{customerID: settings.CustomerID, integration: integration, ci: ci}
	// A connected integration is checked only when a device call needs the answer, so a
	// run with nothing to do doesn't call Google. One that shows the failure is checked on
	// every run, so it shows connected again once Google accepts it.
	if integration.State == fleet.ConditionalAccessGoogleStateDelegationFailed {
		if err := s.checkDelegation(ctx, run); err != nil {
			if conditional_access_google.IsAuthError(err) {
				return nil, s.setDelegationFailed(ctx, integration, err)
			}
			return nil, ctxerr.Wrap(ctx, err, "check google delegation")
		}
		if _, err := s.ds.SetGoogleConditionalAccessIntegrationState(ctx, integration.ProxyServerSecret,
			[]fleet.ConditionalAccessGoogleState{fleet.ConditionalAccessGoogleStateDelegationFailed},
			fleet.ConditionalAccessGoogleStateConnected, nil); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "set google conditional access connected")
		}
		integration.State = fleet.ConditionalAccessGoogleStateConnected
	}
	return run, nil
}

// checkDelegation runs the delegation check at most once per run. Besides recovering a
// failed integration, it decides what a 403 naming a device means: Google answers that
// for a device that left the Workspace, but also for every device once the impersonated
// admin loses its device privileges. Its error is an auth error or an outage, both of
// which end the run.
func (s *GoogleConditionalAccessSyncer) checkDelegation(ctx context.Context, run *googleSyncRun) error {
	if run.delegationChecked {
		return nil
	}
	if err := run.ci.CheckDelegation(ctx); err != nil {
		return err
	}
	run.delegationChecked = true
	return nil
}

// googleStopsRun reports whether err ends the run: Google refused the delegation, which
// every later call would hit too, or is unavailable, which the next run retries.
func googleStopsRun(err error) bool {
	return conditional_access_google.IsAuthError(err) || conditional_access_google.IsUnavailable(err)
}

// checkCurrent returns errGoogleRunSuperseded when the run's registration is no longer
// the integration's: an admin removed the integration or changed its customer. It reads
// nothing when check is false.
func (s *GoogleConditionalAccessSyncer) checkCurrent(ctx context.Context, run *googleSyncRun, check bool) error {
	if !check {
		return nil
	}
	integration, err := s.integration(ctx)
	if err != nil {
		return err
	}
	if integration == nil || integration.State == fleet.ConditionalAccessGoogleStateDeleted ||
		integration.ProxyServerSecret != run.integration.ProxyServerSecret {
		return errGoogleRunSuperseded
	}
	return nil
}

func (s *GoogleConditionalAccessSyncer) handleRunError(ctx context.Context, run *googleSyncRun, err error) error {
	if errors.Is(err, errGoogleRunSuperseded) {
		s.logger.InfoContext(ctx, "google conditional access integration changed, stopping the run")
		return nil
	}
	if conditional_access_google.IsAuthError(err) {
		return s.setDelegationFailed(ctx, run.integration, err)
	}
	return ctxerr.Wrap(ctx, err, "google conditional access run")
}

// setDelegationFailed records the failure on the integration for the admin. The run is
// reported as failed the first time only: the integration shows it after that.
func (s *GoogleConditionalAccessSyncer) setDelegationFailed(ctx context.Context, integration *fleet.ConditionalAccessGoogleIntegration, cause error) error {
	msg := "Couldn't connect to Google: " + cause.Error()
	// Only the same registration, still connected, is changed, so an admin's concurrent
	// change wins.
	if _, err := s.ds.SetGoogleConditionalAccessIntegrationState(ctx, integration.ProxyServerSecret,
		[]fleet.ConditionalAccessGoogleState{fleet.ConditionalAccessGoogleStateConnected, fleet.ConditionalAccessGoogleStateDelegationFailed},
		fleet.ConditionalAccessGoogleStateDelegationFailed, &msg); err != nil {
		return ctxerr.Wrap(ctx, err, "set google conditional access delegation failed")
	}
	if integration.State == fleet.ConditionalAccessGoogleStateDelegationFailed {
		s.logger.WarnContext(ctx, "google conditional access delegation still failing", "err", cause)
		return nil
	}
	return ctxerr.Wrap(ctx, cause, "google conditional access delegation failed")
}

// googleNameGone reports whether the error of a write to a name means Google no longer
// has the name for the customer. A 404 does. A 403 is also what a service account that
// can't write gets, so it counts only when Google can't read the name's device either.
func (s *GoogleConditionalAccessSyncer) googleNameGone(ctx context.Context, run *googleSyncRun, name string, writeErr error) (bool, error) {
	if !conditional_access_google.IsStale(writeErr) {
		return false, nil
	}
	if fleet.IsNotFound(writeErr) {
		return true, nil
	}
	found, err := run.ci.DeviceExists(ctx, fleet.GoogleDeviceIDFromDeviceUserName(name))
	switch {
	case googleStopsRun(err):
		return false, err
	case conditional_access_google.IsStale(err):
		if err := s.checkDelegation(ctx, run); err != nil {
			return false, err
		}
		return true, nil
	case err != nil:
		s.logger.WarnContext(ctx, "check google device of a refused write", "device_user_name", name, "err", err)
		return false, nil
	case found:
		s.logger.WarnContext(ctx, "google refused a write on a device it lets Fleet read, check the service account's admin role",
			"device_user_name", name, "err", writeErr)
	}
	return !found, nil
}

func (s *GoogleConditionalAccessSyncer) eachDeviceUser(
	ctx context.Context, run *googleSyncRun, states []fleet.GoogleConditionalAccessDeviceUserState, hostExists bool,
	fn func(fleet.GoogleConditionalAccessDeviceUser) error,
) error {
	after := ""
	for {
		if err := s.checkCurrent(ctx, run, after != ""); err != nil {
			return err
		}
		dus, err := s.ds.ListGoogleConditionalAccessDeviceUsers(ctx, fleet.GoogleConditionalAccessDeviceUserFilter{
			CustomerID: run.customerID, States: states, HostExists: hostExists, AfterName: after, Limit: googleSyncBatchSize,
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
	now := s.now()
	return s.eachDeviceUser(ctx, run, []fleet.GoogleConditionalAccessDeviceUserState{unmanaging}, false,
		func(du fleet.GoogleConditionalAccessDeviceUser) error {
			if !googleUnmarkRetryDue(du, now) {
				return nil
			}
			writeErr := run.ci.SetClientState(ctx, du.Name, fleet.GoogleConditionalAccessDesiredClientState(du))
			if googleStopsRun(writeErr) {
				return writeErr
			}
			// A record Google no longer has needs no unmark.
			gone, err := s.googleNameGone(ctx, run, du.Name, writeErr)
			if err != nil {
				return err
			}
			if writeErr != nil && !gone {
				msg := writeErr.Error()
				if _, err := s.ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, du.Name, unmanaging, unmanaging, &msg); err != nil {
					return ctxerr.Wrap(ctx, err, "record google unmark error")
				}
				return nil
			}
			deleted, err := s.ds.DeleteGoogleConditionalAccessUnmarkedDeviceUser(ctx, du.Name)
			if err != nil || deleted {
				return ctxerr.Wrap(ctx, err, "delete unmarked google device user")
			}
			return s.markRegisteredAgain(ctx, run, du)
		})
}

// markRegisteredAgain marks a name its host registered again while the cron unmarked it:
// the registration's MANAGED write may have come before the cron's UNMANAGED one. A
// name still pending is left to the registration's own mark and to retryMarks.
func (s *GoogleConditionalAccessSyncer) markRegisteredAgain(ctx context.Context, run *googleSyncRun, du fleet.GoogleConditionalAccessDeviceUser) error {
	repending, err := s.ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, du.Name,
		fleet.GoogleConditionalAccessDeviceUserManaged, fleet.GoogleConditionalAccessDeviceUserPending, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set re-registered google device user pending")
	}
	if !repending {
		return nil
	}
	if err := markGoogleDeviceUser(ctx, s.ds, run.ci, du.Name, du.HostUUID); err != nil {
		if googleStopsRun(err) {
			return err
		}
		s.logger.WarnContext(ctx, "mark re-registered google device user", "device_user_name", du.Name, "err", err)
	}
	return nil
}

func (s *GoogleConditionalAccessSyncer) retryMarks(ctx context.Context, run *googleSyncRun) error {
	return s.eachDeviceUser(ctx, run, []fleet.GoogleConditionalAccessDeviceUserState{fleet.GoogleConditionalAccessDeviceUserPending}, true,
		func(du fleet.GoogleConditionalAccessDeviceUser) error {
			if !googleMarkRetryDue(du, s.now()) {
				return nil
			}
			markErr := markGoogleDeviceUser(ctx, s.ds, run.ci, du.Name, du.HostUUID)
			if markErr == nil {
				return nil
			}
			if googleStopsRun(markErr) {
				return markErr
			}
			gone, err := s.googleNameGone(ctx, run, du.Name, markErr)
			if err != nil {
				return err
			}
			if !gone {
				s.logger.WarnContext(ctx, "mark google device user", "device_user_name", du.Name, "err", markErr)
				return nil
			}
			s.logger.InfoContext(ctx, "google refused a stored device user, the host signs in again", "device_user_name", du.Name, "err", markErr)
			if _, err := s.ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, du.Name,
				fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserStale, new(markErr.Error())); err != nil {
				return ctxerr.Wrap(ctx, err, "set google device user stale")
			}
			return nil
		})
}

// migrateBookmarkHosts stores and marks the DeviceUsers of hosts that only have a Google
// device ID from the bookmark flow, so their users don't have to sign in again.
func (s *GoogleConditionalAccessSyncer) migrateBookmarkHosts(ctx context.Context, run *googleSyncRun) error {
	customerID := run.customerID
	now := s.now()
	for k, retryAt := range s.bookmarkSkips {
		if !now.Before(retryAt) {
			delete(s.bookmarkSkips, k)
		}
	}
	var afterHostID uint
	for {
		if err := s.checkCurrent(ctx, run, afterHostID != 0); err != nil {
			return err
		}
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
	if googleStopsRun(err) {
		return err
	}
	if conditional_access_google.IsStale(err) && !fleet.IsNotFound(err) {
		if err := s.checkDelegation(ctx, run); err != nil {
			return err
		}
	}
	if err != nil || len(names) == 0 {
		if err != nil && !conditional_access_google.IsStale(err) {
			s.logger.WarnContext(ctx, "list google device users of bookmark-era host", "host_uuid", h.HostUUID, "err", err)
		}
		skip()
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
	err := s.ds.RecordGoogleConditionalAccessRegistration(ctx, hostUUID, run.customerID, names)
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
		if err := markGoogleDeviceUser(ctx, s.ds, run.ci, name, hostUUID); err != nil {
			if googleStopsRun(err) {
				return err
			}
			s.logger.WarnContext(ctx, "mark google device user", "host_uuid", hostUUID, "device_user_name", name, "err", err)
		}
	}
	return nil
}

// resolveAndroidHosts registers company-owned Android hosts without a sign-in: the
// Android Management API device ID Fleet stores for them is the deviceId of their record
// in Google. It takes the hosts that failed at least minAttempts times and, when
// maxAttempts is above 0, fewer than maxAttempts times.
func (s *GoogleConditionalAccessSyncer) resolveAndroidHosts(ctx context.Context, run *googleSyncRun, minAttempts, maxAttempts int) error {
	hosts, err := s.ds.ListGoogleConditionalAccessUnresolvedAndroidHosts(ctx, run.customerID, minAttempts, maxAttempts, googleSyncBatchSize)
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
	customerID := run.customerID
	// Google creates the record when the managed account first checks in, which can be
	// after the enrollment, so a host that can't be registered is counted and tried again.
	countAttempt := func() error {
		if err := s.ds.CountGoogleConditionalAccessResolutionAttempt(ctx, h.HostUUID, customerID); err != nil {
			return ctxerr.Wrap(ctx, err, "count android google resolution attempt")
		}
		return nil
	}

	deviceIDs, err := run.ci.FindDeviceIDs(ctx, h.AndroidDeviceID)
	if googleStopsRun(err) {
		return err
	}
	var names []string
	for _, deviceID := range deviceIDs {
		deviceNames, listErr := run.ci.ListDeviceUserNames(ctx, deviceID)
		if googleStopsRun(listErr) {
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
		s.logger.WarnContext(ctx, "find google device users of android host", "host_uuid", h.HostUUID, "err", err)
		return countAttempt()
	}
	if names = dedupeDeviceUserNames(names); len(names) == 0 {
		return countAttempt()
	}

	previousID, err := s.ds.GetHostGoogleDeviceID(ctx, h.HostUUID, customerID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android host's google device id")
	}
	recorded, err := s.recordNames(ctx, run, h.HostUUID, names)
	if err != nil {
		return err
	}
	if !recorded {
		return countAttempt()
	}
	s.logAndroidRegistration(ctx, h, customerID, previousID)
	return s.markNames(ctx, run, h.HostUUID, names)
}

// logAndroidRegistration logs the activity of a registration that changed the host's
// Google device ID: a host whose names went stale is registered again on the same device.
// A failure doesn't stop the marking, which is what lets the host in.
func (s *GoogleConditionalAccessSyncer) logAndroidRegistration(ctx context.Context, h fleet.GoogleConditionalAccessAndroidHost, customerID, previousID string) {
	googleDeviceID, err := s.ds.GetHostGoogleDeviceID(ctx, h.HostUUID, customerID)
	if err != nil {
		s.logger.ErrorContext(ctx, "get android host's google device id", "host_uuid", h.HostUUID, "err", err)
		return
	}
	if googleDeviceID == previousID {
		return
	}
	host, err := s.ds.HostLite(ctx, h.HostID)
	if err != nil {
		s.logger.ErrorContext(ctx, "get android host for google registration activity", "host_uuid", h.HostUUID, "err", err)
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

func (s *GoogleConditionalAccessSyncer) queueUnmarkOfUnenrolledHosts(ctx context.Context, customerID string) error {
	for {
		n, err := s.ds.QueueGoogleConditionalAccessUnmarkOfUnenrolledHosts(ctx, customerID, googleSyncBatchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queue google unmark of unenrolled hosts")
		}
		if n < googleSyncBatchSize {
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

// googleUnmarkRetryDue reports whether the cron should unmark a queued name: at once,
// and after a failure with the backoff of a failed mark, counted from the queueing.
func googleUnmarkRetryDue(du fleet.GoogleConditionalAccessDeviceUser, now time.Time) bool {
	return du.ErrorMessage == nil || googleMarkRetryDue(du, now)
}
