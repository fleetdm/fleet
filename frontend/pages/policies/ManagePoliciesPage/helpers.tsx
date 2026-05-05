import React from "react";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import { IInstallSoftwareFormData } from "./components/InstallSoftwareModal/InstallSoftwareModal";
import { IPolicyRunScriptFormData } from "./components/PolicyRunScriptModal/PolicyRunScriptModal";

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

/** Derives a comma-separated string of automation types enabled for a policy.
 *  Returns "---" if no automations are enabled. */
export const getAutomationTypesString = (policy: {
  install_software?: { software_title_id: number };
  run_script?: { id: number };
  calendar_events_enabled: boolean;
  conditional_access_enabled: boolean;
  webhook?: string;
}): string => {
  const types: string[] = [];

  if (policy.install_software) {
    types.push("Software");
  }
  if (policy.run_script) {
    types.push("Script");
  }
  if (policy.calendar_events_enabled) {
    types.push("Calendar");
  }
  if (policy.conditional_access_enabled) {
    types.push("Conditional access");
  }
  if (policy.webhook === "On") {
    types.push("Other");
  }

  if (types.length === 0) return DEFAULT_EMPTY_CELL_VALUE;
  // Lowercase all types after the first to match sentence-case display
  return types.map((t, i) => (i === 0 ? t : t.toLowerCase())).join(", ");
};
