import Card from "components/Card";
import { ISetupSoftwareStatus } from "interfaces/software";
import React from "react";
import InfoButton from "../InfoButton";
import SetupSoftwareStatusTable from "./SetupSoftwareStatusTable";

const baseClass = "setting-up-your-device";

interface ISettingUpYourDevice {
  softwareStatuses: ISetupSoftwareStatus[];
  toggleInfoModal: () => void;
}

const SettingUpYourDevice = ({
  softwareStatuses,
  toggleInfoModal,
}: ISettingUpYourDevice) => {
  return (
    <div className={`${baseClass} main-content device-user`}>
      <span className={`${baseClass}__header`}>
        <h1 className={`${baseClass}__title`}>My device</h1>
        <InfoButton onClick={toggleInfoModal} />
      </span>
      <Card borderRadiusSize="xxlarge" paddingSize="xlarge">
        <h2>Setting up your device...</h2>
        <p>
          Your computer is currently being configured by your organization.
          Please don&apos;t attempt to restart or shut down the computer unless
          prompted to do so.
        </p>
        <SetupSoftwareStatusTable statuses={softwareStatuses} />
      </Card>
    </div>
  );
};

export default SettingUpYourDevice;
