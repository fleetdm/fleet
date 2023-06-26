import React, { useContext } from "react";

import Card from "components/Card/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import configAPI from "services/entities/config";
import { NotificationContext } from "context/notification";

const baseClass = "windows-mdm-section";

interface ITurnOnWindowsMdmProps {
  onClickTurnOn: () => void;
}
const TurnOnWindowsMdm = ({ onClickTurnOn }: ITurnOnWindowsMdmProps) => {
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
      <Button onClick={onClickTurnOn}>Turn on</Button>
    </div>
  );
};

interface ITurnOffWindowsMdmProps {
  onClickEdit: () => void;
}

const TurnOffWindowsMdm = ({ onClickEdit }: ITurnOffWindowsMdmProps) => {
  return (
    <div className={`${baseClass}__turn-off-windows`}>
      <Icon name="success" />
      <p>Windows MDM turned on (servers excluded).</p>
      <Button onClick={onClickEdit}>Edit</Button>
    </div>
  );
};

interface IWindowsMdmSectionProps {
  turnOnWindowsMdm: () => void;
  editWindowsMdm: () => void;
}

const WindowsMdmSection = ({
  turnOnWindowsMdm,
  editWindowsMdm,
}: IWindowsMdmSectionProps) => {
  return (
    <Card className={baseClass} color="purple">
      <TurnOnWindowsMdm onClickTurnOn={turnOnWindowsMdm} />
      {/* <TurnOffWindowsMdm onClickEdit={editWindowsMdm} /> */}
    </Card>
  );
};

export default WindowsMdmSection;
