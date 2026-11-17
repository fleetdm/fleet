export type TierQuestion = {
  id: string;
  feature: string;
  // The correct tier for this feature.
  answer: "free" | "premium";
  // One-sentence explanation shown after the user picks.
  explanation: string;
};

export const TIER_QUESTIONS: TierQuestion[] = [
  // ─── Surprising premium ones ───
  { id: "p-install-admin", feature: "Clicking Install on a host's software library", answer: "premium", explanation: "svc.InstallSoftwareTitle returns ErrMissingLicense in core. Every software-install path through Fleet is premium." },
  { id: "p-install-self", feature: "Self-service installs from the My Device page", answer: "premium", explanation: "SelfServiceInstallSoftwareTitle is premium-wrapped. Not just install actions. The whole surface." },
  { id: "p-uninstall", feature: "Uninstalling software from the admin UI", answer: "premium", explanation: "Mirror of InstallSoftwareTitle. Same gate." },
  { id: "p-upload-installer", feature: "Uploading a custom .pkg / .msi / .deb installer", answer: "premium", explanation: "UploadSoftwareInstaller fails on Free. Even just storing an installer in Fleet is premium." },
  { id: "p-fma-browse", feature: "Browsing the Fleet-maintained apps catalog", answer: "premium", explanation: "ListFleetMaintainedApps returns ErrMissingLicense. You can't even look without a license." },
  { id: "p-vpp", feature: "VPP installs (Apple Business Manager volume license)", answer: "premium", explanation: "All VPP methods. Token upload, catalog sync, assignment. Are premium." },
  { id: "p-inhouse", feature: "In-house / private app installs (private IPA, private .pkg)", answer: "premium", explanation: "Separate from VPP; still fully premium." },
  { id: "p-run-script", feature: "Running an ad-hoc script on a host", answer: "premium", explanation: "RunHostScript is premium. Scripts via policy automations and setup-experience scripts are too." },
  { id: "p-wipe", feature: "Wiping a host remotely", answer: "premium", explanation: "WipeHost (and LockHost / UnlockHost) are premium-only in core." },
  { id: "p-fleets", feature: "Creating a second fleet (team) with its own config", answer: "premium", explanation: "Multi-tenancy is the single biggest tier boundary. Free is one global config." },
  { id: "p-sso", feature: "Logging in with SAML or OIDC SSO", answer: "premium", explanation: "Local email+password is the only free auth path. All IdP-connected auth is premium." },
  { id: "p-scim", feature: "SCIM user+group provisioning from Okta/Entra", answer: "premium", explanation: "SCIM is premium. Fleet is a SCIM target when provisioned via SSO IdPs." },
  { id: "p-end-user-sso", feature: "End-user SSO on the My Device page", answer: "premium", explanation: "Separate flag from admin SSO; also premium." },
  { id: "p-ddm-custom-activation", feature: "DDM custom activation declarations (vs standard Fleet-managed activation)", answer: "premium", explanation: "Base DDM declarations work on Free. Custom activation declarations hit a specific premium gate. See svc.handleCustomActivation in server/service/apple_mdm.go." },
  { id: "p-bootstrap", feature: "Bootstrap packages during Apple DEP enrollment", answer: "premium", explanation: "Create / download / delete all return ErrMissingLicense in core." },
  { id: "p-setup-exp", feature: "Setup experience software + scripts (pre-login installs)", answer: "premium", explanation: "The whole setup-experience branch is premium. Software list, scripts, next-step state machine." },
  { id: "p-psso", feature: "Platform SSO (PSSO) for Macs", answer: "premium", explanation: "All 5 PSSO service methods are premium-only. There is no free PSSO path." },
  { id: "p-ca", feature: "Registering an external SCEP / ACME certificate authority", answer: "premium", explanation: "Cert authority CRUD + proxied issuance is premium. SCEP/ACME providers can't be wired on Free." },
  { id: "p-autopilot", feature: "Syncing Autopilot device registrations from Microsoft Graph", answer: "premium", explanation: "The microsoft_autopilot_sync cron is premium-only. Free can't ingest Autopilot registrations." },
  { id: "p-pol-automation-ticket", feature: "Policy automation: open a Jira ticket when a policy fails", answer: "premium", explanation: "All policy automation actions are premium (ticket, webhook, script, install, calendar)." },
  { id: "p-pol-automation-install", feature: "Policy automation: install remediation software on failing hosts", answer: "premium", explanation: "Routes to the same InstallSoftwareTitle code path as the admin button. Premium end-to-end." },
  { id: "p-vuln-auto", feature: "Vulnerability automation: open a ticket when a new CVE matches", answer: "premium", explanation: "Vuln detection is free, but the automation layer that reacts to new matches is premium." },
  { id: "p-activity-stream", feature: "Streaming activity events to a SIEM or webhook", answer: "premium", explanation: "Local activity log is free; streaming it to external destinations is premium." },
  { id: "p-compliance-partner", feature: "Microsoft compliance partner (Entra conditional access)", answer: "premium", explanation: "Fleet reports device compliance to Entra; Entra gates app access. Premium integration." },
  { id: "p-gitops-batch-software", feature: "GitOps batch apply of software installers across fleets", answer: "premium", explanation: "BatchSetSoftwareInstallers is premium. Even dry-run against premium content fails on Free." },

  // ─── Surprising free ones ───
  { id: "f-mdm-enroll", feature: "Enrolling a Mac or Windows device into MDM", answer: "free", explanation: "Core MDM enrollment is free. The things you can DO after enrollment is where premium kicks in." },
  { id: "f-mobileconfig", feature: "Delivering .mobileconfig / SyncML config profiles to devices", answer: "free", explanation: "Config-profile delivery itself is free. Base DDM declarations are also free; bootstrap packages and PSSO are premium." },
  { id: "f-ddm-base", feature: "Uploading and delivering a standard DDM declaration", answer: "free", explanation: "Base DDM works on Free. Only custom activation, label scoping, and OS-update declarations trip the premium gate." },
  { id: "f-live-queries", feature: "Running live (ad-hoc) osquery queries against hosts", answer: "free", explanation: "The whole live-query path (campaigns, pubsub, WebSocket to UI) is free." },
  { id: "f-cve", feature: "Detecting known CVEs on installed software (vuln pipeline)", answer: "free", explanation: "NVD / OSV / OVAL / MSRC matching is free. Automation on top of the matches is premium." },
  { id: "f-inventory", feature: "Viewing which software is installed on a host", answer: "free", explanation: "host_software is populated by osquery detail queries. Pure read; no premium gate." },
  { id: "f-policy-basic", feature: "Writing a policy with SQL + pass/fail evaluation (no automation action)", answer: "free", explanation: "Policy authoring and evaluation is free. Automation reactions (ticket, webhook, script, install) are the premium layer." },
  { id: "f-policy-library", feature: "Browsing + enabling policies from the Fleet-maintained policy library", answer: "free", explanation: "The maintained POLICY catalog is free (CIS benchmarks, common hardening checks). Only maintained APPS require premium to list." },
  { id: "f-saved-report", feature: "Creating a scheduled report (saved query) at global scope", answer: "free", explanation: "NewQuery is free for the base case. Attaching labels, per-fleet scoping, or several advanced flags trips the premium gate." },
  { id: "f-local-auth", feature: "Admin login with local email + password", answer: "free", explanation: "Local auth is the baseline. Only IdP-backed SSO (SAML / OIDC) and SCIM provisioning require premium." },
  { id: "f-api-token", feature: "Creating long-lived API tokens for an admin account", answer: "free", explanation: "API tokens live in the sessions table; no premium gate on issuance." },
  { id: "f-activity-log", feature: "Viewing the activity log in Fleet's admin UI", answer: "free", explanation: "In-app activity log is free. Streaming those events to an external SIEM or webhook is premium." },
  { id: "f-dashboard", feature: "Dashboard tiles: host counts, OS summary, policy pass rate", answer: "free", explanation: "Aggregate rollups are populated from scheduled queries + policy_membership, both of which are free." },
  { id: "f-host-export", feature: "Exporting the host list as CSV or JSON", answer: "free", explanation: "Host list reads are free. Export happens entirely in the UI from an endpoint that has no premium gate." },
  { id: "f-fleetctl-get", feature: "Reading Fleet config with `fleetctl get` commands", answer: "free", explanation: "`fleetctl get` is a thin wrapper over the admin GET endpoints. Same license surface as viewing the UI." },
];
