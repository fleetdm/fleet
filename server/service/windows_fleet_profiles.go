package service

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
)

// Windows MDM has no CSP that writes an arbitrary registry value. The documented route is to ingest an ADMX that declares a
// policy pointing at the key you want, then set that policy. The value this writes, HKLM\SOFTWARE\FleetDM\Orbit\EnrollSecret, is
// what orbit reads on every start and clears once the secret is in its keystore. That is what makes this profile the recovery
// path: an administrator resending it delivers a fresh secret to a host whose previous one was spent, without reinstalling the
// MSI (whose fixed product GUID would refuse to run again anyway).
//
//nolint:gosec // G101 false positive, a policy definition, not a credential
const windowsEnrollSecretADMX = `<policyDefinitions revision="1.0" schemaVersion="1.0">
  <policyNamespaces>
    <target namespace="FleetDM.Policies.Orbit" prefix="fleetd"/>
    <using namespace="Microsoft.Policies.Windows" prefix="windows"/>
  </policyNamespaces>
  <resources minRequiredRevision="1.0"/>
  <categories>
    <category name="fleetd" displayName="Fleet"/>
  </categories>
  <policies>
    <policy name="EnrollSecret" class="Machine" displayName="Fleetd enroll secret"
            explainText="One-time enroll secret used by fleetd to enroll this host in Fleet."
            key="SOFTWARE\FleetDM\Orbit">
      <parentCategory ref="fleetd"/>
      <supportedOn ref="windows:SUPPORTED_Windows7"/>
      <elements>
        <text id="EnrollSecret_Value" valueName="EnrollSecret" required="true"/>
      </elements>
    </policy>
  </policies>
</policyDefinitions>`

const (
	// windowsEnrollSecretADMXInstallURI ingests the ADMX above. The path is ADMXInstall/{AppName}/{SettingType}/{FileUID}.
	//nolint:gosec // G101 false positive, an OMA-DM URI, not a credential
	windowsEnrollSecretADMXInstallURI = "./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetdEnrollSecret/Policy/FleetdEnrollSecretAdmx"
	// windowsEnrollSecretPolicyURI sets the ingested policy. The area name is {AppName}~{SettingType}~{CategoryFromTheADMX}.
	//nolint:gosec // G101 false positive, an OMA-DM URI, not a credential
	windowsEnrollSecretPolicyURI = "./Device/Vendor/MSFT/Policy/Config/FleetdEnrollSecret~Policy~fleetd/EnrollSecret"
)

// windowsEnrollSecretProfileSyncML builds the Fleet-managed profile that carries a one-time enroll secret to the registry.
func windowsEnrollSecretProfileSyncML() ([]byte, error) {
	var admx bytes.Buffer
	if err := xml.EscapeText(&admx, []byte(windowsEnrollSecretADMX)); err != nil {
		return nil, err
	}

	// The policy payload is itself markup, so it is escaped into the <Data> element rather than sent raw. The placeholder
	// survives escaping unchanged, and the token it expands to is URL-safe base64, so nothing in it needs escaping either.
	policyValue := `<enabled/><data id="EnrollSecret_Value" value="` +
		fleet.HostSecretPlaceholder(fleet.HostSecretEnrollSecret) + `"/>`
	var value bytes.Buffer
	if err := xml.EscapeText(&value, []byte(policyValue)); err != nil {
		return nil, err
	}

	return []byte(`<Add>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
      <Type xmlns="syncml:metinf">text/plain</Type>
    </Meta>
    <Target>
      <LocURI>` + windowsEnrollSecretADMXInstallURI + `</LocURI>
    </Target>
    <Data>` + admx.String() + `</Data>
  </Item>
</Add>
<Replace>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
      <Type xmlns="syncml:metinf">text/plain</Type>
    </Meta>
    <Target>
      <LocURI>` + windowsEnrollSecretPolicyURI + `</LocURI>
    </Target>
    <Data>` + value.String() + `</Data>
  </Item>
</Replace>`), nil
}

// pushEnrollSecretToOrphanedEnrollment gives a Windows MDM enrollment whose host was deleted a way back. Deleting a host deletes
// its one-time enroll secrets, so the agent is left holding one the server no longer knows, and with no host there is no profile
// to resend. The enrollment survives the delete, and its MDM session is a channel only that device holds, so Fleet mints a secret
// for the enrollment and writes it to the registry value the enroll secret profile carries. orbit reads that value before each
// enroll attempt, and its enrollment recreates the host, or claims the pending Autopilot host by serial. Failures are logged
// rather than returned, so the session is unaffected and the next one tries again.
//
// For an enrollment Fleet installs fleetd on (a user-driven or Autopilot enrollment), the fleetd install is queued with the same
// secret in case fleetd was removed before the host was deleted. With fleetd still installed, Windows runs it as a reconfigure
// that fails and changes nothing.
func (svc *Service) pushEnrollSecretToOrphanedEnrollment(ctx context.Context, enrolledDevice *fleet.MDMWindowsEnrolledDevice) {
	logger := svc.logger.With("enrollment_id", enrolledDevice.ID, "host_uuid", enrolledDevice.HostUUID)

	// A device whose fleetd is gone for good keeps checking in, so the push must not repeat for every session: once per secret.
	pushed, err := svc.ds.WindowsMDMEnrollSecretPushed(ctx, enrolledDevice.ID, windowsEnrollSecretPolicyURI)
	if err != nil {
		logger.ErrorContext(ctx, "failed to check for a queued one-time enroll secret push", "err", err)
		ctxerr.Handle(ctx, err)
		return
	}
	if pushed {
		return
	}

	syncML, err := windowsEnrollSecretProfileSyncML()
	if err != nil {
		logger.ErrorContext(ctx, "failed to build the enroll secret push", "err", err)
		ctxerr.Handle(ctx, err)
		return
	}
	pushCmd, err := buildCommandFromProfileBytes(fmt.Appendf(nil, "<Atomic>%s</Atomic>", syncML), uuid.NewString())
	if err != nil {
		logger.ErrorContext(ctx, "failed to build the enroll secret push", "err", err)
		ctxerr.Handle(ctx, err)
		return
	}
	pushCmd.TargetLocURI = windowsEnrollSecretPolicyURI

	var installCmd *fleet.MDMWindowsCommand
	if microsoft_mdm.IsValidUPN(enrolledDevice.MDMEnrollUserID) {
		if installCmd, err = svc.buildFleetdInstallCommand(ctx, enrolledDevice); err != nil {
			logger.ErrorContext(ctx, "failed to build the fleetd install for a one-time enroll secret push", "err", err)
			ctxerr.Handle(ctx, err)
			return
		}
		// Queuing the push alone would mark the secret pushed, and no later session would send the install.
		if installCmd == nil {
			return
		}
	}

	queued, err := svc.ds.QueueWindowsMDMEnrollSecretPush(ctx, enrolledDevice.ID, enrolledDevice.MDMDeviceID, pushCmd, installCmd)
	if err != nil {
		logger.ErrorContext(ctx, "failed to queue a one-time enroll secret push", "err", err)
		ctxerr.Handle(ctx, err)
		return
	}
	if queued {
		logger.InfoContext(ctx, "queued a one-time enroll secret for a windows mdm enrollment whose host was deleted",
			"command_uuid", pushCmd.CommandUUID, "with_fleetd_install", installCmd != nil)
	}
}

// ensureFleetWindowsProfiles keeps the Fleet-managed Windows profiles in step with the server configuration, the way
// ensureFleetProfiles does for Apple. Today that is only the enroll secret profile.
//
// One profile per team, plus "no team", because a Windows configuration profile is scoped by team and every Windows MDM host
// needs the enroll secret profile. The contents are identical everywhere: the secret is resolved per enrollment at delivery,
// not per team.
func ensureFleetWindowsProfiles(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, useOneTimeEnrollSecrets bool) error {
	// This runs on every reconcile, so one read decides what, if anything, needs writing.
	existing, err := ds.ListMDMWindowsConfigProfilesByName(ctx, mdm.FleetWindowsEnrollSecretProfileName)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing windows enroll secret profiles")
	}

	if !useOneTimeEnrollSecrets {
		// Turning the feature off has to take the profile with it.
		for _, profile := range existing {
			if err := ds.DeleteMDMWindowsConfigProfile(ctx, profile.ProfileUUID); err != nil && !fleet.IsNotFound(err) {
				return ctxerr.Wrap(ctx, err, "deleting windows enroll secret profile")
			}
		}
		return nil
	}

	// Only missing profiles are written, never an existing one. Rewriting would change its checksum and redeliver it to every host
	// in the team, a fleet-wide wave of commands on an upgrade for a profile that almost always carries an empty value. A future
	// content change that has to reach existing hosts should be a deliberate step.
	present := make(map[uint]struct{}, len(existing))
	for _, profile := range existing {
		present[ptr.ValOrZero(profile.TeamID)] = struct{}{}
	}

	syncML, err := windowsEnrollSecretProfileSyncML()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building windows enroll secret profile")
	}

	teams, err := ds.TeamsSummary(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing teams for windows enroll secret profiles")
	}
	teamIDs := []*uint{nil}
	for _, team := range teams {
		teamIDs = append(teamIDs, &team.ID)
	}
	written := 0
	for _, teamID := range teamIDs {
		if _, ok := present[ptr.ValOrZero(teamID)]; ok {
			continue
		}
		if err := ds.SetOrUpdateMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
			TeamID: teamID,
			Name:   mdm.FleetWindowsEnrollSecretProfileName,
			SyncML: syncML,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "upserting windows enroll secret profile")
		}
		written++
	}
	if written > 0 {
		logger.DebugContext(ctx, "wrote windows enroll secret profiles", "teams", written)
	}
	return nil
}
