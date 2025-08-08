package microsoft_mdm

import (
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
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

func TestSystemDrRequiresStartupAuthCmd_Template(t *testing.T) {
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
					<CmdID>uuid-123-1</CmdID>
					<Replace>
						<CmdID>uuid-123-2</CmdID>
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
					<CmdID>uuid-789-1</CmdID>
					<Replace>
						<CmdID>uuid-789-2</CmdID>
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
