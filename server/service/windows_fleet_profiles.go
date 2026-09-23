package service

import (
	"bytes"
	"context"
	"encoding/xml"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
)

// Windows MDM has no CSP that writes an arbitrary registry value. The documented route is to ingest an ADMX that declares a
// policy pointing at the key you want, then set that policy. The restriction on ingested policies is a denylist of Microsoft's
// own trees (System, Software\Microsoft, Software\Policies\Microsoft), so SOFTWARE\FleetDM\Orbit is a legal target, and a
// <text> element produces a REG_SZ. Requires Windows 10 1709 or later, device scope, and not Home.
//
// The value this writes, HKLM\SOFTWARE\FleetDM\Orbit\EnrollSecret, is what orbit reads on every start and clears once the
// secret is in its keystore. That is what makes this profile the recovery path: an administrator resending it delivers a fresh
// secret to a host whose previous one was spent, without reinstalling the MSI (whose fixed product GUID would refuse to run
// again anyway).
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
    <policy name="EnrollSecret" class="Machine" displayName="Fleet enroll secret"
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
	windowsEnrollSecretADMXInstallURI = "./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetdEnrollSecret/Policy/FleetdEnrollSecretAdmx"
	// windowsEnrollSecretPolicyURI sets the ingested policy. The area name is {AppName}~{SettingType}~{CategoryFromTheADMX}.
	windowsEnrollSecretPolicyURI = "./Device/Vendor/MSFT/Policy/Config/FleetdEnrollSecret~Policy~fleetd/EnrollSecret"
)

// windowsEnrollSecretProfileSyncML builds the Fleet-managed profile that carries a one-time enroll secret to the registry.
//
// The Add and the Replace are separate top-level elements rather than one Atomic, because the ADMX has to be ingested before
// the policy it declares can be set, and Fleet delivers a profile's elements in document order within a single SyncML body. A
// re-sent Add on a device that already ingested it answers 418, which Fleet already converts to a Replace in the same session.
//
// The secret itself is never in here: the profile stores the $FLEET_HOST_SECRET_ENROLL_SECRET placeholder and the real value is
// minted per enrollment when the command is delivered.
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

// ensureFleetWindowsProfiles keeps the Fleet-managed Windows profiles in step with the server configuration, the way
// ensureFleetProfiles does for Apple. Today that is only the enroll secret profile.
//
// One profile per team, plus "no team", because a Windows configuration profile is scoped by team and every Windows MDM host
// needs the carrier. The contents are identical everywhere: the secret is minted per enrollment at delivery, not per team.
func ensureFleetWindowsProfiles(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, useOneTimeEnrollSecrets bool) error {
	teamIDs, err := windowsProfileTeamTargets(ctx, ds)
	if err != nil {
		return err
	}

	if !useOneTimeEnrollSecrets {
		// Turning the feature off has to take the carrier with it, or hosts keep a profile whose placeholder nothing expands.
		for _, teamID := range teamIDs {
			if err := ds.DeleteMDMWindowsConfigProfileByTeamAndName(ctx, teamID, mdm.FleetWindowsEnrollSecretProfileName); err != nil &&
				!fleet.IsNotFound(err) {
				return ctxerr.Wrap(ctx, err, "deleting windows enroll secret profile")
			}
		}
		return nil
	}

	syncML, err := windowsEnrollSecretProfileSyncML()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building windows enroll secret profile")
	}
	for _, teamID := range teamIDs {
		if err := ds.SetOrUpdateMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
			TeamID: teamID,
			Name:   mdm.FleetWindowsEnrollSecretProfileName,
			SyncML: syncML,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "upserting windows enroll secret profile")
		}
	}
	logger.DebugContext(ctx, "ensured windows enroll secret profiles", "teams", len(teamIDs))
	return nil
}

// windowsProfileTeamTargets returns every team a Fleet-managed Windows profile belongs to, with nil for "no team".
//
// AggregateEnrollSecretPerTeam is the enumeration Apple's equivalent uses. It returns a row per team whether or not the team has
// a shared secret, but omits "no team" when that has none, so "no team" is added when missing.
func windowsProfileTeamTargets(ctx context.Context, ds fleet.Datastore) ([]*uint, error) {
	secrets, err := ds.AggregateEnrollSecretPerTeam(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting enroll secrets aggregates")
	}
	targets := make([]*uint, 0, len(secrets)+1)
	hasNoTeam := false
	for _, es := range secrets {
		if es.TeamID == nil {
			hasNoTeam = true
		}
		targets = append(targets, es.TeamID)
	}
	if !hasNoTeam {
		targets = append(targets, nil)
	}
	return targets, nil
}
