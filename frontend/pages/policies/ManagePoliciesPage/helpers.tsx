import React from "react";

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
