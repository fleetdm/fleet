package microsoft_mdm

import (
	"bytes"
	"errors"
	"text/template"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	PolicyOptDropdownDisallowed = iota
	PolicyOptDropdownRequired
	PolicyOptDropdownOptional
)

var systemDriveRequiresStartupAuthTmpl = template.Must(template.New("cmd").Funcs(map[string]any{
	"derefBool": func(val *bool) bool {
		return val != nil && *val
	}, "derefUint": func(val *uint) uint {
		if val == nil {
			// Use 'optional' as the default value.
			return PolicyOptDropdownOptional
		}
		return *val
	}, "minPINLength": func() int {
		return BitLockerPINMinLength
	}}).Parse(`
<Atomic>
	<CmdID>{{ .CmdUUID }}</CmdID>
	<Replace>
		<CmdID>{{ .CmdUUID }}-1</CmdID>
		<Item>
			<Meta>
			  <Format>chr</Format>
			  <Type>text/plain</Type>
			</Meta>
			<Target>
				<LocURI>./Device/Vendor/MSFT/BitLocker/SystemDrivesRequireStartupAuthentication</LocURI>
			</Target>
			<Data>
			{{ if .Enabled }}
			<![CDATA[
				<enabled/>
				<data id="ConfigureNonTPMStartupKeyUsage_Name" value="{{ derefBool .ConfigureNonTPMStartupKey }}"/>
				<data id="ConfigureTPMStartupKeyUsageDropDown_Name" value="{{ derefUint .ConfigureTPMStartupKey }}"/>
				<data id="ConfigurePINUsageDropDown_Name" value="{{ derefUint .ConfigurePIN }}"/>
				<data id="ConfigureTPMPINKeyUsageDropDown_Name" value="{{ derefUint .ConfigureTPMPINKey }}"/>
				<data id="ConfigureTPMUsageDropDown_Name" value="{{ derefUint .ConfigureTPM }}"/>
			]]>
			{{ else }}
			<![CDATA[<disabled/>]]>
			{{ end }}
			</Data>
		</Item>
	</Replace>
	{{ if .ConfigurePINPolicies }}
	<Replace>
		<CmdID>{{ .CmdUUID }}-2</CmdID>
		<Item>
			<Meta>
			  <Format>chr</Format>
			  <Type>text/plain</Type>
			</Meta>
			<Target>
				<LocURI>./Device/Vendor/MSFT/BitLocker/SystemDrivesMinimumPINLength</LocURI>
			</Target>
			<Data><![CDATA[<enabled/><data id="MinPINLength" value="{{ minPINLength }}"/>]]></Data>
		</Item>
	</Replace>
	<Replace>
		<CmdID>{{ .CmdUUID }}-3</CmdID>
		<Item>
			<Meta>
			  <Format>chr</Format>
			  <Type>text/plain</Type>
			</Meta>
			<Target>
				<LocURI>./Device/Vendor/MSFT/BitLocker/SystemDrivesEnhancedPIN</LocURI>
			</Target>
			<Data><![CDATA[<enabled/>]]></Data>
		</Item>
	</Replace>
	{{- /* This node is phrased as a prohibition, so disabling it is what lets standard users change their PIN. */}}
	<Replace>
		<CmdID>{{ .CmdUUID }}-4</CmdID>
		<Item>
			<Meta>
			  <Format>chr</Format>
			  <Type>text/plain</Type>
			</Meta>
			<Target>
				<LocURI>./Device/Vendor/MSFT/BitLocker/SystemDrivesDisallowStandardUsersCanChangePIN</LocURI>
			</Target>
			<Data><![CDATA[<disabled/>]]></Data>
		</Item>
	</Replace>
	{{ end }}
</Atomic>`,
))

// SystemDriveRequiresStartupAuthSpec specification for the SystemDrivesRequireStartupAuthentication command.
// uint values must be one of the PolicyOptDropdown* variants
type SystemDriveRequiresStartupAuthSpec struct {
	CmdUUID string
	// Enabled specifies whether the 'Require additional authentication at startup'
	// policy setting should be enabled or not.
	Enabled bool
	// ConfigureNonTPMStartup allows BitLocker without a compatible TPM
	ConfigureNonTPMStartupKey *bool
	// ConfigureTPMStartupKey configures TPM startup key.
	ConfigureTPMStartupKey *uint
	// ConfigurePIN configures TPM startup PIN
	ConfigurePIN *uint
	// ConfigureTPMPINKey configures TPM startup key and PIN
	ConfigureTPMPINKey *uint
	// ConfigureTPM configures configure TPM startup
	ConfigureTPM *uint
	// ConfigurePINPolicies also sets the startup PIN policies Fleet's PIN flow assumes.
	ConfigurePINPolicies bool
}

func (spec SystemDriveRequiresStartupAuthSpec) validate() error {
	if spec.CmdUUID == "" {
		return errors.New("cmdUUID is required")
	}

	if !spec.Enabled && (spec.ConfigureNonTPMStartupKey != nil ||
		spec.ConfigureTPMStartupKey != nil ||
		spec.ConfigurePIN != nil ||
		spec.ConfigureTPMPINKey != nil ||
		spec.ConfigureTPM != nil ||
		spec.ConfigurePINPolicies) {
		return errors.New("enabled must be true if any other field is set")
	}

	validateVariants := func(name string, val *uint) error {
		if val != nil &&
			*val != PolicyOptDropdownDisallowed &&
			*val != PolicyOptDropdownRequired &&
			*val != PolicyOptDropdownOptional {
			return errors.New(name + " must be one of the PolicyOptDropdown* variants")
		}
		return nil
	}
	variantFields := map[string]*uint{
		"ConfigureTPMStartupKey": spec.ConfigureTPMStartupKey,
		"ConfigurePIN":           spec.ConfigurePIN,
		"ConfigureTPMPINKey":     spec.ConfigureTPMPINKey,
		"ConfigureTPM":           spec.ConfigureTPM,
	}
	for name, val := range variantFields {
		if err := validateVariants(name, val); err != nil {
			return err
		}
	}
	return nil
}

// SystemDriveRequiresStartupAuthCmd turns a SystemDrRequiresStartupAuthSpec into a
// https://learn.microsoft.com/en-us/windows/client-management/mdm/bitlocker-csp#systemdrivesrequirestartupauthentication
// CMD.
func SystemDriveRequiresStartupAuthCmd(spec SystemDriveRequiresStartupAuthSpec) (*fleet.MDMWindowsCommand, error) {
	if err := spec.validate(); err != nil {
		return nil, err
	}

	var contents bytes.Buffer
	if err := systemDriveRequiresStartupAuthTmpl.Execute(&contents, spec); err != nil {
		return nil, errors.New("failed to execute SystemDrRequiresStartupAuthCmd template")
	}

	return &fleet.MDMWindowsCommand{
		CommandUUID:  spec.CmdUUID,
		RawCommand:   contents.Bytes(),
		TargetLocURI: "./Device/Vendor/MSFT/BitLocker/SystemDrivesRequireStartupAuthentication",
	}, nil
}
