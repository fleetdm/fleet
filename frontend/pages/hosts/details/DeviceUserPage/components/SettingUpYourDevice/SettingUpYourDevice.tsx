import Card from "components/Card";
import { ISetupSoftwareStatus } from "interfaces/software";
import HostHeader from "pages/hosts/details/cards/HostHeader";
import React from "react";

const baseClass = "setting-up-your-device";

interface ISettingUpYourDevice {
  softwareStatuses: ISetupSoftwareStatus[];
}

const SettingUpYourDevice = ({ softwareStatuses }: ISettingUpYourDevice) => {
  return (
    <div className={`${baseClass} main-content device-user`}>
      <h1 className={`${baseClass}__header`}>My device</h1>
      <Card
        borderRadiusSize="xxlarge"
        paddingSize="xlarge"
        includeShadow
        // className={classNames}
      >
        <></>
      </Card>
    </div>
  );
};

export default SettingUpYourDevice;
