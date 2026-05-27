import paths from "router/paths";

import { ICommandItem, ICommandPaletteContext } from "../helpers";
import { IDerivedContext } from "./derivations";

export const buildControlsItems = (
  ctx: ICommandPaletteContext,
  derived: IDerivedContext
): ICommandItem[] => {
  const { canAccessControls, isPremiumTier, isTechnician, withTeamId } = ctx;
  const { hasTeamOrUnassigned } = derived;

  // Controls pages don't support "All fleets" (includeAllTeams: false),
  // so only show when a team or unassigned is selected. Also gated by
  // canAccessControls (maintainers, admins, technicians).
  if (!canAccessControls || !hasTeamOrUnassigned) return [];

  return [
    {
      id: "controls-os-updates",
      label: "OS updates",
      group: "Controls" as const,
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
      group: "Controls" as const,
      path: withTeamId(paths.CONTROLS_OS_SETTINGS),
      keywords: ["enforce", "remotely", "profiles"],
      subItems: [
        // Disk encryption is Premium-only.
        ...(isPremiumTier
          ? [
              {
                id: "controls-disk-encryption",
                label: "Disk encryption",
                path: withTeamId(paths.CONTROLS_DISK_ENCRYPTION),
                keywords: ["filevault", "bitlocker", "recovery key"],
              },
            ]
          : []),
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
        // Certificates and Passwords — Premium-only, and not
        // available to technicians.
        ...(isPremiumTier && !isTechnician
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
    // Setup experience sub-pages — Premium-only.
    ...(isPremiumTier
      ? [
          {
            id: "controls-setup-experience",
            label: "Setup experience",
            group: "Controls" as const,
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
        ]
      : []),
    // Scripts
    {
      id: "controls-scripts",
      label: "Scripts",
      group: "Controls" as const,
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
      group: "Controls" as const,
      path: withTeamId(paths.CONTROLS_VARIABLES),
      keywords: ["custom", "scripts", "profiles"],
    },
  ];
};
