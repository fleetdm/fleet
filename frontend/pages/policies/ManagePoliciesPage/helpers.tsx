import React from "react";

import { IPolicyStats, OtherAutomationType } from "interfaces/policy";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import { IInstallSoftwareFormData } from "./components/InstallSoftwareModal/InstallSoftwareModal";
import { IPolicyRunScriptFormData } from "./components/PolicyRunScriptModal/PolicyRunScriptModal";

export type AutomationDisplayType =
  | "software"
  | "script"
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

/** Returns an ordered list of automations configured for a policy. */
export const getAutomationsForPolicy = (
  policy: Pick<
    IPolicyStats,
    | "install_software"
    | "run_script"
    | "calendar_events_enabled"
    | "conditional_access_enabled"
    | "webhook"
  >,
  otherAutomationType?: OtherAutomationType
): IAutomationData[] => {
  const automations: IAutomationData[] = [];

  if (policy.install_software) {
    automations.push({
      type: "software",
      name: getDisplayedSoftwareName(
        policy.install_software.name,
        policy.install_software.display_name
      ),
      iconName: policy.install_software.name,
      softwareTitleId: policy.install_software.software_title_id,
      iconUrl: policy.install_software.icon_url,
    });
  }
  if (policy.run_script) {
    automations.push({
      type: "script",
      name: policy.run_script.name,
    });
  }
  if (policy.calendar_events_enabled) {
    automations.push({
      type: "calendar",
      name: "Maintenance window",
    });
  }
  if (policy.conditional_access_enabled) {
    automations.push({
      type: "conditional_access",
      name: "Conditional access",
    });
  }
  if (policy.webhook === "On") {
    automations.push({
      type: "other",
      name: otherAutomationType === "ticket" ? "Ticket" : "Webhook",
    });
  }

  return automations;
};

/** Creates a readable JSX element from the error message */
export const getInstallSoftwareErrorMessage = (
  result: PromiseRejectedResult,
  formData: IInstallSoftwareFormData,
  currentTeamName?: string
): JSX.Element => {
  const apiErrorMessage = result.reason.data.errors[0].reason;
  const parts = apiErrorMessage.split(
    /(Software title with ID \d+|team ID \d+)/i
  );

  const jsxElement = parts.map((part: string) => {
    if (part.startsWith("Software title with ID")) {
      const swId = part.match(/\d+/)?.[0];
      const policy = formData.find(
        (item) => item.swIdToInstall?.toString() === swId
      );
      return policy ? (
        <React.Fragment key={part}>
          <b>{policy.swNameToInstall}</b> (ID: {swId})
        </React.Fragment>
      ) : (
        part
      );
    } else if (part.startsWith("team ID")) {
      return currentTeamName ? <b key={part}>{currentTeamName}</b> : part;
    }
    return <React.Fragment key={part}>{part}</React.Fragment>;
  });

  return <>Could not update policy. {jsxElement}</>;
};

export const getRunScriptErrorMessage = (
  result: PromiseRejectedResult,
  formData: IPolicyRunScriptFormData,
  currentTeamName?: string
): JSX.Element => {
  const apiErrorMessage = result.reason.data.errors[0].reason;
  const parts = apiErrorMessage.split(/(Script with ID \d+|team ID \d+)/i);

  const jsxElement = parts.map((part: string) => {
    if (part.startsWith("Script with ID")) {
      const scriptId = part.match(/\d+/)?.[0];
      const policy = formData.find(
        (item) => item.scriptIdToRun?.toString() === scriptId
      );

      return policy ? (
        <React.Fragment key={part}>
          <b>{policy.scriptNameToRun}</b> (ID: {scriptId})
        </React.Fragment>
      ) : (
        part
      );
    } else if (part.startsWith("team ID")) {
      return currentTeamName ? <b key={part}>{currentTeamName}</b> : part;
    }
    return <React.Fragment key={part}>{part}</React.Fragment>;
  });

  return <>Could not update policy. {jsxElement}</>;
};
