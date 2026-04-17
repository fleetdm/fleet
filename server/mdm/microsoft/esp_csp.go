package microsoft_mdm

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
)

// ESP item installation states as defined by the EnrollmentStatusTracking CSP.
// https://learn.microsoft.com/en-us/windows/client-management/mdm/enrollmentstatustracking-csp
const (
	ESPItemStatusNotInstalled uint = 1
	ESPItemStatusNotRequired  uint = 2
	ESPItemStatusCompleted    uint = 3
	ESPItemStatusError        uint = 4
)

// ESPTimeoutSeconds is the default timeout for the Enrollment Status Page (3 hours).
const ESPTimeoutSeconds = 10800

// providerID is the MDM provider ID used in DMClient CSP paths.
// This must match the ProviderID used in the provisioning document (see syncml.DocProvisioningAppProviderID).
const providerID = syncml.DocProvisioningAppProviderID

// ESPProfileTrackingInfo holds the information needed to track a profile on the ESP.
type ESPProfileTrackingInfo struct {
	// ProfileUUID is the Fleet profile UUID, used to build unique LocURIs.
	ProfileUUID string
	// TopLocURI is the LocURI of the first setting in the profile, used for ExpectedPolicies.
	TopLocURI string
	// HasSCEP indicates whether this profile contains SCEP certificate settings.
	HasSCEP bool
	// SCEPOnly indicates this profile contains only SCEP settings and should be
	// tracked under Certificates only, not under Security policies.
	SCEPOnly bool
}

// ESPSoftwareTrackingInfo holds the information needed to track a software item on the ESP.
type ESPSoftwareTrackingInfo struct {
	// Name is a display-friendly name for the software item, used in the tracking path.
	Name string
	// Status is the current installation status (ESPItemStatus* constant).
	Status uint
}

// ESPInitialCommandSpec defines the parameters for building the initial ESP SyncML command
// sent when a device transitions from awaiting_configuration=1 to =2.
type ESPInitialCommandSpec struct {
	CmdUUID  string
	Profiles []ESPProfileTrackingInfo
	Software []ESPSoftwareTrackingInfo
}

func (spec ESPInitialCommandSpec) validate() error {
	if spec.CmdUUID == "" {
		return errors.New("cmdUUID is required")
	}
	return nil
}

// espInitialCommandTmpl builds the initial ESP SyncML command that configures:
// - DMClient FirstSyncStatus ExpectedPolicies (one per profile top setting)
// - DMClient FirstSyncStatus TimeoutUntilSyncFailure (3-hour timeout)
// - DMClient FirstSyncStatus BlockInStatusPage (true to hold at ESP)
// - EnrollmentStatusTracking DevicePreparation/PolicyProviders entries for each profile
// - DMClient FirstSyncStatus ExpectedSCEPCerts entries for profiles with SCEP
// - EnrollmentStatusTracking DevicePreparation/PolicyProviders/.../TrackingPolicies/Apps entries for each software item
var espInitialCommandTmpl = template.Must(template.New("esp_init").Funcs(template.FuncMap{
	"escapeXML": escapeXMLString,
}).Parse(`<Atomic>
<CmdID>{{ .CmdUUID }}</CmdID>
<Replace>
<CmdID>{{ .CmdUUID }}-timeout</CmdID>
<Item>
<Meta><Format>int</Format><Type>text/plain</Type></Meta>
<Target><LocURI>./Device/Vendor/MSFT/DMClient/Provider/` + providerID + `/FirstSyncStatus/TimeOutUntilSyncFailure</LocURI></Target>
<Data>` + fmt.Sprintf("%d", ESPTimeoutSeconds) + `</Data>
</Item>
</Replace>
<Replace>
<CmdID>{{ .CmdUUID }}-block</CmdID>
<Item>
<Meta><Format>bool</Format><Type>text/plain</Type></Meta>
<Target><LocURI>./Device/Vendor/MSFT/DMClient/Provider/` + providerID + `/FirstSyncStatus/BlockInStatusPage</LocURI></Target>
<Data>true</Data>
</Item>
</Replace>
<Replace>
<CmdID>{{ .CmdUUID }}-skipuser</CmdID>
<Item>
<Meta><Format>bool</Format><Type>text/plain</Type></Meta>
<Target><LocURI>./Device/Vendor/MSFT/DMClient/Provider/` + providerID + `/FirstSyncStatus/SkipUserStatusPage</LocURI></Target>
<Data>true</Data>
</Item>
</Replace>
{{- range $i, $p := .Profiles }}
{{- if not $p.SCEPOnly }}
<Add>
<CmdID>{{ $.CmdUUID }}-policy-{{ $i }}</CmdID>
<Item>
<Meta><Format>chr</Format><Type>text/plain</Type></Meta>
<Target><LocURI>./Device/Vendor/MSFT/DMClient/Provider/` + providerID + `/FirstSyncStatus/ExpectedPolicies/{{ escapeXML $p.TopLocURI }}</LocURI></Target>
</Item>
</Add>
<Add>
<CmdID>{{ $.CmdUUID }}-pp-{{ $i }}</CmdID>
<Item>
<Target><LocURI>./Device/Vendor/MSFT/EnrollmentStatusTracking/DevicePreparation/PolicyProviders/` + providerID + `/TrackingPolicies/{{ escapeXML $p.ProfileUUID }}</LocURI></Target>
</Item>
</Add>
{{- end }}
{{- if $p.HasSCEP }}
<Add>
<CmdID>{{ $.CmdUUID }}-scep-{{ $i }}</CmdID>
<Item>
<Target><LocURI>./Device/Vendor/MSFT/DMClient/Provider/` + providerID + `/FirstSyncStatus/ExpectedSCEPCerts/{{ escapeXML $p.ProfileUUID }}</LocURI></Target>
</Item>
</Add>
{{- end }}
{{- end }}
{{- range $i, $s := .Software }}
<Add>
<CmdID>{{ $.CmdUUID }}-app-{{ $i }}</CmdID>
<Item>
<Target><LocURI>./Device/Vendor/MSFT/EnrollmentStatusTracking/DevicePreparation/PolicyProviders/` + providerID + `/TrackingPolicies/Apps/{{ escapeXML $s.Name }}</LocURI></Target>
</Item>
</Add>
<Replace>
<CmdID>{{ $.CmdUUID }}-appst-{{ $i }}</CmdID>
<Item>
<Meta><Format>int</Format><Type>text/plain</Type></Meta>
<Target><LocURI>./Device/Vendor/MSFT/EnrollmentStatusTracking/DevicePreparation/PolicyProviders/` + providerID + `/TrackingPolicies/Apps/{{ escapeXML $s.Name }}/InstallationState</LocURI></Target>
<Data>{{ $s.Status }}</Data>
</Item>
</Replace>
{{- end }}
</Atomic>`))

// ESPInitialCommand builds the SyncML command that initializes the Enrollment Status Page
// for a Windows device. This is sent when the device transitions from awaiting_configuration=1 to =2.
func ESPInitialCommand(spec ESPInitialCommandSpec) (*fleet.MDMWindowsCommand, error) {
	if err := spec.validate(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := espInitialCommandTmpl.Execute(&buf, spec); err != nil {
		return nil, fmt.Errorf("failed to execute ESP initial command template: %w", err)
	}

	return &fleet.MDMWindowsCommand{
		CommandUUID:  spec.CmdUUID,
		RawCommand:   buf.Bytes(),
		TargetLocURI: "./Device/Vendor/MSFT/DMClient/Provider/" + providerID + "/FirstSyncStatus",
	}, nil
}

// ESPStatusUpdateSpec defines the parameters for building an ESP status update
// returned inline on subsequent checkins (awaiting_configuration=2).
type ESPStatusUpdateSpec struct {
	CmdUUID  string
	Software []ESPSoftwareTrackingInfo
}

func (spec ESPStatusUpdateSpec) validate() error {
	if spec.CmdUUID == "" {
		return errors.New("cmdUUID is required")
	}
	return nil
}

var espStatusUpdateTmpl = template.Must(template.New("esp_status").Funcs(template.FuncMap{
	"escapeXML": escapeXMLString,
}).Parse(`<Atomic>
<CmdID>{{ .CmdUUID }}</CmdID>
{{- range $i, $s := .Software }}
<Replace>
<CmdID>{{ $.CmdUUID }}-appst-{{ $i }}</CmdID>
<Item>
<Meta><Format>int</Format><Type>text/plain</Type></Meta>
<Target><LocURI>./Device/Vendor/MSFT/EnrollmentStatusTracking/DevicePreparation/PolicyProviders/` + providerID + `/TrackingPolicies/Apps/{{ escapeXML $s.Name }}/InstallationState</LocURI></Target>
<Data>{{ $s.Status }}</Data>
</Item>
</Replace>
{{- end }}
</Atomic>`))

// ESPStatusUpdateCommand builds the SyncML command that updates the ESP with current
// software installation statuses. This is returned inline (not enqueued) on subsequent
// checkins while awaiting_configuration=2.
func ESPStatusUpdateCommand(spec ESPStatusUpdateSpec) (*fleet.MDMWindowsCommand, error) {
	if err := spec.validate(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := espStatusUpdateTmpl.Execute(&buf, spec); err != nil {
		return nil, fmt.Errorf("failed to execute ESP status update template: %w", err)
	}

	return &fleet.MDMWindowsCommand{
		CommandUUID:  spec.CmdUUID,
		RawCommand:   buf.Bytes(),
		TargetLocURI: "./Device/Vendor/MSFT/EnrollmentStatusTracking/DevicePreparation/PolicyProviders/" + providerID + "/TrackingPolicies",
	}, nil
}

// SetupExperienceStatusToESP converts a Fleet setup experience status to an ESP item status.
func SetupExperienceStatusToESP(status fleet.SetupExperienceStatusResultStatus) uint {
	switch status {
	case fleet.SetupExperienceStatusSuccess:
		return ESPItemStatusCompleted
	case fleet.SetupExperienceStatusFailure:
		return ESPItemStatusError
	case fleet.SetupExperienceStatusPending, fleet.SetupExperienceStatusRunning:
		return ESPItemStatusNotInstalled
	default:
		return ESPItemStatusNotInstalled
	}
}

// escapeXMLString escapes characters that are not safe in XML attribute values or text content.
func escapeXMLString(s string) string {
	var buf bytes.Buffer
	for _, c := range s {
		switch c {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(c)
		}
	}
	return buf.String()
}
