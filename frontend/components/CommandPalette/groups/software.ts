import paths from "router/paths";

import { ICommandItem, ICommandPaletteContext } from "../helpers";
import { IDerivedContext } from "./derivations";

const buildSoftwareItems = (
  ctx: ICommandPaletteContext,
  derived: IDerivedContext
): ICommandItem[] => {
  const { isPremiumTier, withTeamId } = ctx;
  const { hasTeamOrUnassigned } = derived;

  return [
    {
      id: "software",
      label: "Software inventory",
      group: "Software" as const,
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
      group: "Software" as const,
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
      group: "Software" as const,
      path: withTeamId(paths.SOFTWARE_VULNERABILITIES),
      keywords: ["cve", "cvss", "exploit", "vulnerable software"],
    },
    // Library is available for any team including unassigned, but not "All fleets"
    ...(isPremiumTier && hasTeamOrUnassigned
      ? [
          {
            id: "software-library",
            label: "Software library",
            group: "Software" as const,
            path: withTeamId(paths.SOFTWARE_LIBRARY),
            keywords: [
              "managed",
              "installable",
              "packages",
              "self-service",
              "library",
            ],
          },
        ]
      : []),
  ];
};

export default buildSoftwareItems;
