import { isDarkMode } from "utilities/theme";
import paths from "router/paths";
import { ITeamSummary } from "interfaces/team";
import { IConfig } from "interfaces/config";

export interface ICommandSubItem {
  id: string;
  label: string;
  path: string;
  keywords?: string[];
}

export interface ICommandItem {
  id: string;
  label: string;
  group: string;
  path?: string;
  keywords?: string[];
  /** When set, displays the team/fleet name right-aligned in the item row */
  teamName?: string;
  /** Nested items shown when the parent is expanded via chevron */
  subItems?: ICommandSubItem[];
  /** Custom action instead of navigation */
  onAction?: () => void;
}

export interface ICommandPaletteContext {
  search: string;
  currentTeam?: ITeamSummary;
  config: IConfig | null;
  canAccessControls?: boolean;
  canWrite?: boolean;
  canAccessSettings?: boolean;
  canManagePolicyAutomations?: boolean;
  canManageSoftwareAutomations?: boolean;
  isTechnician?: boolean;
  isPremiumTier?: boolean;
  isMacMdmEnabledAndConfigured?: boolean;
  isWindowsMdmEnabledAndConfigured?: boolean;
  isAndroidMdmEnabledAndConfigured?: boolean;
  isVppEnabled?: boolean;
  hasTeamSelected?: boolean;
  withTeamId: (path: string) => string;
  onToggleDarkMode: () => void;
}

export const GROUPS = [
  "Pages",
  "Controls",
  "Software",
  "Settings",
  "MDM",
  "Automations",
  "Actions",
] as const;

export const buildCommandItems = (
  ctx: ICommandPaletteContext
): ICommandItem[] => {
  const {
    search,
    currentTeam,
    config,
    canAccessControls,
    canWrite,
    canAccessSettings,
    canManagePolicyAutomations,
    canManageSoftwareAutomations,
    isTechnician,
    isPremiumTier,
    isMacMdmEnabledAndConfigured,
    isWindowsMdmEnabledAndConfigured,
    isAndroidMdmEnabledAndConfigured,
    isVppEnabled,
    hasTeamSelected,
    withTeamId,
    onToggleDarkMode,
  } = ctx;

  const isAbmConfigured = config?.mdm?.apple_bm_enabled_and_configured ?? false;

  // Compute teamName from currentTeam — single source of truth
  const isUnassigned = currentTeam?.id === 0;
  // eslint-disable-next-line no-nested-ternary
  const teamName = hasTeamSelected
    ? currentTeam?.name
    : isUnassigned
    ? "Unassigned"
    : "All fleets";

  // Actions that target the unassigned team when no team is selected
  const teamOrUnassigned = hasTeamSelected ? teamName : "Unassigned";

  // Whether a specific team OR unassigned is selected (not "All fleets")
  const hasTeamOrUnassigned = hasTeamSelected || isUnassigned;

  return [
    // Pages — always visible
    {
      id: "dashboard",
      label: "Dashboard",
      group: "Pages",
      path: withTeamId(paths.DASHBOARD),
      keywords: ["home", "hosts", "activity", "platform"],
    },
    {
      id: "hosts",
      label: "Hosts",
      group: "Pages",
      path: withTeamId(paths.MANAGE_HOSTS),
      keywords: ["devices", "hostname", "serial number", "manage"],
    },
    ...(canAccessControls
      ? [
          {
            id: "controls-page",
            label: "Controls",
            group: "Pages",
            path: withTeamId(paths.CONTROLS),
            keywords: ["mdm", "os settings", "os updates"],
          },
        ]
      : []),
    {
      id: "software-page",
      label: "Software",
      group: "Pages",
      path: withTeamId(paths.SOFTWARE_INVENTORY),
      keywords: ["installed", "inventory", "titles", "library", "managed"],
    },
    {
      id: "reports",
      label: "Reports",
      group: "Pages",
      path: withTeamId(paths.MANAGE_REPORTS),
      keywords: ["report", "sql", "gather data", "live query"],
    },
    {
      id: "policies",
      label: "Policies",
      group: "Pages",
      path: withTeamId(paths.MANAGE_POLICIES),
      keywords: ["compliance", "failing", "device health"],
    },
    ...(canAccessSettings
      ? [
          {
            id: "settings-page",
            label: "Settings",
            group: "Pages",
            path: paths.ADMIN_SETTINGS,
            keywords: ["admin", "organization", "integrations"],
          },
        ]
      : []),
    {
      id: "labels",
      label: "Labels",
      group: "Pages",
      path: paths.MANAGE_LABELS,
      keywords: ["group hosts", "filter", "dynamic", "manual"],
    },
    ...(canAccessSettings
      ? [
          {
            id: "users-page",
            label: "Users",
            group: "Pages",
            path: paths.ADMIN_USERS,
            keywords: [
              "accounts",
              "admins",
              "invite",
              "add user",
              "edit user",
              "delete user",
            ],
          },
        ]
      : []),
    {
      id: "my-account",
      label: "My account",
      group: "Pages",
      path: paths.ACCOUNT,
      keywords: ["profile", "password", "api token", "settings"],
    },

    // Packs — only visible when searching for specific terms
    ...(/packs|create new pack|add new pack/.test(search.toLowerCase())
      ? [
          {
            id: "packs",
            label: "Packs",
            group: "Pages",
            path: paths.MANAGE_PACKS,
            keywords: ["packs", "legacy", "scheduled queries"],
          },
          {
            id: "new-pack",
            label: "Create new pack",
            group: "Actions",
            path: paths.NEW_PACK,
            keywords: ["packs", "add new pack", "create new pack"],
          },
        ]
      : []),

    // Controls — maintainers, admins, technicians only
    ...(canAccessControls
      ? [
          {
            id: "controls-os-updates",
            label: "OS updates",
            group: "Controls",
            path: withTeamId(paths.CONTROLS_OS_UPDATES),
            keywords: [
              "minimum version",
              "deadline",
              "nudge",
              "macos",
              "windows",
              "ios",
              "ipados",
              "patch",
            ],
          },
          // OS settings sub-pages
          {
            id: "controls-os-settings",
            label: "OS settings",
            group: "Controls",
            path: withTeamId(paths.CONTROLS_OS_SETTINGS),
            keywords: ["enforce", "remotely", "profiles"],
            subItems: [
              {
                id: "controls-disk-encryption",
                label: "Disk encryption",
                path: withTeamId(paths.CONTROLS_DISK_ENCRYPTION),
                keywords: ["filevault", "bitlocker", "recovery key"],
              },
              {
                id: "controls-custom-settings",
                label: "Configuration profiles",
                path: withTeamId(paths.CONTROLS_CUSTOM_SETTINGS),
                keywords: [
                  "custom profiles",
                  "mobileconfig",
                  "deploy",
                  "ddm",
                  "windows csp",
                ],
              },
              // Certificates and Passwords — not available to technicians
              ...(!isTechnician
                ? [
                    {
                      id: "controls-certificates",
                      label: "Certificates",
                      path: withTeamId(paths.CONTROLS_CERTIFICATES),
                      keywords: [
                        "scep",
                        "est",
                        "pki",
                        "digicert",
                        "ndes",
                        "certificate authority",
                        "ca",
                      ],
                    },
                    {
                      id: "controls-passwords",
                      label: "Passwords",
                      path: withTeamId(paths.CONTROLS_PASSWORDS),
                      keywords: ["rotation", "recovery", "macos"],
                    },
                  ]
                : []),
            ],
          },
          // Setup experience sub-pages
          {
            id: "controls-setup-experience",
            label: "Setup experience",
            group: "Controls",
            path: withTeamId(paths.CONTROLS_SETUP_EXPERIENCE),
            keywords: ["customize", "end user", "enrollment"],
            subItems: [
              {
                id: "controls-users",
                label: "Users",
                path: withTeamId(paths.CONTROLS_USERS),
                keywords: ["idp", "login", "sso"],
              },
              {
                id: "controls-bootstrap-package",
                label: "Bootstrap package",
                path: withTeamId(paths.CONTROLS_BOOTSTRAP_PACKAGE),
                keywords: ["pkg", "deploy"],
              },
              {
                id: "controls-install-software",
                label: "Install software",
                path: withTeamId(paths.CONTROLS_INSTALL_SOFTWARE("macos")),
                keywords: ["automatic install"],
              },
              {
                id: "controls-run-script",
                label: "Run script",
                path: withTeamId(paths.CONTROLS_RUN_SCRIPT),
                keywords: ["shell", "post-enrollment"],
              },
              {
                id: "controls-setup-assistant",
                label: "Setup Assistant",
                path: withTeamId(paths.CONTROLS_SETUP_ASSISTANT),
                keywords: ["apple", "dep", "ade"],
              },
            ],
          },
          // Scripts
          {
            id: "controls-scripts",
            label: "Scripts",
            group: "Controls",
            path: withTeamId(paths.CONTROLS_SCRIPTS),
            keywords: ["remediate", "macos", "windows", "linux"],
            subItems: [
              {
                id: "controls-scripts-library",
                label: "Script library",
                path: withTeamId(paths.CONTROLS_SCRIPTS_LIBRARY),
                keywords: ["saved", "uploaded", "manage"],
              },
              {
                id: "controls-scripts-batch-progress",
                label: "Script batch progress",
                path: withTeamId(paths.CONTROLS_SCRIPTS_BATCH_PROGRESS),
                keywords: ["status", "running", "results"],
              },
            ],
          },
          // Variables
          {
            id: "controls-variables",
            label: "Variables",
            group: "Controls",
            path: withTeamId(paths.CONTROLS_VARIABLES),
            keywords: ["custom", "scripts", "profiles"],
          },
        ]
      : []),

    // Software
    {
      id: "software",
      label: "Software inventory",
      group: "Software",
      path: withTeamId(paths.SOFTWARE_INVENTORY),
      keywords: ["installed", "inventory", "software titles", "detected"],
      subItems: [
        {
          id: "software-versions",
          label: "Software versions",
          path: withTeamId(paths.SOFTWARE_VERSIONS),
          keywords: ["versions", "installed"],
        },
        {
          id: "software-vulnerable",
          label: "Vulnerable software",
          path: withTeamId(`${paths.SOFTWARE_INVENTORY}?vulnerable=true`),
          keywords: ["cve", "exploited", "security"],
        },
      ],
    },
    {
      id: "software-os",
      label: "Operating systems",
      group: "Software",
      path: withTeamId(paths.SOFTWARE_OS),
      keywords: [
        "os versions",
        "macos",
        "windows",
        "linux",
        "ios",
        "ipados",
        "android",
        "chrome",
      ],
    },
    {
      id: "software-vulnerabilities",
      label: "Vulnerabilities",
      group: "Software",
      path: withTeamId(paths.SOFTWARE_VULNERABILITIES),
      keywords: ["cve", "cvss", "exploit", "vulnerable software"],
    },
    // Library is available for any team including unassigned, but not "All fleets"
    ...(isPremiumTier && hasTeamOrUnassigned
      ? [
          {
            id: "software-library",
            label: "Software library",
            group: "Software",
            path: withTeamId(paths.SOFTWARE_LIBRARY),
            keywords: [
              "managed",
              "installable",
              "packages",
              "self-service",
              "library",
            ],
            teamName,
          },
        ]
      : []),

    // Settings — global admins only
    ...(canAccessSettings
      ? [
          // Organization settings
          {
            id: "settings-organization",
            label: "Organization settings",
            group: "Settings",
            path: paths.ADMIN_ORGANIZATION,
            keywords: ["admin", "organization"],
            subItems: [
              {
                id: "settings-org-info",
                label: "Organization info",
                path: paths.ADMIN_ORGANIZATION_INFO,
                keywords: ["name", "logo", "branding", "support url"],
              },
              {
                id: "settings-org-webaddress",
                label: "Fleet web address",
                path: paths.ADMIN_ORGANIZATION_WEBADDRESS,
                keywords: ["url", "server address"],
              },
              {
                id: "settings-org-smtp",
                label: "SMTP options",
                path: paths.ADMIN_ORGANIZATION_SMTP,
                keywords: ["email", "sender", "password reset"],
              },
              {
                id: "settings-org-agents",
                label: "Agent options",
                path: paths.ADMIN_ORGANIZATION_AGENTS,
                keywords: [
                  "osquery",
                  "fleetd",
                  "orbit",
                  "flags",
                  "global config",
                  "command line flags",
                ],
              },
              {
                id: "settings-org-statistics",
                label: "Usage statistics",
                path: paths.ADMIN_ORGANIZATION_STATISTICS,
                keywords: ["telemetry", "anonymous"],
              },
              {
                id: "settings-org-fleet-desktop",
                label: "Fleet Desktop",
                path: paths.ADMIN_ORGANIZATION_FLEET_DESKTOP,
                keywords: [
                  "tray icon",
                  "transparency",
                  "end user",
                  "browser host",
                  "custom proxy",
                ],
              },
              {
                id: "settings-org-advanced",
                label: "Advanced options",
                path: paths.ADMIN_ORGANIZATION_ADVANCED,
                keywords: [
                  "live report",
                  "host expiry",
                  "usage statistics",
                  "sso user url",
                  "sso",
                  "apple mdm server url",
                  "verify ssl certs",
                  "starttls",
                  "host expiry",
                  "generative ai features",
                  "hardware attestation",
                ],
              },
            ],
          },

          // Integrations
          {
            id: "settings-integrations",
            label: "Integrations",
            group: "Settings",
            path: paths.ADMIN_INTEGRATIONS,
            keywords: ["mdm", "jira", "zendesk", "sso", "calendar"],
            subItems: [
              {
                id: "settings-int-ticket-destinations",
                label: "Ticket destinations",
                path: paths.ADMIN_INTEGRATIONS_TICKET_DESTINATIONS,
                keywords: ["jira", "zendesk", "tickets"],
              },
              {
                id: "settings-int-mdm",
                label: "MDM",
                path: paths.ADMIN_INTEGRATIONS_MDM,
                keywords: [
                  "apple",
                  "windows",
                  "android",
                  "device management",
                  "apple business",
                  "vpp",
                  "entra",
                ],
              },
              {
                id: "settings-int-calendars",
                label: "Calendars",
                path: paths.ADMIN_INTEGRATIONS_CALENDARS,
                keywords: [
                  "google calendar api",
                  "google workspace",
                  "service account",
                  "events",
                ],
              },
              {
                id: "settings-int-change-management",
                label: "Change management",
                path: paths.ADMIN_INTEGRATIONS_CHANGE_MANAGEMENT,
                keywords: ["workflow", "gitops mode"],
              },
              {
                id: "settings-int-sso-fleet-users",
                label: "Single sign-on (SSO) for Fleet users",
                path: paths.ADMIN_INTEGRATIONS_SSO_FLEET_USERS,
                keywords: ["saml", "idp", "admin", "login"],
              },
              {
                id: "settings-int-sso-end-users",
                label: "Single sign-on (SSO) for end users",
                path: paths.ADMIN_INTEGRATIONS_SSO_END_USERS,
                keywords: ["saml", "idp", "device user", "login"],
              },
              {
                id: "settings-int-certificate-authorities",
                label: "Certificate authorities",
                path: paths.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES,
                keywords: [
                  "scep",
                  "est",
                  "digicert",
                  "ndes",
                  "smallstep",
                  "scep",
                ],
              },
              {
                id: "add-certificate-authority",
                label: "Add certificate authority",
                path: paths.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES,
                keywords: [
                  "scep",
                  "est",
                  "digicert",
                  "ndes",
                  "smallstep",
                  "pki",
                ],
              },
              {
                id: "settings-int-identity-provider",
                label: "Identity provider (IdP)",
                path: paths.ADMIN_INTEGRATIONS_IDENTITY_PROVIDER,
                keywords: ["okta", "entra", "azure ad"],
              },
              {
                id: "settings-int-host-status-webhook",
                label: "Host status webhook",
                path: paths.ADMIN_INTEGRATIONS_HOST_STATUS_WEBHOOK,
                keywords: ["offline", "missing hosts", "notification"],
              },
              {
                id: "settings-int-conditional-access",
                label: "Conditional access",
                path: paths.ADMIN_INTEGRATIONS_CONDITIONAL_ACCESS,
                keywords: ["okta", "entra", "intune", "zero trust"],
              },
            ],
          },

          // Users and Fleets
          {
            id: "settings-users",
            label: "Users",
            group: "Settings",
            path: paths.ADMIN_USERS,
            keywords: ["accounts", "admins", "invite"],
          },
          {
            id: "settings-fleets",
            label: "Fleets",
            group: "Settings",
            path: paths.ADMIN_FLEETS,
            keywords: [
              "teams",
              "groups",
              "add fleet",
              "create fleet",
              "edit fleet",
              "delete fleet",
            ],
          },
        ]
      : []),

    // Actions — users who can write
    ...(canWrite
      ? [
          {
            id: "add-hosts",
            label: "Add hosts",
            group: "Actions",
            path: withTeamId(`${paths.MANAGE_HOSTS}?add_hosts=1`),
            keywords: ["enroll", "install", "fleetd", "device"],
            teamName: teamOrUnassigned,
          },
          {
            id: "add-report",
            label: "Add report",
            group: "Actions",
            path: withTeamId(paths.NEW_REPORT),
            keywords: ["create report", "new report", "sql"],
            teamName,
          },
          {
            id: "add-policy",
            label: "Add policy",
            group: "Actions",
            path: withTeamId(paths.NEW_POLICY),
            keywords: [
              "create policy",
              "new policy",
              "compliance",
              "device health",
            ],
            teamName,
          },
          // Software add actions require a team or unassigned (not "All fleets")
          ...(hasTeamOrUnassigned
            ? [
                {
                  id: "add-fleet-maintained-app",
                  label: "Add Fleet-maintained app",
                  group: "Actions",
                  path: withTeamId(paths.SOFTWARE_ADD_FLEET_MAINTAINED),
                  keywords: [
                    "install",
                    "software",
                    "managed app",
                    "fma",
                    "add app",
                  ],
                  teamName,
                },
                {
                  id: "add-vpp-app",
                  label: "Add VPP app",
                  group: "Actions",
                  path: withTeamId(
                    `${paths.SOFTWARE_ADD_APP_STORE}?platform=apple`
                  ),
                  keywords: [
                    "app store",
                    "volume purchase",
                    "apple",
                    "ios",
                    "ipados",
                    "macos",
                    "add app",
                  ],
                  teamName,
                },
                {
                  id: "add-android-app-store-app",
                  label: "Add Android app store app",
                  group: "Actions",
                  path: withTeamId(
                    `${paths.SOFTWARE_ADD_APP_STORE}?platform=android`
                  ),
                  keywords: ["google play", "android", "play store", "add app"],
                  teamName,
                },
                {
                  id: "add-custom-package",
                  label: "Add custom package",
                  group: "Actions",
                  path: withTeamId(paths.SOFTWARE_ADD_PACKAGE),
                  keywords: [
                    "install",
                    "upload",
                    "software",
                    "add package",
                    "pkg",
                    "ipa",
                    "msi",
                    "exe",
                    "ps1",
                    "deb",
                    "rpm",
                    "tar.gz",
                    "tarballs",
                    "sh",
                  ],
                  teamName,
                },
              ]
            : []),
          // Script and variable actions require a team or unassigned (not "All fleets")
          ...(hasTeamOrUnassigned
            ? [
                {
                  id: "add-script",
                  label: "Add script",
                  group: "Actions",
                  path: withTeamId(paths.CONTROLS_SCRIPTS_LIBRARY),
                  keywords: [
                    "upload script",
                    "shell",
                    "sh",
                    "ps1",
                    "create script",
                  ],
                  teamName: teamOrUnassigned,
                },
                {
                  id: "add-custom-variable",
                  label: "Add custom variable",
                  group: "Actions",
                  path: withTeamId(
                    `${paths.CONTROLS_VARIABLES}?add_variable=1`
                  ),
                  keywords: [
                    "secret",
                    "scripts",
                    "profiles",
                    "add variable",
                    "create variable",
                  ],
                  teamName: teamOrUnassigned,
                },
              ]
            : []),
          {
            id: "manage-enroll-secrets",
            label: "Manage enroll secrets",
            group: "Actions",
            path: withTeamId(`${paths.MANAGE_HOSTS}?manage_enroll_secrets=1`),
            keywords: ["enrollment", "token", "fleetd", "enroll secret"],
            teamName: teamOrUnassigned,
          },
          {
            id: "run-live-report",
            label: "Run live report",
            group: "Actions",
            path: withTeamId(paths.NEW_REPORT),
            keywords: [
              "osquery",
              "sql",
              "live",
              "ad hoc",
              "query",
              "run report",
            ],
            teamName,
          },
          {
            id: "run-live-policy",
            label: "Run live policy",
            group: "Actions",
            path: withTeamId(paths.NEW_POLICY),
            keywords: ["check", "compliance", "live", "ad hoc", "run policy"],
            teamName,
          },
          {
            id: "add-label",
            label: "Add label",
            group: "Actions",
            path: paths.NEW_LABEL,
            keywords: [
              "create label",
              "new label",
              "group hosts",
              "filter",
              "dynamic",
              "manual",
            ],
          },
          ...(canAccessSettings
            ? [
                {
                  id: "create-fleet",
                  label: "Create fleet",
                  group: "Actions",
                  path: paths.ADMIN_FLEETS,
                  keywords: ["new fleet", "add fleet", "team"],
                },
              ]
            : []),
          {
            id: "toggle-dark-mode",
            label: isDarkMode()
              ? "Switch to light mode"
              : "Switch to dark mode",
            group: "Actions",
            keywords: ["dark mode", "light mode", "theme", "toggle"],
            onAction: onToggleDarkMode,
          },
        ]
      : []),

    // MDM — global admins only
    ...(canAccessSettings
      ? [
          // Apple MDM — turn on or edit
          ...(!isMacMdmEnabledAndConfigured
            ? [
                {
                  id: "turn-on-apple-mdm",
                  label: "Turn on Apple (macOS, iOS, iPadOS) MDM",
                  group: "MDM",
                  path: paths.ADMIN_INTEGRATIONS_MDM_APPLE,
                  keywords: ["enable", "apns", "dep"],
                },
              ]
            : [
                {
                  id: "edit-apple-mdm",
                  label: "Edit Apple (macOS, iOS, iPadOS) MDM",
                  group: "MDM",
                  path: paths.ADMIN_INTEGRATIONS_MDM_APPLE,
                  keywords: ["apns", "certificate", "renew"],
                },
                {
                  id: isAbmConfigured ? "edit-abm" : "add-abm",
                  label: isAbmConfigured
                    ? "Edit Apple Business Manager (ABM)"
                    : "Add Apple Business Manager (ABM)",
                  group: "MDM",
                  path: paths.ADMIN_INTEGRATIONS_APPLE_BUSINESS_MANAGER,
                  keywords: ["dep", "automated enrollment", "apple"],
                },
                {
                  id: isVppEnabled ? "edit-vpp" : "add-vpp",
                  label: isVppEnabled
                    ? "Edit Volume Purchasing Program (VPP)"
                    : "Add Volume Purchasing Program (VPP)",
                  group: "MDM",
                  path: paths.ADMIN_INTEGRATIONS_VPP,
                  keywords: ["app store", "apple", "token"],
                },
              ]),
          // Windows MDM — turn on or edit
          ...(!isWindowsMdmEnabledAndConfigured
            ? [
                {
                  id: "turn-on-windows-mdm",
                  label: "Turn on Windows MDM",
                  group: "MDM",
                  path: paths.ADMIN_INTEGRATIONS_MDM_WINDOWS,
                  keywords: ["enable", "microsoft"],
                },
              ]
            : [
                {
                  id: "edit-windows-mdm",
                  label: "Edit Windows MDM",
                  group: "MDM",
                  path: paths.ADMIN_INTEGRATIONS_MDM_WINDOWS,
                  keywords: ["microsoft", "enrollment"],
                },
                {
                  id: "windows-automatic-enrollment",
                  label: "Windows automatic enrollment (Entra)",
                  group: "MDM",
                  path: paths.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS,
                  keywords: ["entra", "azure ad", "microsoft"],
                },
              ]),
          // Android MDM — turn on or edit
          ...(!isAndroidMdmEnabledAndConfigured
            ? [
                {
                  id: "turn-on-android-mdm",
                  label: "Turn on Android MDM",
                  group: "MDM",
                  path: paths.ADMIN_INTEGRATIONS_MDM_ANDROID,
                  keywords: ["enable", "google", "enterprise"],
                },
              ]
            : [
                {
                  id: "edit-android-mdm",
                  label: "Edit Android MDM",
                  group: "MDM",
                  path: paths.ADMIN_INTEGRATIONS_MDM_ANDROID,
                  keywords: ["google", "enterprise"],
                },
              ]),
        ]
      : []),

    // Manage automations — software (global admin, all fleets only).
    // Hardcoded "All fleets" because software automations are global-only
    // and don't use the team-scoped teamName.
    ...(canManageSoftwareAutomations
      ? [
          {
            id: "manage-software-automations",
            label: "Manage software automations",
            group: "Automations",
            path: `${paths.SOFTWARE_INVENTORY}?manage_automations=1`,
            keywords: ["vulnerability", "webhook", "jira", "zendesk"],
            teamName: "All fleets",
          },
        ]
      : []),

    // Manage automations — activity feed (global admin only)
    ...(canAccessSettings
      ? [
          {
            id: "manage-activity-automations",
            label: "Manage activity automations",
            group: "Automations",
            path: `${paths.DASHBOARD}?manage_automations=1`,
            keywords: ["activity feed", "webhook", "audit log"],
          },
        ]
      : []),

    // Manage automations — reports (anyone who can write)
    ...(canWrite
      ? [
          {
            id: "manage-report-automations",
            label: "Manage report automations",
            group: "Automations",
            path: withTeamId(`${paths.MANAGE_REPORTS}?manage_automations=1`),
            keywords: ["report", "logging", "destination"],
            teamName,
          },
        ]
      : []),

    // Manage automations — policies (admins and maintainers)
    ...(canManagePolicyAutomations
      ? [
          {
            id: "manage-policy-automations",
            label: "Manage policy automations",
            group: "Automations",
            path: withTeamId(paths.MANAGE_POLICIES),
            keywords: ["failing", "webhook", "jira", "zendesk"],
            teamName,
            subItems: [
              {
                id: "manage-policy-automations-webhooks",
                label: "Tickets & webhooks",
                path: withTeamId(
                  `${paths.MANAGE_POLICIES}?manage_automations=webhooks`
                ),
                keywords: ["jira", "zendesk", "failing"],
              },
              // Team-scoped policy automations (premium, require a specific fleet selected)
              ...(isPremiumTier && hasTeamSelected
                ? [
                    {
                      id: "manage-policy-automations-install-software",
                      label: "Install software",
                      path: `${paths.MANAGE_POLICIES}?fleet_id=${currentTeam?.id}&manage_automations=install_software`,
                      keywords: ["resolve", "remediate"],
                    },
                    {
                      id: "manage-policy-automations-run-script",
                      label: "Run script",
                      path: `${paths.MANAGE_POLICIES}?fleet_id=${currentTeam?.id}&manage_automations=run_script`,
                      keywords: ["resolve", "remediate"],
                    },
                    {
                      id: "manage-policy-automations-calendar",
                      label: "Calendar events",
                      path: `${paths.MANAGE_POLICIES}?fleet_id=${currentTeam?.id}&manage_automations=calendar`,
                      keywords: [
                        "reserve time",
                        "maintenance window",
                        "google calendar",
                      ],
                    },
                    {
                      id: "manage-policy-automations-conditional-access",
                      label: "Conditional access",
                      path: `${paths.MANAGE_POLICIES}?fleet_id=${currentTeam?.id}&manage_automations=conditional_access`,
                      keywords: [
                        "sso",
                        "okta",
                        "entra",
                        "intune",
                        "zero trust",
                      ],
                    },
                  ]
                : []),
            ],
          },
        ]
      : []),
  ];
};
