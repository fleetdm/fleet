import React, {
  useContext,
  useEffect,
  useState,
  useCallback,
  useRef,
} from "react";
import { Command } from "cmdk";
import { browserHistory } from "react-router";

import { AppContext } from "context/app";
import { APP_CONTEXT_ALL_TEAMS_ID } from "interfaces/team";
import Icon from "components/Icon";
import { isDarkMode, toggleDarkMode } from "utilities/theme";
import paths from "router/paths";

const baseClass = "command-palette";

type Page = "root" | "switch-fleet";

interface ICommandSubItem {
  id: string;
  label: string;
  path: string;
  keywords?: string[];
}

interface ICommandItem {
  id: string;
  label: string;
  group: string;
  path?: string;
  keywords?: string[];
  /** When set, displays the team/fleet name right-aligned in the item row */
  teamName?: string;
  /** Nested items shown when the parent is expanded via "more" */
  subItems?: ICommandSubItem[];
  /** Custom action instead of navigation */
  onAction?: () => void;
}

const CommandPalette = (): JSX.Element | null => {
  const [open, setOpen] = useState(false);
  const [page, setPage] = useState<Page>("root");
  const [search, setSearch] = useState("");
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());
  const inputRef = useRef<HTMLInputElement>(null);

  const {
    availableTeams,
    currentTeam,
    setCurrentTeam,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamAdmin,
    isAnyTeamMaintainer,
    isGlobalTechnician,
    isAnyTeamTechnician,
    isPremiumTier,
    isMacMdmEnabledAndConfigured,
    isWindowsMdmEnabledAndConfigured,
    isAndroidMdmEnabledAndConfigured,
    isNoAccess,
  } = useContext(AppContext);

  const isTechnician = isGlobalTechnician || isAnyTeamTechnician;

  const canAccessControls =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isAnyTeamAdmin ||
    isAnyTeamMaintainer ||
    isTechnician;

  const canWrite =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isAnyTeamAdmin ||
    isAnyTeamMaintainer ||
    isTechnician;

  // Policy automations: same as canAddOrDeletePolicies in ManagePoliciesPage
  const canManagePolicyAutomations =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isAnyTeamAdmin ||
    isAnyTeamMaintainer;

  // Software automations require global admin (all fleets view)
  const canManageSoftwareAutomations = isGlobalAdmin;

  const canAccessSettings = isGlobalAdmin;

  // Whether a specific team is selected (not "All teams")
  const hasTeamSelected = currentTeam && currentTeam.id > 0;
  const teamName = hasTeamSelected ? currentTeam?.name : undefined;

  // Append fleet_id to a path so navigation preserves the current team context
  const withTeamId = useCallback(
    (path: string) => {
      if (!hasTeamSelected) {
        return path;
      }
      const separator = path.includes("?") ? "&" : "?";
      return `${path}${separator}fleet_id=${currentTeam?.id}`;
    },
    [hasTeamSelected, currentTeam?.id]
  );

  // Reset page and search when dialog opens/closes
  useEffect(() => {
    if (!open) {
      setPage("root");
      setSearch("");
      setExpandedItems(new Set());
    }
  }, [open]);

  // Toggle open on Cmd+K / Ctrl+K
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((prev) => !prev);
      }
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, []);

  const navigate = useCallback((path: string) => {
    setOpen(false);
    browserHistory.push(path);
  }, []);

  const goToPage = useCallback((newPage: Page) => {
    setSearch("");
    setPage(newPage);
  }, []);

  const goBack = useCallback(() => {
    setSearch("");
    setPage("root");
  }, []);

  // Backspace on empty input returns to root page
  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (page !== "root" && e.key === "Backspace" && !search) {
        e.preventDefault();
        goBack();
      }
    },
    [page, search, goBack]
  );

  const handleSwitchFleet = useCallback(
    (fleetId: number) => {
      const selected = availableTeams?.find((t) => t.id === fleetId);
      if (selected) {
        setCurrentTeam(selected);
      }
      setOpen(false);

      // Update the current URL's fleet_id param to reflect the switch
      const { pathname, search: currentSearch } = window.location;
      const params = new URLSearchParams(currentSearch);
      if (fleetId === APP_CONTEXT_ALL_TEAMS_ID) {
        params.delete("fleet_id");
      } else {
        params.set("fleet_id", String(fleetId));
      }
      const qs = params.toString();
      browserHistory.push(qs ? `${pathname}?${qs}` : pathname);
    },
    [availableTeams, setCurrentTeam]
  );

  if (isNoAccess) {
    return null;
  }

  const items: ICommandItem[] = [
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
      path: withTeamId(paths.SOFTWARE_TITLES),
      keywords: ["installed", "inventory", "titles"],
    },
    {
      id: "reports",
      label: "Reports",
      group: "Pages",
      path: withTeamId(paths.MANAGE_REPORTS),
      keywords: ["report", "sql", "gather data"],
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
    {
      id: "users-page",
      label: "Users",
      group: "Pages",
      path: paths.ADMIN_USERS,
      keywords: ["accounts", "admins", "invite"],
    },
    {
      id: "my-account",
      label: "My account",
      group: "Pages",
      path: paths.ACCOUNT,
      keywords: ["profile", "password", "api token"],
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
                keywords: ["custom profiles", "mobileconfig", "deploy"],
              },
              // Certificates and Passwords — not available to technicians
              ...(!isTechnician
                ? [
                    {
                      id: "controls-certificates",
                      label: "Certificates",
                      path: withTeamId(paths.CONTROLS_CERTIFICATES),
                      keywords: ["scep", "est", "pki", "digicert", "ndes"],
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
                id: "controls-end-user-auth",
                label: "End user authentication",
                path: withTeamId(paths.CONTROLS_END_USER_AUTHENTICATION),
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
      label: "Software",
      group: "Software",
      path: withTeamId(paths.SOFTWARE_TITLES),
      keywords: ["installed", "inventory", "titles"],
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
          path: withTeamId(`${paths.SOFTWARE_TITLES}?vulnerable=true`),
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
                keywords: ["name", "logo", "branding"],
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
                keywords: ["osquery", "fleetd", "orbit", "flags"],
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
                keywords: ["tray icon", "transparency", "end user"],
              },
              {
                id: "settings-org-advanced",
                label: "Advanced options",
                path: paths.ADMIN_ORGANIZATION_ADVANCED,
                keywords: ["live report", "host expiry", "usage statistics"],
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
                keywords: ["apple", "windows", "android", "device management"],
              },
              {
                id: "settings-int-calendars",
                label: "Calendars",
                path: paths.ADMIN_INTEGRATIONS_CALENDARS,
                keywords: ["google calendar", "service account", "events"],
              },
              {
                id: "settings-int-change-management",
                label: "Change management",
                path: paths.ADMIN_INTEGRATIONS_CHANGE_MANAGEMENT,
                keywords: ["approval", "workflow", "gitops mode"],
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
                keywords: ["scep", "est", "digicert", "ndes", "smallstep"],
              },
              {
                id: "add-certificate-authority",
                label: "Add certificate authority",
                path: paths.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES,
                keywords: ["scep", "est", "digicert", "ndes", "smallstep", "pki"],
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
              // Apple MDM — turn on or edit
              ...(!isMacMdmEnabledAndConfigured
                ? [
                    {
                      id: "turn-on-apple-mdm",
                      label: "Turn on Apple (macOS, iOS, iPadOS) MDM",
                      path: paths.ADMIN_INTEGRATIONS_MDM_APPLE,
                      keywords: ["enable", "apns", "dep"],
                    },
                  ]
                : [
                    {
                      id: "edit-apple-mdm",
                      label: "Edit Apple (macOS, iOS, iPadOS) MDM",
                      path: paths.ADMIN_INTEGRATIONS_MDM_APPLE,
                      keywords: ["apns", "certificate", "renew"],
                    },
                    {
                      id: "add-abm",
                      label: "Add Apple Business Manager (ABM)",
                      path: paths.ADMIN_INTEGRATIONS_APPLE_BUSINESS_MANAGER,
                      keywords: ["dep", "automated enrollment", "apple"],
                    },
                    {
                      id: "add-vpp",
                      label: "Add Volume Purchasing Program (VPP)",
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
                      path: paths.ADMIN_INTEGRATIONS_MDM_WINDOWS,
                      keywords: ["enable", "microsoft"],
                    },
                  ]
                : [
                    {
                      id: "edit-windows-mdm",
                      label: "Edit Windows MDM",
                      path: paths.ADMIN_INTEGRATIONS_MDM_WINDOWS,
                      keywords: ["microsoft", "enrollment"],
                    },
                    {
                      id: "windows-automatic-enrollment",
                      label: "Windows automatic enrollment (Entra)",
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
                      path: paths.ADMIN_INTEGRATIONS_MDM_ANDROID,
                      keywords: ["enable", "google", "enterprise"],
                    },
                  ]
                : [
                    {
                      id: "edit-android-mdm",
                      label: "Edit Android MDM",
                      path: paths.ADMIN_INTEGRATIONS_MDM_ANDROID,
                      keywords: ["google", "enterprise"],
                    },
                  ]),
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
            keywords: ["teams", "groups"],
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
            teamName,
          },
          {
            id: "add-label",
            label: "Add label",
            group: "Actions",
            path: paths.NEW_LABEL,
            keywords: ["create", "group hosts", "filter", "dynamic", "manual"],
          },
          {
            id: "new-report",
            label: "New report",
            group: "Actions",
            path: withTeamId(paths.NEW_REPORT),
            keywords: ["create", "report", "sql"],
            teamName,
          },
          {
            id: "new-policy",
            label: "New policy",
            group: "Actions",
            path: withTeamId(paths.NEW_POLICY),
            keywords: ["create", "compliance", "device health"],
            teamName,
          },
          {
            id: "add-fleet-maintained-app",
            label: "Add Fleet-maintained app",
            group: "Actions",
            path: withTeamId(paths.SOFTWARE_ADD_FLEET_MAINTAINED),
            keywords: ["install", "software", "managed app"],
            teamName,
          },
          {
            id: "add-app-store-app",
            label: "Add App Store app",
            group: "Actions",
            path: withTeamId(paths.SOFTWARE_ADD_APP_STORE),
            keywords: ["install", "software", "vpp", "apple", "ios", "ipados"],
            teamName,
          },
          {
            id: "add-custom-package",
            label: "Add custom package",
            group: "Actions",
            path: withTeamId(paths.SOFTWARE_ADD_PACKAGE),
            keywords: ["upload", "software", "pkg", "msi", "deb", "exe"],
            teamName,
          },
          {
            id: "add-script",
            label: "Add script",
            group: "Actions",
            path: withTeamId(paths.CONTROLS_SCRIPTS_LIBRARY),
            keywords: ["upload", "shell", "remediate"],
            teamName,
          },
          {
            id: "add-custom-variable",
            label: "Add custom variable",
            group: "Actions",
            path: withTeamId(paths.CONTROLS_VARIABLES),
            keywords: ["secret", "scripts", "profiles"],
            teamName,
          },
          {
            id: "manage-enroll-secrets",
            label: "Manage enroll secrets",
            group: "Actions",
            path: withTeamId(paths.MANAGE_HOSTS),
            keywords: ["enrollment", "token", "fleetd"],
            teamName,
          },
          {
            id: "toggle-dark-mode",
            label: isDarkMode() ? "Switch to light mode" : "Switch to dark mode",
            group: "Actions",
            keywords: ["dark mode", "light mode", "theme", "toggle"],
            onAction: () => {
              toggleDarkMode();
              setOpen(false);
            },
          },
          {
            id: "sign-out",
            label: "Sign out",
            group: "Actions",
            path: paths.LOGOUT,
            keywords: ["logout", "log out", "sign out"],
          },
        ]
      : []),

    // Manage automations — software (global admin, all fleets only)
    ...(canManageSoftwareAutomations
      ? [
          {
            id: "manage-software-automations",
            label: "Manage software automations",
            group: "Automations",
            path: `${paths.SOFTWARE_TITLES}?manage_automations=1`,
            keywords: ["vulnerability", "webhook", "jira", "zendesk"],
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
                path: withTeamId(paths.MANAGE_POLICIES),
                keywords: ["jira", "zendesk", "failing"],
              },
              // Team-scoped policy automations (premium, require a specific fleet selected)
              ...(isPremiumTier && hasTeamSelected
                ? [
                    {
                      id: "manage-policy-automations-install-software",
                      label: "Install software",
                      path: `${paths.MANAGE_POLICIES}?fleet_id=${currentTeam?.id}`,
                      keywords: ["resolve", "remediate"],
                    },
                    {
                      id: "manage-policy-automations-run-script",
                      label: "Run script",
                      path: `${paths.MANAGE_POLICIES}?fleet_id=${currentTeam?.id}`,
                      keywords: ["resolve", "remediate"],
                    },
                    {
                      id: "manage-policy-automations-calendar",
                      label: "Calendar events",
                      path: `${paths.MANAGE_POLICIES}?fleet_id=${currentTeam?.id}`,
                      keywords: ["reserve time", "maintenance window", "google calendar"],
                    },
                    {
                      id: "manage-policy-automations-conditional-access",
                      label: "Conditional access",
                      path: `${paths.MANAGE_POLICIES}?fleet_id=${currentTeam?.id}`,
                      keywords: ["sso", "okta", "entra", "intune", "zero trust"],
                    },
                  ]
                : []),
            ],
          },
        ]
      : []),
  ];

  // Build groups in display order
  const groups = [
    "Pages",
    "Controls",
    "Software",
    "Settings",
    "Actions",
    "Automations",
  ];
  const groupedItems: Record<string, ICommandItem[]> = {};
  for (const item of items) {
    if (!groupedItems[item.group]) {
      groupedItems[item.group] = [];
    }
    groupedItems[item.group].push(item);
  }

  const toggleExpanded = useCallback((id: string) => {
    setExpandedItems((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const isSearching = search.length > 0;
  const searchLower = search.toLowerCase().trim();

  // Find exact match — an item or sub-item whose label exactly matches the search
  const exactMatchIds = new Set<string>();
  if (isSearching) {
    for (const item of items) {
      if (item.label.toLowerCase() === searchLower) {
        exactMatchIds.add(item.id);
      }
      if (item.subItems) {
        for (const sub of item.subItems) {
          if (sub.label.toLowerCase() === searchLower) {
            exactMatchIds.add(sub.id);
          }
        }
      }
    }
  }

  const getItemValue = (item: ICommandItem) => {
    const parts = [item.label, ...(item.keywords ?? [])];
    if (item.subItems) {
      for (const sub of item.subItems) {
        parts.push(sub.label, ...(sub.keywords ?? []));
      }
    }
    return parts.join(" ");
  };

  const renderItem = (item: ICommandItem) => {
    const isExpanded = expandedItems.has(item.id);
    const hasSubItems = item.subItems && item.subItems.length > 0;

    return (
      <React.Fragment key={item.id}>
        <Command.Item
          value={getItemValue(item)}
          onSelect={() => (item.onAction ? item.onAction() : navigate(item.path!))}
          className={`${baseClass}__item`}
        >
          <div className={`${baseClass}__item-left`}>
            <span className={`${baseClass}__item-label`}>{item.label}</span>
            {hasSubItems && !isSearching && (
              <button
                type="button"
                className={`${baseClass}__item-more`}
                onClick={(e) => {
                  e.stopPropagation();
                  toggleExpanded(item.id);
                }}
                onPointerDown={(e) => e.preventDefault()}
              >
                {isExpanded ? "less" : "more"}
              </button>
            )}
          </div>
          {item.teamName && (
            <span className={`${baseClass}__item-team`}>
              {item.teamName}
            </span>
          )}
        </Command.Item>
        {/* Render sub-items when expanded (browsing) or always when searching */}
        {hasSubItems &&
          (isExpanded || isSearching) &&
          item.subItems!.map((sub) => (
            <Command.Item
              key={sub.id}
              value={`${sub.label} ${sub.keywords?.join(" ") ?? ""}`}
              onSelect={() => navigate(sub.path)}
              className={`${baseClass}__item ${baseClass}__item--sub`}
            >
              <span className={`${baseClass}__item-label`}>{sub.label}</span>
            </Command.Item>
          ))}
      </React.Fragment>
    );
  };

  // Collect exact match items for the "Best match" section
  const exactMatchItems: Array<{ item: ICommandItem; sub?: ICommandSubItem }> = [];
  if (exactMatchIds.size > 0) {
    for (const item of items) {
      if (exactMatchIds.has(item.id)) {
        exactMatchItems.push({ item });
      }
      if (item.subItems) {
        for (const sub of item.subItems) {
          if (exactMatchIds.has(sub.id)) {
            exactMatchItems.push({ item, sub });
          }
        }
      }
    }
  }

  const renderRootPage = () => (
    <>
      {/* Exact match at the top with a separator */}
      {exactMatchItems.length > 0 && (
        <>
          <Command.Group
            heading="Best match"
            className={`${baseClass}__group`}
          >
            {exactMatchItems.map(({ item, sub }) => {
              const target = sub || item;
              return (
                <Command.Item
                  key={`exact-${target.id}`}
                  value={`EXACT_MATCH ${target.label}`}
                  onSelect={() =>
                    item.onAction ? item.onAction() : navigate(target.path!)
                  }
                  className={`${baseClass}__item`}
                >
                  <span className={`${baseClass}__item-label`}>
                    {target.label}
                  </span>
                  {"teamName" in item && item.teamName && (
                    <span className={`${baseClass}__item-team`}>
                      {item.teamName}
                    </span>
                  )}
                </Command.Item>
              );
            })}
          </Command.Group>
          <Command.Separator className={`${baseClass}__separator`} />
        </>
      )}
      {groups.map((group) => {
        const groupItems = groupedItems[group];
        if (!groupItems?.length) {
          return null;
        }
        return (
          <Command.Group
            key={group}
            heading={group}
            className={`${baseClass}__group`}
          >
            {groupItems.map(renderItem)}
          </Command.Group>
        );
      })}
      {/* Switch fleet action — always at the bottom of root */}
      {isPremiumTier && availableTeams && availableTeams.length > 1 && (
        <Command.Group heading="Navigate" className={`${baseClass}__group`}>
          <Command.Item
            value="Switch fleet team change"
            onSelect={() => goToPage("switch-fleet")}
            className={`${baseClass}__item`}
          >
            <span className={`${baseClass}__item-label`}>Switch fleet...</span>
            {currentTeam?.name && (
              <span className={`${baseClass}__item-team`}>
                {currentTeam.name}
              </span>
            )}
          </Command.Item>
        </Command.Group>
      )}
    </>
  );

  const renderSwitchFleetPage = () => (
    <Command.Group heading="Switch fleet" className={`${baseClass}__group`}>
      {availableTeams?.map((fleet) => {
        const isActive = currentTeam?.id === fleet.id;
        return (
          <Command.Item
            key={`fleet-${fleet.id}`}
            value={fleet.name}
            onSelect={() => handleSwitchFleet(fleet.id)}
            className={`${baseClass}__item`}
          >
            <span className={`${baseClass}__item-label`}>{fleet.name}</span>
            {isActive && (
              <span className={`${baseClass}__item-team`}>current</span>
            )}
          </Command.Item>
        );
      })}
    </Command.Group>
  );

  return (
    <Command.Dialog
      open={open}
      onOpenChange={setOpen}
      label="Command palette"
      className={baseClass}
      overlayClassName={`${baseClass}__overlay`}
      contentClassName={`${baseClass}__content`}
      filter={(value, searchTerm) => {
        // Always show exact match items at the top
        if (value.startsWith("EXACT_MATCH ")) {
          return 1;
        }
        // Default cmdk filtering
        if (value.toLowerCase().includes(searchTerm.toLowerCase())) {
          return 1;
        }
        return 0;
      }}
    >
      {page !== "root" && (
        <div className={`${baseClass}__breadcrumb`}>
          <button
            type="button"
            className={`${baseClass}__breadcrumb-back`}
            onClick={goBack}
          >
            Switch fleet
          </button>
        </div>
      )}
      <div className={`${baseClass}__input-wrapper`}>
        <Icon
          name="search"
          color="ui-fleet-black-50"
          size="small"
          className={`${baseClass}__input-icon`}
        />
        <Command.Input
          ref={inputRef}
          className={`${baseClass}__input`}
          placeholder={
            page === "switch-fleet"
              ? "Search fleets..."
              : "Search or jump to..."
          }
          value={search}
          onValueChange={setSearch}
          onKeyDown={onKeyDown}
        />
      </div>
      <Command.List className={`${baseClass}__list`}>
        <Command.Empty className={`${baseClass}__empty`}>
          No results found.
        </Command.Empty>
        {page === "root" && renderRootPage()}
        {page === "switch-fleet" && renderSwitchFleetPage()}
      </Command.List>
    </Command.Dialog>
  );
};

export default CommandPalette;
