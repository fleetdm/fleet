package service

import (
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
