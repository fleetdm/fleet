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
    canManageReportAutomations,
    hasTeamSelected,
    isPrimoMode,
    withTeamId,
  } = ctx;
  const { isUnassigned, switchesFromUnassigned } = derived;

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
            keywords: [
              "manage automations",
              "vulnerability",
              "webhook",
              "jira",
              "zendesk",
            ],
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
            keywords: [
              "manage automations",
              "activity feed",
              "webhook",
              "audit log",
            ],
          },
        ]
      : []),

    // Manage automations — reports. Mirrors ManageQueriesPage's
    // `canManageAutomations` (admin-only). The destination page opens
    // the modal from `?manage_automations=1` without re-checking the
    // role, so the palette must gate strictly here.
    ...(canManageReportAutomations
      ? [
          {
            id: "manage-report-automations",
            label: "Manage report automations",
            group: "Automations" as const,
            path: withTeamId(`${paths.MANAGE_REPORTS}?manage_automations=1`),
            keywords: [
              "manage automations",
              "report",
              "logging",
              "destination",
            ],
            teamName: switchesFromUnassigned,
          },
        ]
      : []),

    // Manage automations — policies (admins and maintainers). Mirrors
    // the reports pattern: ManagePoliciesPage reads ?manage_automations=1
    // and opens AutomationsModal. The page re-checks role +
    // hasPoliciesToAutomate before opening, then strips the param.
    //
    // Keywords match the sections AutomationsModal actually renders for
    // the current fleet scope (see AutomationsModal.tsx:230, 244, 281):
    //   - All fleets:  Webhooks/tickets only
    //   - Unassigned:  Webhooks/tickets + Conditional access
    //   - Team:        Webhooks/tickets + Calendar + Conditional access
    // Hiding the inapplicable keywords keeps the palette from matching
    // (e.g.) "calendar" to a fleet where the section isn't rendered.
    ...(canManagePolicyAutomations
      ? [
          {
            id: "manage-policy-automations",
            label: "Manage policy automations",
            group: "Automations" as const,
            path: withTeamId(`${paths.MANAGE_POLICIES}?manage_automations=1`),
            keywords: [
              "manage automations",
              "failing",
              "tickets",
              "webhook",
              "jira",
              "zendesk",
              ...(hasTeamSelected
                ? [
                    "calendar",
                    "google calendar",
                    "scheduled maintenance windows",
                  ]
                : []),
              ...(hasTeamSelected || isUnassigned
                ? ["conditional access", "sso", "okta", "entra", "intune"]
                : []),
            ],
          },
        ]
      : []),
  ];
};

export default buildAutomationsItems;
