package fleet

import (
	"encoding/json"
	"time"
)

type StatisticsPayload struct {
	AnonymousIdentifier            string `json:"anonymousIdentifier"`
	FleetVersion                   string `json:"fleetVersion"`
	LicenseTier                    string `json:"licenseTier"`
	Organization                   string `json:"organization"`
	NumHostsEnrolled               int    `json:"numHostsEnrolled"`
	NumHostsABMPending             int    `json:"numHostsABMPending"`
	NumUsers                       int    `json:"numUsers"`
	NumSoftwareVersions            int    `json:"numSoftwareVersions"`
	NumHostSoftwares               int    `json:"numHostSoftwares"`
	NumSoftwareTitles              int    `json:"numSoftwareTitles"`
	NumHostSoftwareInstalledPaths  int    `json:"numHostSoftwareInstalledPaths"`
	NumSoftwareCPEs                int    `json:"numSoftwareCPEs"`
	NumSoftwareCVEs                int    `json:"numSoftwareCVEs"`
	NumTeams                       int    `json:"numTeams"` //nolint:apiparamcheck // don't want to break analytics ingestion
	NumPolicies                    int    `json:"numPolicies"`
	// NumPoliciesAutomationEnabledSoftware is the number of policies with a software
	// automation, using the same definition as the public automation_type=software
	// filter: the policy installs software, or it is a patch policy. Kept in sync with
	// policiesSoftwareAutomationClause in server/datastore/mysql/policies.go.
	NumPoliciesAutomationEnabledSoftware int `json:"numPoliciesAutomationEnabledSoftware"`
	NumQueries                     int    `json:"numQueries"` //nolint:apiparamcheck // don't want to break analytics ingestion
	NumLabels                      int    `json:"numLabels"`
	SoftwareInventoryEnabled       bool   `json:"softwareInventoryEnabled"`
	VulnDetectionEnabled           bool   `json:"vulnDetectionEnabled"`
	SystemUsersEnabled             bool   `json:"systemUsersEnabled"`
	HostsStatusWebHookEnabled      bool   `json:"hostsStatusWebHookEnabled"`
	MDMMacOsEnabled                bool   `json:"mdmMacOsEnabled"`
	HostExpiryEnabled              bool   `json:"hostExpiryEnabled"`
	MDMWindowsEnabled              bool   `json:"mdmWindowsEnabled"`
	MDMRecoveryLockPasswordEnabled bool   `json:"mdmRecoveryLockPasswordEnabled"`
	LiveQueryDisabled              bool   `json:"liveQueryDisabled"` //nolint:apiparamcheck // osquery live-query feature
	NumWeeklyActiveUsers           int    `json:"numWeeklyActiveUsers"`
	// NumWeeklyPolicyViolationDaysActual is an aggregate count of actual policy violation days. One
	// policy violation day is added for each policy that a host is failing as of the time the count
	// is incremented. The count increments once per 24-hour interval and resets each week.
	NumWeeklyPolicyViolationDaysActual int `json:"numWeeklyPolicyViolationDaysActual"`
	// NumWeeklyPolicyViolationDaysActual is an aggregate count of possible policy violation
	// days. The count is incremented by the organization's total number of policies
	// multiplied by the total number of hosts as of the time the count is incremented. The count
	// increments once per 24-hour interval and resets each week.
	NumWeeklyPolicyViolationDaysPossible int                                `json:"numWeeklyPolicyViolationDaysPossible"`
	HostsEnrolledByOperatingSystem       map[string][]HostsCountByOSVersion `json:"hostsEnrolledByOperatingSystem"`
	// HostsEnrolledByOrbitVersion is a count of hosts enrolled to Fleet grouped by orbit version
	HostsEnrolledByOrbitVersion []HostsCountByOrbitVersion `json:"hostsEnrolledByOrbitVersion"`
	// HostsEnrolledByOsqueryVersion is a count of hosts enrolled to Fleet grouped by osquery version
	HostsEnrolledByOsqueryVersion []HostsCountByOsqueryVersion `json:"hostsEnrolledByOsqueryVersion"`
	StoredErrors                  json.RawMessage              `json:"storedErrors"`
	// NumHostsNotResponding is a count of hosts that connect to Fleet successfully but fail to submit results for distributed queries.
	NumHostsNotResponding int `json:"numHostsNotResponding"`
	// Whether server_settings.ai_features_disabled is set to true in the config.
	AIFeaturesDisabled bool `json:"aiFeaturesDisabled"`
	// Whether at least one team has integrations.google_calendar.enable_calendar_events set to true
	MaintenanceWindowsEnabled bool `json:"maintenanceWindowsEnabled"`
	// Maintenance windows are considered "configured" if:
	// configuration has value set for integrations.google_calendar[0].domain
	// configuration has value set for integrations.google_calendar[0].api_key_json
	MaintenanceWindowsConfigured bool `json:"maintenanceWindowsConfigured"`
	// GoogleWorkspaceConfigured is true when a Google Workspace IdP integration is
	// configured (integrations.google_workspace[0] has a domain and service account).
	GoogleWorkspaceConfigured bool `json:"googleWorkspaceConfigured"`
	// The number of hosts with Fleet desktop installed.
	NumHostsFleetDesktopEnabled int `json:"numHostsFleetDesktopEnabled"`
	// FleetMaintainedAppsMacOS is the set of Fleet-maintained apps being used on macOS
	FleetMaintainedAppsMacOS []FleetMaintainedAppUsage `json:"fleetMaintainedAppsMacOS,omitempty"`
	// FleetMaintainedAppsWindows is the set of Fleet-maintained apps being used on Windows
	FleetMaintainedAppsWindows []FleetMaintainedAppUsage `json:"fleetMaintainedAppsWindows,omitempty"`

	// ConditionalAccessEnabled indicates whether any team has conditional access enabled.
	ConditionalAccessEnabled bool `json:"conditionalAccessEnabled"`
	// OktaConditionalAccessConfigured indicates if the Okta conditional access integration is configured.
	OktaConditionalAccessConfigured bool `json:"oktaConditionalAccessConfigured"`
	// ConditionalAccessBypassDisabled indicates if bypass is disabled for Okta.
	ConditionalAccessBypassDisabled bool `json:"conditionalAccessBypassDisabled"`
	// EntraConditionalAccessConfigured indicates if the Entra conditional access integration is configured.
	EntraConditionalAccessConfigured bool `json:"entraConditionalAccessConfigured"`

	// GitOpsModeEnabled indicates whether GitOps mode is enabled in the app config.
	GitOpsModeEnabled bool `json:"gitOpsModeEnabled"`
	// GitOpsModeExceptions lists the configured GitOps mode exceptions (e.g. "labels", "software", "secrets").
	// Exceptions are persisted independently of GitOpsModeEnabled.
	GitOpsModeExceptions []string `json:"gitOpsModeExceptions"`

	// NumHostsFleetMDMEnrolledMacOS is the number of macOS hosts actually enrolled in Fleet's own MDM
	NumHostsFleetMDMEnrolledMacOS int `json:"numHostsFleetMDMEnrolledMacOS"`
	// NumHostsFleetMDMEnrolledWindows is the number of Windows hosts actually enrolled in Fleet's own MDM
	NumHostsFleetMDMEnrolledWindows int `json:"numHostsFleetMDMEnrolledWindows"`
}

// FleetMaintainedAppUsage reports a Fleet-maintained app in use, whether a patch policy
// covers it, and whether that patch policy carries a software automation.
//
// The patch policy is matched on the app's software title, so two slugs that share a title
// (Firefox GA and ESR) report the same patch policy.
type FleetMaintainedAppUsage struct {
	// Name is the Fleet-maintained app slug, e.g. "1password/darwin".
	Name               string `json:"name" db:"name"`
	PatchPolicy        bool   `json:"patchPolicy" db:"patch_policy"`
	SoftwareAutomation bool   `json:"softwareAutomation" db:"software_automation"`
}

type HostsCountByOrbitVersion struct {
	OrbitVersion string `json:"orbitVersion" db:"orbit_version"`
	NumHosts     int    `json:"numHosts" db:"num_hosts"`
}
type HostsCountByOsqueryVersion struct {
	OsqueryVersion string `json:"osqueryVersion" db:"osquery_version"`
	NumHosts       int    `json:"numHosts" db:"num_hosts"`
}

type HostsCountByOSVersion struct {
	Version     string `json:"version"`
	NumEnrolled int    `json:"numEnrolled"`
}

const (
	StatisticsFrequency = time.Hour * 1
)
