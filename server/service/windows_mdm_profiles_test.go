package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWindowsProfileFleetVariablesLicense(t *testing.T) {
	t.Parallel()
	profileWithVars := `<Replace>
			<Item>
				<Target>
					<LocURI>./Device/Vendor/MSFT/Accounts/DomainName</LocURI>
				</Target>
				<Data>Host UUID: $FLEET_VAR_HOST_UUID</Data>
			</Item>
		</Replace>`

	// Test with free license
	freeLic := &fleet.LicenseInfo{Tier: fleet.TierFree}
	_, err := validateWindowsProfileFleetVariables(profileWithVars, freeLic, nil)
	require.ErrorIs(t, err, fleet.ErrMissingLicense)

	// Test with premium license
	premiumLic := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	vars, err := validateWindowsProfileFleetVariables(profileWithVars, premiumLic, nil)
	require.NoError(t, err)
	require.Contains(t, vars, "HOST_UUID")

	// Test profile without variables (should work with free license)
	profileNoVars := `<Replace>
			<Item>
				<Target>
					<LocURI>./Device/Vendor/MSFT/Accounts/DomainName</LocURI>
				</Target>
				<Data>Static Value</Data>
			</Item>
		</Replace>`
	vars, err = validateWindowsProfileFleetVariables(profileNoVars, freeLic, nil)
	require.NoError(t, err)
	require.Nil(t, vars)
}

func TestValidateWindowsProfileFleetVariables(t *testing.T) {
	tests := []struct {
		name        string
		profileXML  string
		wantErr     bool
		errContains string
	}{
		{
			name: "no variables",
			profileXML: `<Replace>
				<Item>
					<Target>
						<LocURI>./Device/Vendor/MSFT/Policy/Config/System/AllowLocation</LocURI>
					</Target>
					<Data>1</Data>
				</Item>
			</Replace>`,
			wantErr: false,
		},
		{
			name: "HOST_UUID variable",
			profileXML: `<Replace>
				<Item>
					<Target>
						<LocURI>./Device/Vendor/MSFT/Policy/Config/System/AllowLocation</LocURI>
					</Target>
					<Data>$FLEET_VAR_HOST_UUID</Data>
				</Item>
			</Replace>`,
			wantErr: false,
		},
		{
			name: "HOST_UUID variable with braces",
			profileXML: `<Replace>
				<Item>
					<Target>
						<LocURI>./Device/Vendor/MSFT/Policy/Config/System/AllowLocation</LocURI>
					</Target>
					<Data>${FLEET_VAR_HOST_UUID}</Data>
				</Item>
			</Replace>`,
			wantErr: false,
		},
		{
			name: "multiple HOST_UUID variables",
			profileXML: `<Replace>
				<Item>
					<Target>
						<LocURI>./Device/Vendor/MSFT/Policy/Config/System/AllowLocation</LocURI>
					</Target>
					<Data>$FLEET_VAR_HOST_UUID-${FLEET_VAR_HOST_UUID}</Data>
				</Item>
			</Replace>`,
			wantErr: false,
		},
		{
			name: "unsupported variable",
			profileXML: `<Replace>
				<Item>
					<Target>
						<LocURI>./Device/Vendor/MSFT/Policy/Config/System/AllowLocation</LocURI>
					</Target>
					<Data>$FLEET_VAR_HOST_FAKE</Data>
				</Item>
			</Replace>`,
			wantErr:     true,
			errContains: "Fleet variable $FLEET_VAR_HOST_FAKE is not supported in Windows profiles",
		},
		{
			name: "HOST_UUID with another unsupported variable",
			profileXML: `<Replace>
				<Item>
					<Target>
						<LocURI>./Device/Vendor/MSFT/Policy/Config/System/AllowLocation</LocURI>
					</Target>
					<Data>$FLEET_VAR_HOST_UUID-$FLEET_VAR_BOGUS_VAR</Data>
				</Item>
			</Replace>`,
			wantErr:     true,
			errContains: "Fleet variable $FLEET_VAR_BOGUS_VAR is not supported in Windows profiles",
		},
		{
			name: "unknown Fleet variable",
			profileXML: `<Replace>
				<Item>
					<Target>
						<LocURI>./Device/Vendor/MSFT/Policy/Config/System/AllowLocation</LocURI>
					</Target>
					<Data>${FLEET_VAR_UNKNOWN_VAR}</Data>
				</Item>
			</Replace>`,
			wantErr:     true,
			errContains: "Fleet variable $FLEET_VAR_UNKNOWN_VAR is not supported in Windows profiles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pass a premium license for testing (we're not testing license validation here)
			premiumLic := &fleet.LicenseInfo{Tier: fleet.TierPremium}
			_, err := validateWindowsProfileFleetVariables(tt.profileXML, premiumLic, nil)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					require.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAdditionalNDESValidationForWindowsProfiles(t *testing.T) {
	ndesVars := &NDESVarsFound{}
	ndesVars, _ = ndesVars.SetChallenge()
	ndesVars, _ = ndesVars.SetURL()

	// Helper to build a SyncML Add item with a LocURI target and Data content.
	addItem := func(locURI, data string) string {
		return fmt.Sprintf(
			`<Add><Item><Target><LocURI>%s</LocURI></Target><Data>%s</Data></Item></Add>`,
			locURI, data,
		)
	}

	// A valid NDES profile with all required fields.
	validProfile := addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge", "$FLEET_VAR_NDES_SCEP_CHALLENGE") +
		addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL", "$FLEET_VAR_NDES_SCEP_PROXY_URL") +
		addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/SubjectName", "CN=test,OU=$FLEET_VAR_SCEP_RENEWAL_ID")

	tests := []struct {
		name        string
		contents    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid NDES profile",
			contents: validProfile,
		},
		{
			name: "valid NDES profile with braces syntax",
			contents: addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge", "${FLEET_VAR_NDES_SCEP_CHALLENGE}") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL", "${FLEET_VAR_NDES_SCEP_PROXY_URL}") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/SubjectName", "CN=test,OU=${FLEET_VAR_SCEP_RENEWAL_ID}"),
		},
		{
			name: "valid NDES profile wrapped in atomic",
			contents: `<Atomic>` +
				`<Add><CmdID>1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge</LocURI></Target>` +
				`<Data>$FLEET_VAR_NDES_SCEP_CHALLENGE</Data></Item></Add>` +
				`<Add><CmdID>2</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL</LocURI></Target>` +
				`<Data>$FLEET_VAR_NDES_SCEP_PROXY_URL</Data></Item></Add>` +
				`<Add><CmdID>3</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/SubjectName</LocURI></Target>` +
				`<Data>CN=test,OU=$FLEET_VAR_SCEP_RENEWAL_ID</Data></Item></Add>` +
				`</Atomic>`,
		},
		{
			name: "challenge var in wrong field",
			contents: addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL", "$FLEET_VAR_NDES_SCEP_CHALLENGE") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge", "$FLEET_VAR_NDES_SCEP_CHALLENGE"),
			wantErr:     true,
			errContains: `must only be in the SCEP certificate's "Challenge" field`,
		},
		{
			name: "challenge var in arbitrary data field",
			contents: addItem("./Device/Vendor/MSFT/Something/Else", "$FLEET_VAR_NDES_SCEP_CHALLENGE") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge", "$FLEET_VAR_NDES_SCEP_CHALLENGE"),
			wantErr:     true,
			errContains: `must only be in the SCEP certificate's "Challenge" field`,
		},
		{
			name: "proxy url var in wrong field",
			contents: addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge", "$FLEET_VAR_NDES_SCEP_PROXY_URL") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL", "$FLEET_VAR_NDES_SCEP_PROXY_URL"),
			wantErr:     true,
			errContains: `must only be in the SCEP certificate's "ServerURL" field`,
		},
		{
			name: "proxy url var in arbitrary data field",
			contents: addItem("./Device/Vendor/MSFT/Something/Else", "$FLEET_VAR_NDES_SCEP_PROXY_URL") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL", "$FLEET_VAR_NDES_SCEP_PROXY_URL"),
			wantErr:     true,
			errContains: `must only be in the SCEP certificate's "ServerURL" field`,
		},
		{
			name: "challenge var in LocURI target",
			contents: addItem(
				"./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_NDES_SCEP_CHALLENGE/Install/Challenge",
				"$FLEET_VAR_NDES_SCEP_CHALLENGE",
			),
			wantErr:     true,
			errContains: "must not appear in LocURI target paths",
		},
		{
			name: "proxy url var in LocURI target",
			contents: addItem(
				"./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_NDES_SCEP_PROXY_URL/Install/ServerURL",
				"$FLEET_VAR_NDES_SCEP_PROXY_URL",
			),
			wantErr:     true,
			errContains: "must not appear in LocURI target paths",
		},
		{
			name: "challenge field has wrong value",
			contents: addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge", "hardcoded-password") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL", "$FLEET_VAR_NDES_SCEP_PROXY_URL"),
			wantErr:     true,
			errContains: `must be in the SCEP certificate's "Challenge" field`,
		},
		{
			name: "server url field has wrong value",
			contents: addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge", "$FLEET_VAR_NDES_SCEP_CHALLENGE") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL", "https://hardcoded.example.com"),
			wantErr:     true,
			errContains: `must be in the SCEP certificate's "ServerURL" field`,
		},
		{
			name: "subject name missing renewal id",
			contents: addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge", "$FLEET_VAR_NDES_SCEP_CHALLENGE") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL", "$FLEET_VAR_NDES_SCEP_PROXY_URL") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/SubjectName", "CN=test"),
			wantErr:     true,
			errContains: "SubjectName item must contain the $FLEET_VAR_CERTIFICATE_RENEWAL_ID variable in the OU field",
		},
		{
			name: "valid NDES profile with preferred CERTIFICATE_RENEWAL_ID",
			contents: addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge", "$FLEET_VAR_NDES_SCEP_CHALLENGE") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL", "$FLEET_VAR_NDES_SCEP_PROXY_URL") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/SubjectName", "CN=test,OU=$FLEET_VAR_CERTIFICATE_RENEWAL_ID"),
		},
		{
			name: "valid NDES profile with preferred CERTIFICATE_RENEWAL_ID (braces syntax)",
			contents: addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/Challenge", "${FLEET_VAR_NDES_SCEP_CHALLENGE}") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/ServerURL", "${FLEET_VAR_NDES_SCEP_PROXY_URL}") +
				addItem("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1/Install/SubjectName", "CN=test,OU=${FLEET_VAR_CERTIFICATE_RENEWAL_ID}"),
		},
		{
			name:     "nil ndes vars returns nil",
			contents: validProfile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := ndesVars
			if tt.name == "nil ndes vars returns nil" {
				vars = nil
			}
			err := additionalNDESValidationForWindowsProfiles(tt.contents, vars)
			if tt.wantErr {
				require.Error(t, err)
				var badReqErr *fleet.BadRequestError
				require.ErrorAs(t, err, &badReqErr, "expected BadRequestError for: %s", tt.name)
				require.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewMDMWindowsConfigProfileSoftwareUpdate(t *testing.T) {
	// osUpdateSyncML contains the Windows Update install policy LocURI, marking it
	// as a software update profile. otherSyncML is an unrelated policy.
	osUpdateSyncML := syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Update/Install")
	otherSyncML := syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera")

	// configuredSettings returns a WindowsUpdates that reports Configured() == true.
	configuredSettings := func() fleet.WindowsUpdates {
		return fleet.WindowsUpdates{
			DeadlineDays:    optjson.SetInt(7),
			GracePeriodDays: optjson.SetInt(2),
		}
	}

	// setup wires the common mocks required for NewMDMWindowsConfigProfile to reach
	// the software-update handling code.
	setup := func(t *testing.T, premium bool) (fleet.Service, context.Context, *mock.Store) {
		lic := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		if !premium {
			lic = &fleet.LicenseInfo{Tier: fleet.TierFree}
		}
		svc, ctx, ds, _ := setupAppleMDMService(t, lic)
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

		ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
			return nil
		}
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{}, nil
		}
		// The existing-profile check and tracking insert now happen inside this
		// datastore call's transaction; the unit test only verifies that the
		// service passes the correct isSoftwareUpdate flag and surfaces errors.
		ds.NewMDMWindowsConfigProfileFunc = func(ctx context.Context, cp fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
			cp.ProfileUUID = "w-profile-uuid"
			return &cp, nil
		}
		ds.BulkSetPendingMDMHostProfilesFunc = func(ctx context.Context, hids, tids []uint, puuids, uuids []string,
		) (updates fleet.MDMProfilesUpdates, err error) {
			return fleet.MDMProfilesUpdates{}, nil
		}
		// Team lookup used by the enterprise overrides for teamID > 0 cases.
		ds.TeamWithExtrasFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
			return &fleet.Team{ID: tid, Name: "team1"}, nil
		}
		return svc, ctx, ds
	}

	// appConfigWith builds an AppConfig with Windows MDM enabled (so
	// VerifyMDMWindowsConfigured passes) plus the provided mutation applied for OS
	// update settings.
	appConfigWith := func(apply func(*fleet.AppConfig)) func(context.Context) (*fleet.AppConfig, error) {
		return func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.MDM.WindowsEnabledAndConfigured = true
			if apply != nil {
				apply(ac)
			}
			return ac, nil
		}
	}

	t.Run("non software-update profile skips OS update checks", func(t *testing.T) {
		svc, ctx, ds := setup(t, true)
		ds.AppConfigFunc = appConfigWith(nil)

		p, err := svc.NewMDMWindowsConfigProfile(ctx, 0, "other", otherSyncML, nil, fleet.LabelsIncludeAll, nil)
		require.NoError(t, err)
		assert.NotNil(t, p)
		assert.False(t, ds.TeamMDMConfigFuncInvoked)
	})

	t.Run("software-update profile requires premium license", func(t *testing.T) {
		svc, ctx, ds := setup(t, false)
		ds.AppConfigFunc = appConfigWith(nil)

		_, err := svc.NewMDMWindowsConfigProfile(ctx, 0, "other", osUpdateSyncML, nil, fleet.LabelsIncludeAll, nil)
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
		// The gate fails before the profile is inserted.
		assert.False(t, ds.NewMDMWindowsConfigProfileFuncInvoked)
		assert.False(t, ds.TeamMDMConfigFuncInvoked)
	})

	t.Run("no team - app config has no OS updates configured", func(t *testing.T) {
		svc, ctx, ds := setup(t, true)
		ds.AppConfigFunc = appConfigWith(nil)

		p, err := svc.NewMDMWindowsConfigProfile(ctx, 0, "os-update", osUpdateSyncML, nil, fleet.LabelsIncludeAll, nil)
		require.NoError(t, err)
		assert.NotNil(t, p)
		assert.False(t, ds.TeamMDMConfigFuncInvoked)
		assert.True(t, ds.NewMDMWindowsConfigProfileFuncInvoked)
	})

	t.Run("team - team config has no OS updates configured", func(t *testing.T) {
		svc, ctx, ds := setup(t, true)
		ds.AppConfigFunc = appConfigWith(nil)
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			assert.EqualValues(t, 5, teamID)
			return &fleet.TeamMDM{}, nil
		}

		p, err := svc.NewMDMWindowsConfigProfile(ctx, 5, "os-update", osUpdateSyncML, nil, fleet.LabelsIncludeAll, nil)
		require.NoError(t, err)
		assert.NotNil(t, p)
		assert.True(t, ds.TeamMDMConfigFuncInvoked)
		assert.True(t, ds.NewMDMWindowsConfigProfileFuncInvoked)
	})

	t.Run("no team - app config has OS updates configured is rejected", func(t *testing.T) {
		svc, ctx, ds := setup(t, true)
		ds.AppConfigFunc = appConfigWith(func(ac *fleet.AppConfig) {
			ac.MDM.WindowsUpdates = configuredSettings()
		})

		_, err := svc.NewMDMWindowsConfigProfile(ctx, 0, "os-update", osUpdateSyncML, nil, fleet.LabelsIncludeAll, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, fleet.OSUpdatesAlreadyConfiguredErrorMessage)
		// The gate fails before the profile is inserted.
		assert.False(t, ds.NewMDMWindowsConfigProfileFuncInvoked)
	})

	t.Run("team - team config has OS updates configured is rejected", func(t *testing.T) {
		svc, ctx, ds := setup(t, true)
		ds.AppConfigFunc = appConfigWith(nil)
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return &fleet.TeamMDM{WindowsUpdates: configuredSettings()}, nil
		}

		_, err := svc.NewMDMWindowsConfigProfile(ctx, 5, "os-update", osUpdateSyncML, nil, fleet.LabelsIncludeAll, nil)
		require.Error(t, err)
		require.ErrorContains(t, err, fleet.OSUpdatesAlreadyConfiguredErrorMessage)
		assert.False(t, ds.NewMDMWindowsConfigProfileFuncInvoked)
	})
}

func TestNewMDMWindowsConfigProfileLicense(t *testing.T) {
	// For now this only verifies if labels are blocked on free tier
	syncML := syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera")

	setup := func(t *testing.T, premium bool) (fleet.Service, context.Context, *mock.Store) {
		lic := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		if !premium {
			lic = &fleet.LicenseInfo{Tier: fleet.TierFree}
		}
		svc, ctx, ds, _ := setupAppleMDMService(t, lic)
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.MDM.WindowsEnabledAndConfigured = true
			return ac, nil
		}
		ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
			return nil
		}
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{}, nil
		}
		ds.NewMDMWindowsConfigProfileFunc = func(ctx context.Context, cp fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
			cp.ProfileUUID = "w-profile-uuid"
			return &cp, nil
		}
		ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string, filter fleet.TeamFilter) (map[string]uint, error) {
			m := make(map[string]uint)
			for i, label := range labels {
				m[label] = uint(i + 1)
			}
			return m, nil
		}
		ds.LabelsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]*fleet.Label, error) {
			m := make(map[string]*fleet.Label)
			for i, name := range names {
				m[name] = &fleet.Label{ID: uint(i + 1), Name: name}
			}
			return m, nil
		}
		return svc, ctx, ds
	}

	t.Run("labels not allowed on free tier", func(t *testing.T) {
		svc, ctx, ds := setup(t, false)

		_, err := svc.NewMDMWindowsConfigProfile(ctx, 0, "with-labels", syncML, nil, fleet.LabelsIncludeAll, []string{"label1"})
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
		require.ErrorContains(t, err, "Scoping configuration profile")
		assert.False(t, ds.NewMDMWindowsConfigProfileFuncInvoked)

		_, err = svc.NewMDMWindowsConfigProfile(ctx, 0, "with-labels", syncML, []string{"label1"}, fleet.LabelsIncludeAll, nil)
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
		require.ErrorContains(t, err, "Scoping configuration profile")
		assert.False(t, ds.NewMDMWindowsConfigProfileFuncInvoked)
	})

	t.Run("profile without labels allowed on free tier", func(t *testing.T) {
		svc, ctx, ds := setup(t, false)

		p, err := svc.NewMDMWindowsConfigProfile(ctx, 0, "without-labels", syncML, nil, fleet.LabelsIncludeAll, nil)
		require.NoError(t, err)
		assert.NotNil(t, p)
		assert.True(t, ds.NewMDMWindowsConfigProfileFuncInvoked)
	})

	t.Run("labels allowed on premium tier", func(t *testing.T) {
		svc, ctx, ds := setup(t, true)

		p, err := svc.NewMDMWindowsConfigProfile(ctx, 0, "with-labels", syncML, nil, fleet.LabelsIncludeAll, []string{"label1"})
		require.NoError(t, err)
		assert.NotNil(t, p)
		assert.True(t, ds.NewMDMWindowsConfigProfileFuncInvoked)
	})
}

func TestUpdateMDMWindowsConfigProfile(t *testing.T) {
	newExistingProfile := func(name string, teamID uint) *fleet.MDMWindowsConfigProfile {
		return &fleet.MDMWindowsConfigProfile{
			ProfileUUID: "w" + uuid.NewString(),
			Name:        name,
			TeamID:      &teamID,
		}
	}

	setup := func(t *testing.T, lic *fleet.LicenseInfo) (fleet.Service, context.Context, *mock.Store, *TestServerOpts) {
		svc, ctx, ds, opts := setupAppleMDMService(t, lic)
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.MDM.WindowsEnabledAndConfigured = true
			return ac, nil
		}
		ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
			return nil
		}
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{}, nil
		}
		ds.TeamWithExtrasFunc = func(ctx context.Context, teamID uint) (*fleet.Team, error) {
			return &fleet.Team{ID: teamID, Name: fmt.Sprintf("team-%d", teamID)}, nil
		}
		ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string, filter fleet.TeamFilter) (map[string]uint, error) {
			m := make(map[string]uint)
			for i, label := range labels {
				m[label] = uint(i + 1) //nolint:gosec // dismiss G115
			}
			return m, nil
		}
		ds.LabelsByNameFunc = func(ctx context.Context, names []string, filter fleet.TeamFilter) (map[string]*fleet.Label, error) {
			m := make(map[string]*fleet.Label)
			for i, name := range names {
				m[name] = &fleet.Label{ID: uint(i + 1), Name: name} //nolint:gosec // dismiss G115
			}
			return m, nil
		}

		return svc, ctx, ds, opts
	}

	t.Run("labels-only update, happy path", func(t *testing.T) {
		svc, ctx, ds, opts := setup(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		existing := newExistingProfile("Test Profile", 0)

		ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
			require.Equal(t, existing.ProfileUUID, puid)
			return existing, nil
		}
		var updated fleet.MDMWindowsConfigProfile
		ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
			updated = p
			return &p, nil
		}
		var firedActivity activity_api.ActivityDetails
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
			firedActivity = activity
			return nil
		}

		err := svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, nil, []string{"label1"}, fleet.LabelsIncludeAny, nil)
		require.NoError(t, err)

		assert.Empty(t, updated.SyncML)
		assert.Equal(t, existing.Name, updated.Name)
		require.Len(t, updated.LabelsIncludeAny, 1)
		assert.Equal(t, "label1", updated.LabelsIncludeAny[0].LabelName)

		require.NotNil(t, firedActivity)
		act, ok := firedActivity.(*fleet.ActivityTypeEditedConfigurationProfile)
		require.True(t, ok)
		assert.Equal(t, existing.Name, act.ProfileName)
		assert.Equal(t, "windows", act.Platform)
	})

	// No "different name is rejected" test here: there's no request field a
	// client could use to submit a new name, so nothing to reject at this
	// layer -- that check only has meaning in the datastore integration test.
	t.Run("profile content update, matching name", func(t *testing.T) {
		svc, ctx, ds, opts := setup(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		existing := newExistingProfile("Test Profile", 0)
		syncML := syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Bluetooth/AllowDiscoverableMode")

		ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
			return existing, nil
		}
		var updated fleet.MDMWindowsConfigProfile
		ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
			updated = p
			return &p, nil
		}
		var firedActivity activity_api.ActivityDetails
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
			firedActivity = activity
			return nil
		}

		err := svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, syncML, nil, fleet.LabelsIncludeAll, nil)
		require.NoError(t, err)
		assert.Equal(t, syncML, []byte(updated.SyncML))
		assert.Equal(t, existing.Name, updated.Name)

		require.NotNil(t, firedActivity)
		act, ok := firedActivity.(*fleet.ActivityTypeEditedConfigurationProfile)
		require.True(t, ok)
		assert.Equal(t, existing.Name, act.ProfileName)
		assert.Equal(t, "windows", act.Platform)
	})

	t.Run("profile content update for a team-scoped profile", func(t *testing.T) {
		svc, ctx, ds, opts := setup(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		existing := newExistingProfile("Test Profile", 5)
		syncML := syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Bluetooth/AllowDiscoverableMode")

		ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
			return existing, nil
		}
		var updated fleet.MDMWindowsConfigProfile
		ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
			updated = p
			return &p, nil
		}
		var firedActivity activity_api.ActivityDetails
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
			firedActivity = activity
			return nil
		}

		err := svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, syncML, nil, fleet.LabelsIncludeAll, nil)
		require.NoError(t, err)
		assert.Equal(t, syncML, []byte(updated.SyncML))
		require.NotNil(t, updated.TeamID)
		assert.EqualValues(t, 5, *updated.TeamID)

		require.NotNil(t, firedActivity)
		act, ok := firedActivity.(*fleet.ActivityTypeEditedConfigurationProfile)
		require.True(t, ok)
		require.NotNil(t, act.TeamID)
		assert.EqualValues(t, 5, *act.TeamID)
		require.NotNil(t, act.TeamName)
		assert.Equal(t, "team-5", *act.TeamName)
	})

	t.Run("profile content and labels update atomically in one call", func(t *testing.T) {
		svc, ctx, ds, _ := setup(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		existing := newExistingProfile("Test Profile", 0)
		syncML := syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Bluetooth/AllowDiscoverableMode")

		ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
			return existing, nil
		}
		var updated fleet.MDMWindowsConfigProfile
		ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
			updated = p
			return &p, nil
		}

		err := svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, syncML, []string{"label1"}, fleet.LabelsIncludeAny, []string{"label2"})
		require.NoError(t, err)
		assert.Equal(t, syncML, []byte(updated.SyncML))
		require.Len(t, updated.LabelsIncludeAny, 1)
		assert.Equal(t, "label1", updated.LabelsIncludeAny[0].LabelName)
		require.Len(t, updated.LabelsExcludeAny, 1)
		assert.Equal(t, "label2", updated.LabelsExcludeAny[0].LabelName)
	})

	t.Run("editing a Fleet-managed profile is rejected", func(t *testing.T) {
		svc, ctx, ds, _ := setup(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		existing := newExistingProfile(mdm.FleetWindowsOSUpdatesProfileName, 0)

		ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
			return existing, nil
		}
		ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
			t.Fatal("should not reach the datastore update")
			return nil, nil
		}

		err := svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, nil, []string{"label1"}, fleet.LabelsIncludeAny, nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "managed by Fleet")
	})

	t.Run("nonexistent profile propagates the not-found error", func(t *testing.T) {
		svc, ctx, ds, _ := setup(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		wantErr := errors.New("simulated profile lookup error")
		ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
			return nil, wantErr
		}

		err := svc.UpdateMDMConfigProfile(ctx, "w"+uuid.NewString(), nil, nil, fleet.LabelsIncludeAll, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, wantErr)
	})

	t.Run("labels require a premium license, content-only edits do not", func(t *testing.T) {
		svc, ctx, ds, _ := setup(t, &fleet.LicenseInfo{Tier: fleet.TierFree})
		existing := newExistingProfile("Test Profile", 0)

		ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
			return existing, nil
		}
		ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
			t.Fatal("should not reach the datastore update")
			return nil, nil
		}

		err := svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, nil, []string{"label1"}, fleet.LabelsIncludeAny, nil)
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
		require.ErrorContains(t, err, "Scoping configuration profiles with labels requires Fleet Premium license")

		// content-only edit (no labels) still succeeds on a free license
		ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
			return &p, nil
		}
		syncML := syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Bluetooth/AllowDiscoverableMode")
		err = svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, syncML, nil, fleet.LabelsIncludeAll, nil)
		require.NoError(t, err)
	})

	t.Run("Fleet variables used in the upload are threaded through to the datastore call", func(t *testing.T) {
		svc, ctx, ds, _ := setup(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		existing := newExistingProfile("Test Profile", 0)
		syncML := syncMLForTest("./Device/Vendor/MSFT/Accounts/DomainName")
		syncML = append(syncML, []byte("$FLEET_VAR_HOST_UUID")...)

		ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
			return existing, nil
		}
		var capturedVars []fleet.FleetVarName
		ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
			capturedVars = usesFleetVars
			return &p, nil
		}

		err := svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, syncML, nil, fleet.LabelsIncludeAll, nil)
		require.NoError(t, err)
		assert.Contains(t, capturedVars, fleet.FleetVarName("HOST_UUID"))
	})

	t.Run("OS-update profile restrictions apply on update the same as on create", func(t *testing.T) {
		osUpdateSyncML := syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Update/Install")

		t.Run("update succeeds when OS updates are not already configured", func(t *testing.T) {
			svc, ctx, ds, _ := setup(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
			existing := newExistingProfile("Test Profile", 0)

			ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
				return existing, nil
			}
			var updated fleet.MDMWindowsConfigProfile
			ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
				updated = p
				return &p, nil
			}

			err := svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, osUpdateSyncML, nil, fleet.LabelsIncludeAll, nil)
			require.NoError(t, err)
			assert.Equal(t, osUpdateSyncML, []byte(updated.SyncML))
		})

		t.Run("update is rejected when OS updates are already configured", func(t *testing.T) {
			svc, ctx, ds, _ := setup(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
			existing := newExistingProfile("Test Profile", 0)

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				ac := &fleet.AppConfig{}
				ac.MDM.WindowsEnabledAndConfigured = true
				ac.MDM.WindowsUpdates = fleet.WindowsUpdates{
					DeadlineDays:    optjson.SetInt(7),
					GracePeriodDays: optjson.SetInt(2),
				}
				return ac, nil
			}
			ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
				return existing, nil
			}
			ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
				t.Fatal("should not reach the datastore update")
				return nil, nil
			}

			err := svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, osUpdateSyncML, nil, fleet.LabelsIncludeAll, nil)
			require.Error(t, err)
			assert.ErrorContains(t, err, fleet.OSUpdatesAlreadyConfiguredErrorMessage)
		})

		t.Run("update requires a premium license", func(t *testing.T) {
			svc, ctx, ds, _ := setup(t, &fleet.LicenseInfo{Tier: fleet.TierFree})
			existing := newExistingProfile("Test Profile", 0)

			ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
				return existing, nil
			}
			ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
				t.Fatal("should not reach the datastore update")
				return nil, nil
			}

			err := svc.UpdateMDMConfigProfile(ctx, existing.ProfileUUID, osUpdateSyncML, nil, fleet.LabelsIncludeAll, nil)
			require.ErrorIs(t, err, fleet.ErrMissingLicense)
		})
	})

	t.Run("authorization outcome matches user role and team membership", func(t *testing.T) {
		testCases := []struct {
			name             string
			user             *fleet.User
			shouldFailGlobal bool
			shouldFailTeam   bool
		}{
			{"global admin", &fleet.User{GlobalRole: new(fleet.RoleAdmin)}, false, false},
			{"global maintainer", &fleet.User{GlobalRole: new(fleet.RoleMaintainer)}, false, false},
			{"global observer", &fleet.User{GlobalRole: new(fleet.RoleObserver)}, true, true},
			{"team admin, belongs to team", &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}}, true, false},
			{"team admin, DOES NOT belong to team", &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleAdmin}}}, true, true},
			{"team maintainer, belongs to team", &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}}, true, false},
			{"team maintainer, DOES NOT belong to team", &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleMaintainer}}}, true, true},
			{"team observer, belongs to team", &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}}, true, true},
			{"team observer, DOES NOT belong to team", &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}}, true, true},
			{"user no roles", &fleet.User{ID: 1337}, true, true},
		}

		checkShouldFail := func(t *testing.T, err error, shouldFail bool) {
			t.Helper()
			if !shouldFail {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
			}
		}

		for _, tt := range testCases {
			t.Run(tt.name, func(t *testing.T) {
				svc, baseCtx, ds, _ := setup(t, &fleet.LicenseInfo{Tier: fleet.TierPremium})
				ctx := viewer.NewContext(baseCtx, viewer.Viewer{User: tt.user})

				noTeamProfile := newExistingProfile("No Team Profile", 0)
				teamProfile := newExistingProfile("Team Profile", 1)

				ds.GetMDMWindowsConfigProfileFunc = func(ctx context.Context, puid string) (*fleet.MDMWindowsConfigProfile, error) {
					if puid == noTeamProfile.ProfileUUID {
						return noTeamProfile, nil
					}
					return teamProfile, nil
				}
				ds.UpdateMDMWindowsConfigProfileFunc = func(ctx context.Context, p fleet.MDMWindowsConfigProfile, usesFleetVars []fleet.FleetVarName) (*fleet.MDMWindowsConfigProfile, error) {
					return &p, nil
				}

				// profile content and labels are deliberately nil/empty here --
				// this isolates the authz checks from content/label validation.
				err := svc.UpdateMDMConfigProfile(ctx, noTeamProfile.ProfileUUID, nil, nil, fleet.LabelsIncludeAll, nil)
				checkShouldFail(t, err, tt.shouldFailGlobal)

				err = svc.UpdateMDMConfigProfile(ctx, teamProfile.ProfileUUID, nil, nil, fleet.LabelsIncludeAll, nil)
				checkShouldFail(t, err, tt.shouldFailTeam)
			})
		}
	})
}
