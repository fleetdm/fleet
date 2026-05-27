import paths from "router/paths";

import { ICommandItem, ICommandPaletteContext } from "../helpers";
import { IDerivedContext } from "./derivations";

const buildAutomationsItems = (
  ctx: ICommandPaletteContext,
  derived: IDerivedContext
): ICommandItem[] => {
  const {
    canAccessSettings,
    canManageSoftwareAutomations,
    canManagePolicyAutomations,
    canWrite,
    currentTeam,
    hasTeamSelected,
    isPremiumTier,
    isPrimoMode,
    withTeamId,
  } = ctx;
  const { isUnassigned, switchesFromUnassigned, hasTeamOrUnassigned } = derived;

  return [
    // Manage automations — software. Normally All-fleets-only, but in
    // Primo Mode the single fleet acts as "all fleets" so the destination
    // page (SoftwarePage) accepts it too.
    ...(canManageSoftwareAutomations &&
    ((!hasTeamSelected && !isUnassigned) || isPrimoMode)
      ? [
          {
            id: "manage-software-automations",
            label: "Manage software automations",
            group: "Automations" as const,
            path: `${paths.SOFTWARE_INVENTORY}?manage_automations=1`,
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
            group: "Automations" as const,
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
            group: "Automations" as const,
            path: withTeamId(`${paths.MANAGE_REPORTS}?manage_automations=1`),
            keywords: ["report", "logging", "destination"],
            teamName: switchesFromUnassigned,
          },
        ]
      : []),

    // Manage automations — policies (admins and maintainers)
    ...(canManagePolicyAutomations
      ? [
          {
            id: "manage-policy-automations",
            label: "Manage policy automations",
            group: "Automations" as const,
            path: withTeamId(paths.MANAGE_POLICIES),
            keywords: ["failing", "webhook", "jira", "zendesk"],
            subItems: [
              {
                id: "manage-policy-automations-webhooks",
                label: "Tickets & webhooks",
                path: withTeamId(
                  `${paths.MANAGE_POLICIES}?manage_automations=webhooks`
                ),
                keywords: ["jira", "zendesk", "failing"],
              },
              // Team-scoped policy automations (Premium-only). The
              // policies page allows install_software / run_script /
              // conditional_access on No team / Unassigned, so those
              // three use `hasTeamOrUnassigned`. Calendar events stay
              // on `hasTeamSelected` — the page disables them when
              // there's no specific team.
              ...(isPremiumTier && hasTeamOrUnassigned
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
                  ]
                : []),
              ...(isPremiumTier && hasTeamSelected
                ? [
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
                  ]
                : []),
              ...(isPremiumTier && hasTeamOrUnassigned
                ? [
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

export default buildAutomationsItems;
