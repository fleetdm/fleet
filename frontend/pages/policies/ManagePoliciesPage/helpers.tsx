import React from "react";

import { IInstallSoftwareFormData } from "./components/InstallSoftwareModal/InstallSoftwareModal";

/** Creates a readable JSX element from the error message */
// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (
  result: PromiseRejectedResult,
  formData: IInstallSoftwareFormData,
  currentTeamName?: string
): JSX.Element => {
  const apiErrorMessage = result.reason.data.errors[0].reason;
  const parts = apiErrorMessage.split(/(software_title_id \d+|team_id \d+)/);

  const jsxElement = parts.map((part: string) => {
    if (part.startsWith("software_title_id")) {
      const swId = part.split(" ")[1];
      const software = formData.find(
        (item) => item.swIdToInstall?.toString() === swId
      );
      return software ? (
        <React.Fragment key={part}>
          <b>{software.swNameToInstall}</b> (ID: {swId})
        </React.Fragment>
      ) : (
        part
      );
    } else if (part.startsWith("team_id")) {
      return currentTeamName ? <b key={part}>{currentTeamName}</b> : part;
    }
    return <React.Fragment key={part}>{part}</React.Fragment>;
  });

  return <>Could not update policy. {jsxElement}</>;
};
