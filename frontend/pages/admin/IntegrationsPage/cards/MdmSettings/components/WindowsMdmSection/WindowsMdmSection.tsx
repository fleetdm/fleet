import React, { useContext } from "react";

import Card from "components/Card/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import configAPI from "services/entities/config";
import { NotificationContext } from "context/notification";

const baseClass = "windows-mdm-section";

const TurnOnWindowsMdm = () => {
  const { renderFlash } = useContext(NotificationContext);

  const onTurnOnMdm = async () => {
    try {
      await configAPI.update({
        mdm: {
          windows_enabled_and_configured: true,
        },
      });
      renderFlash("success", "Windows MDM turned on (servers excluded).");
    } catch {}
  };

  return (
    <div className={`${baseClass}__turn-on-windows`}>
      <div>
        <h3>Turn on Windows MDM</h3>
        <p>Turn MDM on for Windows hosts with fleetd.</p>
      </div>
      <Button onClick={onTurnOnMdm}>Turn on</Button>
    </div>
  );
};

const TurnOffWindowsMdm = () => {
  return (
    <div className={`${baseClass}__turn-off-windows`}>
      <Icon name="success" />
      <p>Windows MDM turned on (servers excluded).</p>
    </div>
  );
};

const WindowsMdmSection = () => {
  return (
    <Card className={baseClass} color="purple">
      <TurnOnWindowsMdm />
      <p>this is something</p>
      {/* <TurnOffWindowsMdm /> */}
    </Card>
  );
};

export default WindowsMdmSection;
