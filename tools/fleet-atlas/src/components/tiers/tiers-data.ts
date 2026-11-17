export type TierSupport = "yes" | "no" | "limited";

export type FeatureRow = {
  feature: string;
  free: TierSupport;
  premium: TierSupport;
  note?: string;
};

export type FeatureCategory = {
  slug: string;
  label: string;
  blurb: string;
  rows: FeatureRow[];
};

// Code anchors in Fleet for the premium check pattern.
export const CODE_ANCHORS = {
  tierConst: {
    path: "server/fleet/app.go",
    hint: "TierPremium = \"premium\" / TierFree = \"free\"",
  },
  isPremium: {
    path: "server/service/appconfig.go",
    hint: "lic.IsPremium(). The sync check used in service code",
  },
  isPremiumCtx: {
    path: "server/service/activities.go",
    hint: "license.IsPremium(ctx). Ctx-flavored variant for middleware",
  },
  errMissingLicense: {
    path: "server/fleet/errors.go",
    hint: "ErrMissingLicense. Sentinel error for premium gating",
  },
  eePackage: {
    path: "ee/server/service/",
    hint: "The enterprise wrapper that overrides core methods per-license",
  },
};

export const FEATURE_MATRIX: FeatureCategory[] = [
  {
    slug: "fleets",
    label: "Fleets (teams)",
    blurb:
      "Multi-tenancy is the single biggest tier boundary. On Free, everything is one global config and one shared enroll secret.",
    rows: [
      { feature: "A single global config", free: "yes", premium: "yes" },
      {
        feature: "Multiple fleets with per-fleet config",
        free: "no",
        premium: "yes",
        note: "Separate agent options, enroll secrets, labels, software scopes.",
      },
      {
        feature: "Per-fleet role assignments (observer / maintainer / admin)",
        free: "no",
        premium: "yes",
        note: "Free only has global admin / observer; no scoped roles.",
      },
      {
        feature: "Per-fleet enroll secrets with rotation",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Transfer hosts between fleets",
        free: "no",
        premium: "yes",
      },
    ],
  },
  {
    slug: "mdm",
    label: "MDM",
    blurb:
      "Core enrollment + config profile delivery is in Free. Essentially everything *added* to that (bootstrap packages, DDM, VPP, PSSO, EULA, multi-ABM, migration tool, disk-encryption summary) is premium. Apple MDM has 43 ErrMissingLicense gates in the core.",
    rows: [
      { feature: "Apple + Windows + Android enrollment", free: "yes", premium: "yes" },
      { feature: ".mobileconfig / SyncML config profiles", free: "yes", premium: "yes" },
      {
        feature: "Bootstrap packages (DEP)",
        free: "no",
        premium: "yes",
        note: "Create / download / delete all return ErrMissingLicense in core.",
      },
      {
        feature: "Setup assistant JSON (custom DEP)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Setup experience software + scripts",
        free: "no",
        premium: "yes",
      },
      {
        feature: "DDM (declarative device management). Base declarations",
        free: "yes",
        premium: "yes",
        note: "Uploading and delivering standard DDM declarations works on Free.",
      },
      {
        feature: "DDM: custom activation declarations",
        free: "no",
        premium: "yes",
        note: "License gate in server/service/apple_mdm.go svc.validateActivation (returns NewLicenseErrorWithCause).",
      },
      {
        feature: "DDM: label scoping on declarations",
        free: "no",
        premium: "yes",
      },
      {
        feature: "DDM: OS software-update declarations",
        free: "no",
        premium: "yes",
      },
      {
        feature: "VPP (volume purchase)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "In-house apps (private IPAs)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Platform SSO (PSSO)",
        free: "no",
        premium: "yes",
        note: "All 5 PSSO methods are premium-only.",
      },
      {
        feature: "Multi-ABM / multi-token support",
        free: "no",
        premium: "yes",
      },
      {
        feature: "MDM EULA (accept-to-enroll)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Account-driven enrollment SSO",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Disk-encryption summary dashboard",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Conditional access (Entra compliance partner)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Windows Autopilot sync",
        free: "no",
        premium: "yes",
      },
      {
        feature: "MDM migration tool",
        free: "no",
        premium: "yes",
      },
      {
        feature: "OS version management + auto-update",
        free: "no",
        premium: "yes",
      },
    ],
  },
  {
    slug: "software",
    label: "Software installs",
    blurb:
      "Reading software inventory is free (osquery populates it). Everything that *writes* software to a host. Admin button, self-service, GitOps, automations, every catalog type. Is premium. The core service methods return ErrMissingLicense; the ee/ wrapper implements them.",
    rows: [
      {
        feature: "View software inventory on a host",
        free: "yes",
        premium: "yes",
        note: "osquery detail queries populate host_software; no premium gate on reading.",
      },
      {
        feature: "Upload a custom installer (.pkg / .msi / .deb)",
        free: "no",
        premium: "yes",
        note: "UploadSoftwareInstaller returns ErrMissingLicense in core.",
      },
      {
        feature: "Install any software from the admin UI",
        free: "no",
        premium: "yes",
        note: "svc.InstallSoftwareTitle is a premium-wrapped method.",
      },
      {
        feature: "Uninstall software from the admin UI",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Self-service install (My Device page)",
        free: "no",
        premium: "yes",
        note: "SelfServiceInstallSoftwareTitle returns ErrMissingLicense in core.",
      },
      {
        feature: "GitOps software batch apply (POST /software/batch)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Fleet-maintained apps (FMA). Browse the catalog",
        free: "no",
        premium: "yes",
        note: "ListFleetMaintainedApps itself returns ErrMissingLicense.",
      },
      {
        feature: "Fleet-maintained apps. Install",
        free: "no",
        premium: "yes",
      },
      {
        feature: "FMA auto-update (nightly cron)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "VPP installs",
        free: "no",
        premium: "yes",
      },
      {
        feature: "In-house / private app installs",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Software categories (create / list / delete)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Install automation from a failing policy",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Install automation from a new vuln",
        free: "no",
        premium: "yes",
      },
    ],
  },
  {
    slug: "scripts",
    label: "Scripts & host actions",
    blurb:
      "Fleet can execute scripts on hosts and perform remote lifecycle actions. All of it is premium. RunHostScript, LockHost, UnlockHost, WipeHost all return ErrMissingLicense in the core service.",
    rows: [
      {
        feature: "Run an ad-hoc script on a host",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Lock a host remotely",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Unlock a host remotely",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Wipe a host remotely",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Setup experience scripts (Mac)",
        free: "no",
        premium: "yes",
      },
    ],
  },
  {
    slug: "policies",
    label: "Policies & automations",
    blurb:
      "Basic policy evaluation (write SQL, see pass/fail) is free. Automation reactions AND several per-policy flags (continuous automations, patch-when-closed, hidden, profile associations) are premium.",
    rows: [
      { feature: "Write + evaluate policies (pass/fail)", free: "yes", premium: "yes" },
      { feature: "Fleet-maintained policy library", free: "yes", premium: "yes" },
      { feature: "Policy results in dashboard + per-host", free: "yes", premium: "yes" },
      {
        feature: "Continuous automations flag on a policy",
        free: "no",
        premium: "yes",
        note: "modifyPolicy / ApplyPolicySpecs return ErrMissingLicense when this field is set.",
      },
      {
        feature: "Patch-when-closed, NotifyBeforePatching, PatchDeadline",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Hidden policies",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Policy linked to a config profile (ProfileUUID)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Automation: create Jira / Zendesk ticket on fail",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Automation: webhook on fail",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Automation: run remediation script",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Automation: install remediation software",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Automation: book calendar event with user",
        free: "no",
        premium: "yes",
      },
    ],
  },
  {
    slug: "reports",
    label: "Reports & live queries",
    blurb:
      "Base writing + running of saved reports and live queries is free. Advanced query features (labels, per-fleet scoping, several flags) hit the premium gate inside NewQuery / modifyLoadedQuery / ApplyQuerySpecs.",
    rows: [
      { feature: "Live queries (ad-hoc, streaming)", free: "yes", premium: "yes" },
      { feature: "Base saved reports (global scope)", free: "yes", premium: "yes" },
      { feature: "Fleet-maintained query library", free: "yes", premium: "yes" },
      {
        feature: "Saved report scoped to specific labels",
        free: "no",
        premium: "yes",
        note: "LabelsIncludeAny / LabelsExcludeAny on a query trigger the premium gate.",
      },
      {
        feature: "Per-fleet query performance overrides",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Discard-data flag on a report",
        free: "yes",
        premium: "yes",
      },
    ],
  },
  {
    slug: "gitops",
    label: "GitOps",
    blurb:
      "GitOps itself is free. The surface area it reconciles grows with premium (fleets, SSO, automations, software scoping).",
    rows: [
      {
        feature: "fleetctl gitops apply + --dry-run",
        free: "yes",
        premium: "yes",
      },
      {
        feature: "Apply global config + resources (policies, profiles, software, scripts)",
        free: "yes",
        premium: "yes",
      },
      {
        feature: "Apply per-fleet (team) YAML",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Declare SSO / SCIM / integrations in YAML",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Secret variables for sensitive values",
        free: "no",
        premium: "yes",
      },
    ],
  },
  {
    slug: "vuln",
    label: "Vulnerability management",
    blurb:
      "Detection (CPE match against NVD/OSV/OVAL/MSRC) is free. Response automation and prioritized surfacing are premium.",
    rows: [
      { feature: "CVE detection + CVSS display", free: "yes", premium: "yes" },
      { feature: "Software inventory per host", free: "yes", premium: "yes" },
      {
        feature: "EPSS exploit-probability scores + Known-Exploited-Vulns badge",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Vuln → ticket automation",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Vuln → auto-install remediation",
        free: "no",
        premium: "yes",
      },
    ],
  },
  {
    slug: "access",
    label: "Access control & identity",
    blurb:
      "Local user + API tokens are the free path. Everything IdP-connected is premium. Certificate-authority registrations (SCEP / ACME providers) are also premium.",
    rows: [
      { feature: "Local email + password auth", free: "yes", premium: "yes" },
      { feature: "API tokens (per-user)", free: "yes", premium: "yes" },
      {
        feature: "SSO: SAML + OIDC",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Just-in-time user provisioning on first SSO login",
        free: "no",
        premium: "yes",
      },
      {
        feature: "SCIM: push user + group lifecycle from the IdP",
        free: "no",
        premium: "yes",
      },
      {
        feature: "End-user SSO on My Device page",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Register certificate authorities (SCEP / ACME providers)",
        free: "no",
        premium: "yes",
        note: "All CA CRUD methods are premium. Issuance + proxying + rotation.",
      },
    ],
  },
  {
    slug: "integrations",
    label: "Integrations & streaming",
    blurb:
      "Local activity log is free. Any external destination (SIEM, ticketing, webhook on activity) is premium.",
    rows: [
      { feature: "In-app activity log", free: "yes", premium: "yes" },
      {
        feature: "Activity streaming to SIEM / webhook",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Jira integration (ticket on policy / vuln)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Zendesk integration (ticket on policy / vuln)",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Google Workspace / Microsoft calendar integration",
        free: "no",
        premium: "yes",
      },
    ],
  },
  {
    slug: "support",
    label: "Support & operations",
    blurb:
      "Operational guarantees Fleet (the company) provides are tier-gated. The product itself runs the same way.",
    rows: [
      {
        feature: "Community support (Slack / GitHub issues)",
        free: "yes",
        premium: "yes",
      },
      {
        feature: "SLA-backed support channels",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Managed Fleet cloud hosting",
        free: "no",
        premium: "yes",
      },
      {
        feature: "Private slack with Fleet engineers",
        free: "no",
        premium: "yes",
      },
    ],
  },
];

export function countPremiumOnlyRows(): number {
  let n = 0;
  for (const c of FEATURE_MATRIX) {
    for (const r of c.rows) {
      if (r.free === "no" && r.premium === "yes") n += 1;
    }
  }
  return n;
}

export function countSharedRows(): number {
  let n = 0;
  for (const c of FEATURE_MATRIX) {
    for (const r of c.rows) {
      if (r.free === "yes" && r.premium === "yes") n += 1;
    }
  }
  return n;
}
