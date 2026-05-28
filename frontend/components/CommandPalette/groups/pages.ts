import paths from "router/paths";

import { ICommandItem, ICommandPaletteContext } from "../helpers";
import { IDerivedContext } from "./derivations";

const buildPagesItems = (
  ctx: ICommandPaletteContext,
  derived: IDerivedContext
): ICommandItem[] => {
  const { search, canAccessControls, canAccessSettings, withTeamId } = ctx;
  const {
    hasTeamOrUnassigned,
    switchesFromUnassigned,
    switchesFromAllFleets,
  } = derived;

  return [
    {
      id: "dashboard",
      label: "Dashboard",
      group: "Pages" as const,
      path: withTeamId(paths.DASHBOARD),
      teamName: switchesFromUnassigned,
      keywords: ["home", "hosts", "activity", "platform"],
    },
    {
      id: "hosts",
      label: "Hosts",
      group: "Pages" as const,
      path: withTeamId(paths.MANAGE_HOSTS),
      keywords: [
        "devices",
        "hostname",
        "serial number",
        "manage",
        "endpoints",
        "machines",
        "computers",
      ],
    },
    ...(canAccessControls && hasTeamOrUnassigned
      ? [
          {
            id: "controls-page",
            label: "Controls",
            group: "Pages" as const,
            path: withTeamId(paths.CONTROLS),
            keywords: ["mdm", "os settings", "os updates"],
            teamName: switchesFromAllFleets,
          },
        ]
      : []),
    {
      id: "software-page",
      label: "Software",
      group: "Pages" as const,
      path: withTeamId(paths.SOFTWARE_INVENTORY),
      keywords: ["installed", "inventory", "titles", "library", "managed"],
    },
    {
      id: "reports",
      label: "Reports",
      group: "Pages" as const,
      path: withTeamId(paths.MANAGE_REPORTS),
      keywords: [
        "report",
        "sql",
        "gather data",
        "live query",
        // Legacy: "Queries" was renamed to "Reports" — users will type
        // the old term for a long time.
        "queries",
        "query",
        "saved queries",
      ],
      teamName: switchesFromUnassigned,
    },
    {
      id: "policies",
      label: "Policies",
      group: "Pages" as const,
      path: withTeamId(paths.MANAGE_POLICIES),
      keywords: [
        "compliance",
        "failing",
        "device health",
        "yara",
        "osquery",
        "sql",
      ],
    },
    ...(canAccessSettings
      ? [
          {
            id: "settings-page",
            label: "Settings",
            group: "Pages" as const,
            path: paths.ADMIN_SETTINGS,
            keywords: ["admin", "organization", "integrations"],
          },
        ]
      : []),
    {
      id: "labels",
      label: "Labels",
      group: "Pages" as const,
      path: paths.MANAGE_LABELS,
      keywords: ["group hosts", "filter", "dynamic", "manual"],
    },
    // "Users" lives in the Settings group only — having it in Pages too
    // surfaced two items with identical destinations.
    {
      id: "my-account",
      label: "My account",
      group: "Pages" as const,
      path: paths.ACCOUNT,
      keywords: [
        "profile",
        "password",
        "api token",
        "settings",
        "change password",
      ],
    },

    // Packs page — only visible when searching for "packs" or similar.
    // The companion "Create new pack" item lives in commands.ts.
    ...(/packs|create new pack|add new pack/.test(search.toLowerCase())
      ? [
          {
            id: "packs",
            label: "Packs",
            group: "Pages" as const,
            path: paths.MANAGE_PACKS,
            keywords: ["packs", "legacy", "scheduled queries"],
          },
        ]
      : []),
  ];
};

export default buildPagesItems;
