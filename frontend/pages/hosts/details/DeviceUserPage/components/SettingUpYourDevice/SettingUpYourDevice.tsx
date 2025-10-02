import Card from "components/Card";
import { ISetupStep } from "interfaces/setup";
import React from "react";
import InfoButton from "../InfoButton";
import SetupSoftwareStatusTable from "./SetupStatusTable";

const baseClass = "setting-up-your-device";

interface ISettingUpYourDevice {
  softwareStatuses: ISetupStep[];
  toggleInfoModal: () => void;
}

const SettingUpYourDevice = ({
  softwareStatuses,
  toggleInfoModal,
}: ISettingUpYourDevice) => {
  return (
    <div className={`${baseClass} main-content device-user`}>
      {/* <span className={`${baseClass}__header`}>
        <h1 className={`${baseClass}__title`}>My device</h1>
        <InfoButton onClick={toggleInfoModal} />
      </span> */}
      <Card borderRadiusSize="xxlarge" paddingSize="xlarge" includeShadow>
        <div className={`${baseClass}__header`}>
          <h2>Setting up your device...</h2>
          <InfoButton onClick={toggleInfoModal} />
        </div>
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
