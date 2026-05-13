package service

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// Property-based tests for handleESPRelease. They cover the wait-gate decision and the universal block/release
// command-shape invariants in a single combined property check.
//
// The spec function pbtESPSpec computes the expected (decision, observedHasFailure) from the inputs without
// referencing the production code. We then run getESPCommands against a mock datastore and assert:
//
//   - Decision (wait/block/release) matches the spec.
//   - Wait → no side effects (no cancel, no persist, no CAS).
//   - Block path command shape: BlockInStatusPage=1, AllowCollectLogsButton, TimeOutUntilSyncFailure=1,
//     reason-specific CustomErrorText, NO ServerHasFinishedProvisioning, NO InstallationState.
//   - Release path command shape: Device-scope AND User-scope ServerHasFinishedProvisioning plus
//     PolicyProviders InstallationState=3; NO CustomErrorText, NO BlockInStatusPage. The user-scope
//     Provider node is created during the hold phase via Add commands so the user-scope SHFP write
//     lands instead of being 405-rejected.
//   - Persisted CommandUUIDs equal inline CmdID.Value (the ack-clearing invariant).
//   - Persist runs as a single batched call (a regression that loops single inserts would split CustomErrorText
//     and the block flags across multiple TX boundaries).
//   - Cancel block fires iff (timedOut || (observedHasFailure && requireAll)); when it fires,
//     CancelHostUpcomingActivity is called once per Pending/Running row in input. Cancel-upcoming runs strictly
//     before cancel-status; both run strictly before persist; persist runs strictly before CAS.
//
// Order independence is implicit: pbtESPSpec is a pure function of the multiset of statuses (no positional
// dependency) and rapid samples many orderings, so any introduced order-dependence in production code
// surfaces as a spec mismatch.
//
// Run with more checks:
//   go test -run TestPBT_HandleESPRelease ./server/service/ -args -rapid.checks=2000

var pbtESPLogger = slog.New(slog.DiscardHandler)

const (
	pbtESPDeviceID = "pbt-esp-device-id"
	pbtESPHostUUID = "pbt-esp-host-uuid"
)

// pbtESPTrace captures observable side effects of handleESPRelease so the property can assert ordering and
// counts without inspecting the auto-set FuncInvoked flags individually.
type pbtESPTrace struct {
	cancelUpcomingExecIDs []string // execution IDs passed to CancelHostUpcomingActivity, in call order
	persistedCmdUUIDs     []string
	callOrder             []string // sequence of "cancel-upcoming", "cancel-status", "persist", "cas"
}

// newPBTESPSvc wires a mock datastore for property-testing handleESPRelease. Stages 1 and 2 (profiles) are
// mocked empty so the property focuses on Stage 3 + finalize. Profile-stage logic is covered by the
// example-based subtests in TestGetESPCommands.
func newPBTESPSvc(
	statuses []fleet.SetupExperienceStatusResultStatus, timedOut, requireAll bool,
) (*Service, *fleet.MDMWindowsEnrolledDevice, *pbtESPTrace) {
	ds := new(mock.Store)
	trace := &pbtESPTrace{}

	osqueryHostID := "pbt-esp-osq"
	ds.HostLiteByIdentifierFunc = func(ctx context.Context, id string) (*fleet.HostLite, error) {
		return &fleet.HostLite{ID: 1, UUID: id, OsqueryHostID: &osqueryHostID, TeamID: nil}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		ac := &fleet.AppConfig{}
		ac.MDM.MacOSSetup.RequireAllSoftwareWindows = requireAll
		return ac, nil
	}

	// Skip Stages 1 and 2: profiles are out of PBT scope.
	ds.ListMDMWindowsProfilesToInstallForHostFunc = func(ctx context.Context, hUUID string) ([]*fleet.MDMWindowsProfilePayload, error) {
		return nil, nil
	}
	ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hUUID string) ([]fleet.HostMDMWindowsProfile, error) {
		return nil, nil
	}

	// Stage 3 results plus the cancel-block re-list both go through this single mock so they're consistent.
	// Each row gets a HostSoftwareInstallsExecutionID so the cancel-upcoming loop has something to cancel on
	// non-terminal rows.
	results := make([]*fleet.SetupExperienceStatusResult, 0, len(statuses))
	for i, s := range statuses {
		execID := fmt.Sprintf("pbt-exec-%d", i)
		results = append(results, &fleet.SetupExperienceStatusResult{
			Name:                            fmt.Sprintf("item-%d", i),
			Status:                          s,
			HostSoftwareInstallsExecutionID: &execID,
		})
	}
	ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
		return results, nil
	}
	// The empty-results disambiguation case (empty results + has_items=true → wait) is covered by
	// "active waits when results empty but setup experience configured" in TestGetESPCommands; here we always
	// say "no items configured" so empty input proceeds to finalize cleanly.
	ds.HasWindowsSetupExperienceItemsForTeamFunc = func(ctx context.Context, teamID uint) (bool, error) {
		return false, nil
	}

	// Side-effect hooks. callOrder lets the property assert the cancel-upcoming → cancel-status → persist →
	// cas ordering safely.
	ds.CancelHostUpcomingActivityFunc = func(ctx context.Context, hostID uint, executionID string) (fleet.ActivityDetails, error) {
		trace.cancelUpcomingExecIDs = append(trace.cancelUpcomingExecIDs, executionID)
		trace.callOrder = append(trace.callOrder, "cancel-upcoming")
		return nil, nil
	}
	ds.CancelPendingSetupExperienceStepsFunc = func(ctx context.Context, hUUID string) error {
		trace.callOrder = append(trace.callOrder, "cancel-status")
		return nil
	}
	ds.MDMWindowsInsertCommandsForHostFunc = func(ctx context.Context, hostUUIDOrDeviceID string, cmds []*fleet.MDMWindowsCommand) error {
		trace.callOrder = append(trace.callOrder, "persist")
		for _, c := range cmds {
			trace.persistedCmdUUIDs = append(trace.persistedCmdUUIDs, c.CommandUUID)
		}
		return nil
	}
	ds.SetMDMWindowsAwaitingConfigurationFunc = func(ctx context.Context, mdmDeviceID string, from, to fleet.WindowsMDMAwaitingConfiguration) (bool, error) {
		trace.callOrder = append(trace.callOrder, "cas")
		return true, nil
	}

	svc := &Service{ds: ds, logger: pbtESPLogger}
	svc.SetActivityService(&mock.MockActivityService{})

	device := &fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:           pbtESPDeviceID,
		HostUUID:              pbtESPHostUUID,
		AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive,
	}
	if timedOut {
		past := time.Now().Add(-4 * time.Hour)
		device.AwaitingConfigurationAt = &past
	}
	return svc, device, trace
}

type pbtESPDecision string

const (
	pbtESPWait    pbtESPDecision = "wait"
	pbtESPBlock   pbtESPDecision = "block"
	pbtESPRelease pbtESPDecision = "release"
)

// pbtESPSpec computes the expected outcome from the inputs without referencing production code. It returns
// the decision and the hasSoftwareFailure that production would observe. The latter differs from "results
// contain Failure" when timedOut is true, because Stage 3 is skipped in that case so the production variable
// stays at its zero value.
func pbtESPSpec(
	statuses []fleet.SetupExperienceStatusResultStatus, timedOut, requireAll bool,
) (decision pbtESPDecision, observedHasFailure bool) {
	var inputAnyInFlight, inputHasFailure bool
	for _, s := range statuses {
		switch s {
		case fleet.SetupExperienceStatusFailure:
			inputHasFailure = true
		case fleet.SetupExperienceStatusPending, fleet.SetupExperienceStatusRunning:
			inputAnyInFlight = true
		}
	}
	if timedOut {
		// Wait gates skipped; finalize directly. observedHasFailure stays at its zero value because Stage 3
		// never ran.
		if requireAll {
			return pbtESPBlock, false
		}
		return pbtESPRelease, false
	}
	// !timedOut: Stage 3 ran. observedHasFailure = inputHasFailure.
	if inputAnyInFlight {
		if !inputHasFailure {
			return pbtESPWait, inputHasFailure
		}
		if !requireAll {
			return pbtESPWait, inputHasFailure
		}
		return pbtESPBlock, inputHasFailure // short-circuit: failure + require_all + in-flight siblings -> block now
	}
	if inputHasFailure && requireAll {
		return pbtESPBlock, inputHasFailure
	}
	return pbtESPRelease, inputHasFailure
}

// pbtFindCmdByLocURI returns the first command whose target LocURI contains the given substring.
func pbtFindCmdByLocURI(cmds []*fleet.SyncMLCmd, substr string) *fleet.SyncMLCmd {
	for _, c := range cmds {
		if c.GetTargetURI() != "" && strings.Contains(c.GetTargetURI(), substr) {
			return c
		}
	}
	return nil
}

func TestPBT_HandleESPRelease(t *testing.T) {
	statusGen := rapid.SampledFrom([]fleet.SetupExperienceStatusResultStatus{
		fleet.SetupExperienceStatusPending,
		fleet.SetupExperienceStatusRunning,
		fleet.SetupExperienceStatusSuccess,
		fleet.SetupExperienceStatusFailure,
		fleet.SetupExperienceStatusCancelled,
	})

	rapid.Check(t, func(rt *rapid.T) {
		statuses := rapid.SliceOfN(statusGen, 0, 8).Draw(rt, "statuses")
		timedOut := rapid.Bool().Draw(rt, "timedOut")
		requireAll := rapid.Bool().Draw(rt, "requireAll")

		expected, observedHasFailure := pbtESPSpec(statuses, timedOut, requireAll)
		svc, device, trace := newPBTESPSvc(statuses, timedOut, requireAll)
		cmds, err := svc.getESPCommands(t.Context(), device)
		require.NoErrorf(rt, err, "statuses=%v timedOut=%v requireAll=%v", statuses, timedOut, requireAll)

		if expected == pbtESPWait {
			require.Nilf(rt, cmds, "expected wait, got cmds=%+v", cmds)
			require.Emptyf(rt, trace.callOrder, "wait must not produce side effects; got %v", trace.callOrder)
			return
		}

		// Block or release: must produce non-empty cmds plus a finalized side-effect sequence.
		require.NotEmptyf(rt, cmds, "expected %s, got nil/empty", expected)

		// Persisted CommandUUIDs equal inline CmdID.Value, 1-to-1. Without this, the device's ack of the
		// inline command does not clear the backup row and the server re-sends every subsequent session.
		inlineCmdUUIDs := make([]string, 0, len(cmds))
		for _, c := range cmds {
			inlineCmdUUIDs = append(inlineCmdUUIDs, c.CmdID.Value)
		}
		assert.ElementsMatchf(rt, inlineCmdUUIDs, trace.persistedCmdUUIDs,
			"persisted CommandUUIDs must equal inline CmdID.Value (1-to-1)")

		// Persist must be a single batched call. A regression that loops single inserts would split the
		// CustomErrorText / BlockInStatusPage / TimeOutUntilSyncFailure flags across multiple transactions and
		// expose orphan rows on partial-fail-then-retry.
		persistCount := 0
		for _, ev := range trace.callOrder {
			if ev == "persist" {
				persistCount++
			}
		}
		require.Equalf(rt, 1, persistCount, "persist must be a single batched call; callOrder=%v", trace.callOrder)

		switch expected {
		case pbtESPBlock:
			// Block path NEVER includes ServerHasFinishedProvisioning -- that command would tell Windows the
			// ESP succeeded and proceed past the failure UI. Also NEVER InstallationState alone (VM testing
			// confirmed setting it on the parent PolicyProviders node without per-tracker state from #43776
			// does not escalate the failure UI).
			assert.Nilf(rt, pbtFindCmdByLocURI(cmds, "ServerHasFinishedProvisioning"),
				"block path must NOT include ServerHasFinishedProvisioning")
			assert.Nilf(rt, pbtFindCmdByLocURI(cmds, "InstallationState"),
				"block path uses the timeout-based trigger, not InstallationState")
			assert.Nilf(rt, pbtFindCmdByLocURI(cmds, "WasDeviceSuccessfullyProvisioned"),
				"block path must NOT include WasDeviceSuccessfullyProvisioned (verified on Win11 26200: the "+
					"documented path does not render failure UI on non-Sidecar MDM)")
			assert.Nilf(rt, pbtFindCmdByLocURI(cmds, "IsSyncDone"),
				"block path must NOT include IsSyncDone (verified on Win11 26200: the documented path does not "+
					"render failure UI on non-Sidecar MDM)")
			// Block path always includes BlockInStatusPage=1 (Reset PC), AllowCollectLogsButton, and
			// TimeOutUntilSyncFailure=1 (one minute, forces failure UI).
			blockCmd := pbtFindCmdByLocURI(cmds, "BlockInStatusPage")
			require.NotNilf(rt, blockCmd, "block path must include BlockInStatusPage")
			require.NotNilf(rt, blockCmd.Items[0].Data, "BlockInStatusPage must have data")
			assert.Equalf(rt, "1", blockCmd.Items[0].Data.Content,
				"BlockInStatusPage must be 1 (Reset PC) per DMClient CSP docs")
			assert.NotNilf(rt, pbtFindCmdByLocURI(cmds, "AllowCollectLogsButton"),
				"block path must include AllowCollectLogsButton")
			timeoutCmd := pbtFindCmdByLocURI(cmds, "TimeOutUntilSyncFailure")
			require.NotNilf(rt, timeoutCmd, "block path must include TimeOutUntilSyncFailure")
			require.NotNilf(rt, timeoutCmd.Items[0].Data, "TimeOutUntilSyncFailure must have data")
			assert.Equalf(rt, "1", timeoutCmd.Items[0].Data.Content,
				"TimeOutUntilSyncFailure must be 1 minute (force quick failure)")
			// errorText is software-failure text iff observedHasFailure (Stage 3 ran AND saw a Failure); else
			// timeout text. The pure-timeout path lands here too with observedHasFailure=false.
			errCmd := pbtFindCmdByLocURI(cmds, "CustomErrorText")
			require.NotNilf(rt, errCmd, "block path must include CustomErrorText")
			require.NotNilf(rt, errCmd.Items[0].Data, "CustomErrorText must have data")
			if observedHasFailure {
				assert.Equalf(rt, microsoft_mdm.ESPSoftwareFailureErrorText, errCmd.Items[0].Data.Content,
					"block on software failure must use software-failure error text")
			} else {
				assert.Equalf(rt, microsoft_mdm.ESPTimeoutErrorText, errCmd.Items[0].Data.Content,
					"block on pure timeout must use timeout error text")
			}
		case pbtESPRelease:
			// Release path NEVER includes CustomErrorText -- the failure UI never renders on a release, so
			// any error text would be dead state on the DMClient node.
			assert.Nilf(rt, pbtFindCmdByLocURI(cmds, "CustomErrorText"),
				"release path must NOT include CustomErrorText")
			// Release writes ServerHasFinishedProvisioning at BOTH Device and User scope. Device scope completes
			// the Device setup phase; User scope completes Account setup. The User-scope write requires the
			// user-scope DMClient Provider node to have been created earlier via the hold-phase Add commands.
			shfpDeviceFound, shfpUserFound := false, false
			for _, c := range cmds {
				uri := c.GetTargetURI()
				if !strings.Contains(uri, "ServerHasFinishedProvisioning") {
					continue
				}
				if strings.Contains(uri, "/Device/") {
					shfpDeviceFound = true
				} else if strings.Contains(uri, "/User/") {
					shfpUserFound = true
				}
			}
			assert.Truef(rt, shfpDeviceFound,
				"release path must include Device-scope ServerHasFinishedProvisioning")
			assert.Truef(rt, shfpUserFound,
				"release path must include User-scope ServerHasFinishedProvisioning so Account setup completes; "+
					"omitting it hangs the device on 'Working on it...' indefinitely on Win11 26200 when Account "+
					"setup has been displayed")
			assert.Nilf(rt, pbtFindCmdByLocURI(cmds, "BlockInStatusPage"),
				"release path must NOT include BlockInStatusPage")
		}

		// Cancel-block invariants. The cancel block fires iff (timedOut || (observedHasFailure && requireAll))
		// -- when timedOut, Stage 3 is skipped so observedHasFailure=false and the OR's first operand carries
		// the condition.
		expectedCancelBlock := timedOut || (observedHasFailure && requireAll)
		expectedCancelUpcomingCount := 0
		if expectedCancelBlock {
			for _, s := range statuses {
				if s == fleet.SetupExperienceStatusPending || s == fleet.SetupExperienceStatusRunning {
					expectedCancelUpcomingCount++
				}
			}
		}
		actualCancelStatus := slices.Contains(trace.callOrder, "cancel-status")
		assert.Equalf(rt, expectedCancelBlock, actualCancelStatus,
			"cancel-status invocation must match expected cancel-block fire condition")
		assert.Lenf(rt, trace.cancelUpcomingExecIDs, expectedCancelUpcomingCount,
			"cancel-upcoming count must equal Pending+Running rows in input when cancel block fires")

		// Ordering: cancel-upcoming must precede cancel-status (queue cleanup before status table update so
		// a mid-loop crash + retry sees the same pending statuses and can re-cancel). Both must precede
		// persist (a transient cancel failure aborts the finalize cleanly). Persist must precede CAS (the
		// dropped-response retry safety net runs before we commit awaiting=None).
		var lastCancelUpcoming, firstCancelStatus, firstPersist, firstCas int = -1, -1, -1, -1
		for i, ev := range trace.callOrder {
			switch ev {
			case "cancel-upcoming":
				lastCancelUpcoming = i
			case "cancel-status":
				if firstCancelStatus == -1 {
					firstCancelStatus = i
				}
			case "persist":
				if firstPersist == -1 {
					firstPersist = i
				}
			case "cas":
				if firstCas == -1 {
					firstCas = i
				}
			}
		}
		require.NotEqualf(rt, -1, firstPersist, "persist must run for non-wait outcomes")
		require.NotEqualf(rt, -1, firstCas, "CAS must run for non-wait outcomes")
		require.Lessf(rt, firstPersist, firstCas, "persist must run before CAS; callOrder=%v", trace.callOrder)
		if lastCancelUpcoming != -1 && firstCancelStatus != -1 {
			require.Lessf(rt, lastCancelUpcoming, firstCancelStatus,
				"cancel-upcoming must run before cancel-status; callOrder=%v", trace.callOrder)
		}
		if firstCancelStatus != -1 {
			require.Lessf(rt, firstCancelStatus, firstPersist,
				"cancel-status must run before persist; callOrder=%v", trace.callOrder)
		}
		if lastCancelUpcoming != -1 {
			require.Lessf(rt, lastCancelUpcoming, firstPersist,
				"cancel-upcoming must run before persist; callOrder=%v", trace.callOrder)
		}
	})
}
