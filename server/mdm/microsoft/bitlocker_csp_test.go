package microsoft_mdm

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestSystemDriveRequiresStartupAuthSpec_validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    SystemDriveRequiresStartupAuthSpec
		wantErr string
	}{
		{
			name:    "empty cmdUUID",
			spec:    SystemDriveRequiresStartupAuthSpec{},
			wantErr: "cmdUUID is required",
		},
		{
			name: "fields set but not enabled",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID:                "test-uuid",
				Enabled:                false,
				ConfigureTPMStartupKey: ptr.Uint(PolicyOptDropdownRequired),
			},
			wantErr: "enabled must be true if any other field is set",
		},
		{
			name: "PIN policies set but not enabled",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID:              "test-uuid",
				ConfigurePINPolicies: true,
			},
			wantErr: "enabled must be true if any other field is set",
		},
		{
			name: "valid configuration with no fields",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID: "test-uuid",
				Enabled: false,
			},
		},
		{
			name: "valid configuration with enabled fields",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID:                "test-uuid",
				Enabled:                true,
				ConfigureTPMStartupKey: ptr.Uint(PolicyOptDropdownRequired),
				ConfigurePIN:           ptr.Uint(PolicyOptDropdownOptional),
			},
		},
		{
			name: "invalid TPMStartupKey value",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID:                "test-uuid",
				Enabled:                true,
				ConfigureTPMStartupKey: ptr.Uint(99),
			},
			wantErr: "ConfigureTPMStartupKey must be one of the PolicyOptDropdown* variants",
		},
		{
			name: "invalid PIN value",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID:      "test-uuid",
				Enabled:      true,
				ConfigurePIN: ptr.Uint(99),
			},
			wantErr: "ConfigurePIN must be one of the PolicyOptDropdown* variants",
		},
		{
			name: "invalid TPMPINKey value",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID:            "test-uuid",
				Enabled:            true,
				ConfigureTPMPINKey: ptr.Uint(99), // Invalid value
			},
			wantErr: "ConfigureTPMPINKey must be one of the PolicyOptDropdown* variants",
		},
		{
			name: "invalid TPM value",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID:      "test-uuid",
				Enabled:      true,
				ConfigureTPM: ptr.Uint(99),
			},
			wantErr: "ConfigureTPM must be one of the PolicyOptDropdown* variants",
		},
		{
			name: "all fields set with valid values",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID:                   "test-uuid",
				Enabled:                   true,
				ConfigureNonTPMStartupKey: ptr.Bool(true),
				ConfigureTPMStartupKey:    ptr.Uint(PolicyOptDropdownDisallowed),
				ConfigurePIN:              ptr.Uint(PolicyOptDropdownRequired),
				ConfigureTPMPINKey:        ptr.Uint(PolicyOptDropdownOptional),
				ConfigureTPM:              ptr.Uint(PolicyOptDropdownRequired),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Errorf(t, err, tt.wantErr)
			}
		})
	}
}

func TestSystemDriveRequiresStartupAuthCmd_Template(t *testing.T) {
	tests := []struct {
		name     string
		spec     SystemDriveRequiresStartupAuthSpec
		expected string
	}{
		{
			name: "disabled",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID: "uuid-123",
				Enabled: false,
			},
			expected: `
				<Atomic>
					<CmdID>uuid-123</CmdID>
					<Replace>
						<CmdID>uuid-123-1</CmdID>
						<Item>
							<Meta>
							  <Format>chr</Format>
							  <Type>text/plain</Type>
							</Meta>
							<Target>
								<LocURI>./Device/Vendor/MSFT/BitLocker/SystemDrivesRequireStartupAuthentication</LocURI>
							</Target>
							<Data>
							<![CDATA[<disabled/>]]>
							</Data>
						</Item>
					</Replace>
				</Atomic>`,
		},
		{
			name: "enabled",
			spec: SystemDriveRequiresStartupAuthSpec{
				CmdUUID:                   "uuid-789",
				Enabled:                   true,
				ConfigureNonTPMStartupKey: ptr.Bool(true),
				ConfigureTPMStartupKey:    ptr.Uint(PolicyOptDropdownRequired),
				ConfigurePIN:              ptr.Uint(PolicyOptDropdownOptional),
				ConfigureTPMPINKey:        ptr.Uint(PolicyOptDropdownDisallowed),
				ConfigureTPM:              ptr.Uint(PolicyOptDropdownRequired),
			},
			expected: `
				<Atomic>
					<CmdID>uuid-789</CmdID>
					<Replace>
						<CmdID>uuid-789-1</CmdID>
						<Item>
							<Meta>
							  <Format>chr</Format>
							  <Type>text/plain</Type>
							</Meta>
							<Target>
								<LocURI>./Device/Vendor/MSFT/BitLocker/SystemDrivesRequireStartupAuthentication</LocURI>
							</Target>
							<Data>
							<![CDATA[
								<enabled/>
								<data id="ConfigureNonTPMStartupKeyUsage_Name" value="true"/>
								<data id="ConfigureTPMStartupKeyUsageDropDown_Name" value="1"/>
								<data id="ConfigurePINUsageDropDown_Name" value="2"/>
								<data id="ConfigureTPMPINKeyUsageDropDown_Name" value="0"/>
								<data id="ConfigureTPMUsageDropDown_Name" value="1"/>
							]]>
							</Data>
						</Item>
					</Replace>
				</Atomic>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := SystemDriveRequiresStartupAuthCmd(tt.spec)
			require.NoError(t, err)

			got := strings.Join(strings.Fields(string(cmd.RawCommand)), " ")
			want := strings.Join(strings.Fields(tt.expected), " ")

			require.Equal(t, want, got)
			require.Equal(t, tt.spec.CmdUUID, cmd.CommandUUID)
			require.Equal(t, "./Device/Vendor/MSFT/BitLocker/SystemDrivesRequireStartupAuthentication", cmd.TargetLocURI)
		})
	}
}

func TestSystemDriveRequiresStartupAuthCmd_PINPolicies(t *testing.T) {
	rawCommand := func(configurePINPolicies bool) string {
		cmd, err := SystemDriveRequiresStartupAuthCmd(SystemDriveRequiresStartupAuthSpec{
			CmdUUID: "uuid-456", Enabled: true, ConfigurePINPolicies: configurePINPolicies,
		})
		require.NoError(t, err)
		return string(cmd.RawCommand)
	}

	raw := rawCommand(true)
	for node, payload := range map[string]string{
		"SystemDrivesMinimumPINLength": fmt.Sprintf(`<enabled/><data id="MinPINLength" value="%d"/>`, BitLockerPINMinLength),
		"SystemDrivesEnhancedPIN":      "<enabled/>",
		// Phrased as a prohibition, so disabling it is what lets standard users change their PIN.
		"SystemDrivesDisallowStandardUsersCanChangePIN": "<disabled/>",
	} {
		pattern := regexp.QuoteMeta("<LocURI>./Device/Vendor/MSFT/BitLocker/"+node+"</LocURI>") +
			`\s*</Target>\s*<Data>` + regexp.QuoteMeta("<![CDATA["+payload+"]]>")
		require.Regexp(t, pattern, raw, node)
	}

	// One Atomic with the startup policy and the three PIN policies, each with its own CmdID.
	cmdIDs := map[string]struct{}{}
	for _, match := range regexp.MustCompile(`<CmdID>([^<]+)</CmdID>`).FindAllStringSubmatch(raw, -1) {
		cmdIDs[match[1]] = struct{}{}
	}
	require.Len(t, cmdIDs, 5)
	require.Equal(t, 1, strings.Count(raw, "<Atomic>"))

	require.NotContains(t, rawCommand(false), "SystemDrivesMinimumPINLength")
}
