import {
  IPolicyStats,
  OtherAutomationType,
  PolicyAutomationType,
} from "interfaces/policy";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";
import { IPoliciesFilterParams } from "services/entities/policies";

/** The Policies table's automation filter. "patch" filters by policy type. */
export type PoliciesAutomationFilter = PolicyAutomationType | "patch";

export const getPoliciesFilterParams = (
  filter: PoliciesAutomationFilter | null
): Pick<IPoliciesFilterParams, "automationType" | "type"> => {
  if (!filter) {
    return {};
  }
  return filter === "patch" ? { type: "patch" } : { automationType: filter };
};

export type AutomationDisplayType =
  | "software"
  | "script"
  | "profile"
  | "calendar"
  | "conditional_access"
  | "other";

interface ISoftwareAutomationData {
  type: "software";
  name: string;
  /** Raw software name passed to SoftwareIcon for name-based fallback matching.
   * Display-name overrides won't match the known-icon lookup (e.g. FMAs without
   * a custom icon_url), so we keep the raw name available alongside `name`. */
  iconName: string;
  softwareTitleId: number;
  iconUrl?: string | null;
}

interface INonSoftwareAutomationData {
  type: Exclude<AutomationDisplayType, "software">;
  name: string;
  iconName?: never;
  softwareTitleId?: never;
  iconUrl?: never;
}

export type IAutomationData =
  | ISoftwareAutomationData
  | INonSoftwareAutomationData;

/** Returns the policy's automation, as a list for the Automation column. */
export const getAutomationsForPolicy = (
  policy: Pick<
    IPolicyStats,
    | "automation_type"
    | "install_software"
    | "run_script"
    | "resend_configuration_profile"
  >,
  otherAutomationType?: OtherAutomationType
): IAutomationData[] => {
  switch (policy.automation_type) {
    case "software":
      return policy.install_software
        ? [
            {
              type: "software",
              name: getDisplayedSoftwareName(
                policy.install_software.name,
                policy.install_software.display_name
              ),
              iconName: policy.install_software.name,
              softwareTitleId: policy.install_software.software_title_id,
              iconUrl: policy.install_software.icon_url,
            },
          ]
        : [];
    case "script":
      return policy.run_script
        ? [{ type: "script", name: policy.run_script.name }]
        : [];
    case "configuration_profile":
      return policy.resend_configuration_profile
        ? [
            {
              type: "profile",
              name: policy.resend_configuration_profile.name,
            },
          ]
        : [];
    case "calendar":
      return [{ type: "calendar", name: "Maintenance window" }];
    case "conditional_access":
      return [{ type: "conditional_access", name: "Conditional access" }];
    case "webhook_or_ticket":
      return [
        {
          type: "other",
          name: otherAutomationType === "ticket" ? "Ticket" : "Webhook",
        },
      ];
    default:
      return [];
  }
};
