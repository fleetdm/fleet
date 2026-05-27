import paths from "router/paths";

import { ICommandItem, ICommandPaletteContext } from "../helpers";
import { IDerivedContext } from "./derivations";

const buildCommandsItems = (
  ctx: ICommandPaletteContext,
  derived: IDerivedContext
): ICommandItem[] => {
  const {
    search,
    canAccessSettings,
    canRunLiveReport,
    canWrite,
    isPremiumTier,
    isPrimoMode,
    isDarkMode,
    withTeamId,
    onToggleDarkMode,
    onViewHost,
    onViewSoftware,
    onViewSoftwareLibrary,
    onViewReport,
    onViewPolicy,
  } = ctx;
  const {
    hasTeamOrUnassigned,
    isGitOpsMode,
    switchesFromUnassigned,
    teamRequiredDestination,
    defaultDestination,
  } = derived;

  return [
    // Create new pack — companion to the "Packs" page in pages.ts. Shares
    // the same search-regex condition. Kept here so the Commands group
    // stays self-contained.
    ...(/packs|create new pack|add new pack/.test(search.toLowerCase())
      ? [
          {
            id: "new-pack",
            label: "Create new pack",
            group: "Commands" as const,
            path: paths.NEW_PACK,
            keywords: ["packs", "add new pack", "create new pack"],
          },
        ]
      : []),

    // View commands — open sub-pages with searchable lists. Placed at
    // the top of the Commands group so view actions appear before write
    // actions like Add hosts within this group.
    {
      id: "view-host",
      label: "View host",
      group: "Commands" as const,
      keywords: [
        "host",
        "device",
        "find host",
        "open host",
        "host details",
        "endpoint",
        "machine",
        "search host",
        "search hosts",
      ],
      onAction: onViewHost,
      opensSubPage: true,
    },
    {
      id: "view-software",
      label: "View software inventory",
      group: "Commands" as const,
      keywords: [
        "software",
        "app",
        "application",
        "package",
        "find software",
        "open software",
        "title",
        "version",
        "search software",
        "search software inventory",
        "inventory",
      ],
      onAction: onViewSoftware,
      opensSubPage: true,
    },
    // View software library — Premium-only and hidden on "All fleets" since
    // libraries are per-fleet.
    ...(isPremiumTier && hasTeamOrUnassigned
      ? [
          {
            id: "view-software-library",
            label: "View software library",
            group: "Commands" as const,
            keywords: [
              "library",
              "installable",
              "install",
              "available",
              "package",
              "vpp",
              "fma",
              "fleet-maintained",
              "search software library",
              "search library",
            ],
            onAction: onViewSoftwareLibrary,
            opensSubPage: true,
          },
        ]
      : []),
    {
      id: "view-report",
      label: "View report",
      group: "Commands" as const,
      keywords: [
        "report",
        "query",
        "queries",
        "sql",
        "saved query",
        "find report",
        "open report",
        "search report",
        "search reports",
      ],
      onAction: onViewReport,
      opensSubPage: true,
    },
    {
      id: "view-policy",
      label: "View policy",
      group: "Commands" as const,
      keywords: [
        "policy",
        "compliance",
        "failing",
        "device health",
        "find policy",
        "open policy",
        "search policy",
        "search policies",
      ],
      onAction: onViewPolicy,
      opensSubPage: true,
    },

    // Actions — users who can write
    ...(canWrite
      ? [
          {
            id: "add-hosts",
            label: "Add hosts",
            group: "Commands" as const,
            path: withTeamId(`${paths.MANAGE_HOSTS}?add_hosts=1`),
            keywords: ["enroll", "install", "fleetd", "device"],
            teamName: teamRequiredDestination,
          },
          {
            id: "add-report",
            label: "Add report",
            group: "Commands" as const,
            path: withTeamId(paths.NEW_REPORT),
            keywords: ["create report", "new report", "sql"],
            teamName: defaultDestination,
          },
          {
            id: "add-policy",
            label: "Add policy",
            group: "Commands" as const,
            path: withTeamId(paths.NEW_POLICY),
            keywords: [
              "create policy",
              "new policy",
              "compliance",
              "device health",
            ],
          },
          // Software add actions require Premium + a team or unassigned
          // (not "All fleets"). Each destination page renders a
          // <PremiumFeatureMessage /> in Free.
          ...(isPremiumTier && hasTeamOrUnassigned
            ? [
                {
                  id: "add-fleet-maintained-app",
                  label: "Add Fleet-maintained app",
                  group: "Commands" as const,
                  path: withTeamId(paths.SOFTWARE_ADD_FLEET_MAINTAINED),
                  keywords: [
                    "install",
                    "software",
                    "managed app",
                    "fma",
                    "add app",
                  ],
                },
                {
                  id: "add-vpp-app",
                  label: "Add VPP app",
                  group: "Commands" as const,
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
                },
                {
                  id: "add-android-app-store-app",
                  label: "Add Android app store app",
                  group: "Commands" as const,
                  path: withTeamId(
                    `${paths.SOFTWARE_ADD_APP_STORE}?platform=android`
                  ),
                  keywords: ["google play", "android", "play store", "add app"],
                },
                {
                  id: "add-custom-package",
                  label: "Add custom package",
                  group: "Commands" as const,
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
                },
              ]
            : []),
          // Script and variable actions require a team or unassigned (not "All fleets")
          ...(hasTeamOrUnassigned
            ? [
                {
                  id: "add-script",
                  label: "Add script",
                  group: "Commands" as const,
                  path: withTeamId(paths.CONTROLS_SCRIPTS_LIBRARY),
                  keywords: [
                    "upload script",
                    "shell",
                    "sh",
                    "ps1",
                    "create script",
                  ],
                },
                // Custom Variables are NOT Premium-gated — the Variables
                // page itself accepts Free-tier users (only the copy
                // varies). Don't add isPremiumTier here.
                {
                  id: "add-custom-variable",
                  label: "Add custom variable",
                  group: "Commands" as const,
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
                },
              ]
            : []),
          {
            id: "manage-enroll-secrets",
            label: "Manage enroll secrets",
            group: "Commands" as const,
            path: withTeamId(`${paths.MANAGE_HOSTS}?manage_enroll_secrets=1`),
            keywords: ["enrollment", "token", "fleetd", "enroll secret"],
            teamName: teamRequiredDestination,
          },
        ]
      : []),

    // Run live report — Observer+ users can also run live queries.
    // Placed here so it sits adjacent to Run live policy below.
    ...(canRunLiveReport
      ? [
          {
            id: "run-live-report",
            label: "Run live report",
            group: "Commands" as const,
            path: withTeamId(paths.NEW_REPORT),
            keywords: [
              "osquery",
              "sql",
              "live",
              "ad hoc",
              "query",
              "run report",
            ],
            teamName: switchesFromUnassigned,
          },
        ]
      : []),

    // Actions (continued) — users who can write
    ...(canWrite
      ? [
          {
            id: "run-live-policy",
            label: "Run live policy",
            group: "Commands" as const,
            path: withTeamId(paths.NEW_POLICY),
            keywords: ["check", "compliance", "live", "ad hoc", "run policy"],
          },
          {
            id: "add-label",
            label: "Add label",
            group: "Commands" as const,
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
                  id: "add-user",
                  label: "Add user",
                  group: "Commands" as const,
                  path: paths.ADMIN_USERS_NEW_HUMAN,
                  keywords: [
                    "new user",
                    "create user",
                    "invite",
                    "account",
                    "human user",
                  ],
                },
                {
                  id: "add-api-only-user",
                  label: "Add API-only user",
                  group: "Commands" as const,
                  path: paths.ADMIN_USERS_NEW_API,
                  keywords: [
                    "api user",
                    "api only user",
                    "service account",
                    "token",
                    "create api user",
                    "create api only user",
                    "gitops user",
                    "add user",
                    "create user",
                  ],
                },
                // Create fleet — Premium-only, hidden in Primo Mode, and
                // hidden in GitOps Mode (ManageFleetsPage disables the
                // primary action in all three states).
                ...(isPremiumTier && !isPrimoMode && !isGitOpsMode
                  ? [
                      {
                        id: "create-fleet",
                        label: "Create fleet",
                        group: "Commands" as const,
                        path: `${paths.ADMIN_FLEETS}?create_fleet=1`,
                        keywords: [
                          "new fleet",
                          "add fleet",
                          "team",
                          "create team",
                          "add team",
                          "new team",
                        ],
                      },
                    ]
                  : []),
              ]
            : []),
        ]
      : []),

    // Theme toggle and Sign out — always available. Theme is a per-user
    // UI preference, not a write against Fleet data (setThemeMode is
    // exposed to every signed-in user via My Account → Theme), so it
    // sits outside the canWrite gate alongside Sign out.
    {
      id: "toggle-dark-mode",
      // isDarkMode comes through as reactive state from the parent
      // so the label re-renders when the theme flips externally.
      label: isDarkMode ? "Switch to light mode" : "Switch to dark mode",
      group: "Commands" as const,
      keywords: ["dark mode", "light mode", "theme", "toggle"],
      onAction: onToggleDarkMode,
    },
    {
      id: "sign-out",
      label: "Sign out",
      path: paths.LOGOUT,
      group: "Commands" as const,
      keywords: ["logout", "log out", "sign out"],
    },
  ];
};

export default buildCommandsItems;
