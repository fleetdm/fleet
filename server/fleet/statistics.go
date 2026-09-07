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
	NumQueries                     int    `json:"numQueries"` //nolint:apiparamcheck // don't want to break analytics ingestion
	NumLabels                      int    `json:"numLabels"`
	SoftwareInventoryEnabled       bool   `json:"softwareInventoryEnabled"`
	VulnDetectionEnabled           bool   `json:"vulnDetectionEnabled"`
	SystemUsersEnabled             bool   `json:"systemUsersEnabled"`
	HostsStatusWebHookEnabled      bool   `json:"hostsStatusWebHookEnabled"`
	MDMMacOsEnabled                bool   `json:"mdmMacOsEnabled"`
	HostExpiryEnabled              bool   `json:"hostExpiryEnabled"`
	MDMWindowsEnabled              bool   `json:"mdmWindowsEnabled"`
	MDMAndroidEnabled              bool   `json:"mdmAndroidEnabled"`
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
	// IDPGoogleWorkspaceConfigured is true when a Google Workspace IdP integration is
	// configured (integrations.google_workspace[0] has a domain and service account).
	IDPGoogleWorkspaceConfigured bool `json:"idpGoogleWorkspaceConfigured"`
	// The number of hosts with Fleet desktop installed.
	NumHostsFleetDesktopEnabled int `json:"numHostsFleetDesktopEnabled"`
	// FleetMaintainedAppsMacOS is an array of Fleet-maintained app slugs being used on macOS
	FleetMaintainedAppsMacOS []string `json:"fleetMaintainedAppsMacOS,omitempty"`
	// FleetMaintainedAppsWindows is an array of Fleet-maintained app slugs being used on Windows
	FleetMaintainedAppsWindows []string `json:"fleetMaintainedAppsWindows,omitempty"`

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

	// FleetDesktopSSOEnabled is true when SSO is required in front of Fleet Desktop (fleet_desktop.sso_enabled).
	FleetDesktopSSOEnabled bool `json:"fleetDesktopSSOEnabled"`

	// NumHostsFleetMDMEnrolledMacOS is the number of macOS hosts actually enrolled in Fleet's own MDM
	NumHostsFleetMDMEnrolledMacOS int `json:"numHostsFleetMDMEnrolledMacOS"`
	// NumHostsFleetMDMEnrolledWindows is the number of Windows hosts actually enrolled in Fleet's own MDM
	NumHostsFleetMDMEnrolledWindows int `json:"numHostsFleetMDMEnrolledWindows"`
	// NumMDMAppleProfiles is the number of Apple (macOS/iOS) configuration profiles defined across all teams
	NumMDMAppleProfiles int `json:"numMDMAppleProfiles"`
	// NumMDMWindowsProfiles is the number of Windows configuration profiles defined across all teams
	NumMDMWindowsProfiles int `json:"numMDMWindowsProfiles"`
	// NumMDMAppleDeclarations is the number of Apple DDM declarations defined across all teams
	NumMDMAppleDeclarations int `json:"numMDMAppleDeclarations"`
	// NumMDMAndroidProfiles is the number of Android configuration profiles defined across all teams
	NumMDMAndroidProfiles int `json:"numMDMAndroidProfiles"`

	// The *LogDestination fields are the configured server-side log plugins
	// (osquery.result_log_plugin, osquery.status_log_plugin, activity.audit_log_plugin),
	// e.g. "filesystem" or "firehose". They come from the server config, not the app config.
	ResultLogDestination string `json:"resultLogDestination"`
	StatusLogDestination string `json:"statusLogDestination"`
	AuditLogDestination  string `json:"auditLogDestination"`

	// Vulnerabilities webhooks are a global-only setting, so despite the "any" prefix this tracks
	// just the one global flag.
	AnyVulnerabilitiesWebhookEnabled bool `json:"anyVulnerabilitiesWebhookEnabled"`
	// AnyFailingPoliciesWebhookEnabled and AnyHostActivitiesWebhookEnabled are true when the
	// webhook is enabled globally or on any fleet, including "No team".
	AnyFailingPoliciesWebhookEnabled bool `json:"anyFailingPoliciesWebhookEnabled"`
	AnyHostActivitiesWebhookEnabled  bool `json:"anyHostActivitiesWebhookEnabled"`
	GlobalActivityWebhookEnabled     bool `json:"globalActivityWebhookEnabled"`

	// A ticket destination is "configured" when a Jira or Zendesk integration exists in the global
	// config; fleet-level entries only toggle an already-configured destination.
	TicketDestinationConfigured bool `json:"ticketDestinationConfigured"`
	// SSOConfiguredFleetUsers covers logging in to Fleet; SSOConfiguredEndUsers covers end user
	// authentication during MDM enrollment.
	SSOConfiguredFleetUsers bool `json:"ssoConfiguredFleetUsers"`
	SSOConfiguredEndUsers   bool `json:"ssoConfiguredEndUsers"`
	// Apple account provisioning (Platform SSO).
	AccountProvisioningConfigured bool `json:"accountProvisioningConfigured"`
	// IDPSCIMConfigured is true once Fleet has received a SCIM request, the same signal the IdP
	// settings page uses to report the connection.
	IDPSCIMConfigured              bool `json:"idpSCIMConfigured"`
	CertificateAuthorityConfigured bool `json:"certificateAuthorityConfigured"`
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
