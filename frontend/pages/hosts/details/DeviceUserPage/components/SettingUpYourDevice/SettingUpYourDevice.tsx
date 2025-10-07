import Card from "components/Card";
import { ISetupStep } from "interfaces/setup";
import React from "react";
import InfoButton from "../InfoButton";
import SetupStatusTable from "./SetupStatusTable";

import { hasRemainingSetupSteps } from "../../helpers";

const baseClass = "setting-up-your-device";

interface ISettingUpYourDevice {
  setupSteps: ISetupStep[];
  toggleInfoModal: () => void;
}

const SettingUpYourDevice = ({
  setupSteps,
  toggleInfoModal,
}: ISettingUpYourDevice) => {
  let title;
  let message;
  if (hasRemainingSetupSteps(setupSteps)) {
    title = "Setting up your device...";
    message = `
      Your computer is currently being configured by your organization.
      Please don't attempt to restart or shut down the computer unless
      prompted to do so.
    `;
  } else {
    title = "Configuration complete";
    message =
      "Your computer has been successfully configured. Setup will continue momentarily.";
  }

  return (
    <div className={`${baseClass} main-content device-user`}>
      <Card borderRadiusSize="xxlarge" paddingSize="xlarge">
        <div className={`${baseClass}__header`}>
          <h2>{title}</h2>
          <InfoButton onClick={toggleInfoModal} />
        </div>
        <p>{message}</p>
        <SetupStatusTable statuses={setupSteps} />
      </Card>
    </div>
  );
};

export default SettingUpYourDevice;
