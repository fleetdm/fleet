import React, { useContext } from "react";

import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import SectionCard from "../../SectionCard";

const baseClass = "windows-mdm-card";

interface ITurnOnWindowsMdmCardProps {
  onClickTurnOn: () => void;
}

const TurnOnWindowsMdmCard = ({
  onClickTurnOn,
}: ITurnOnWindowsMdmCardProps) => {
  return (
    <SectionCard
      className={baseClass}
      header="Turn on Windows MDM"
      cta={
        <Button variant="brand" onClick={onClickTurnOn}>
          Turn on
        </Button>
      }
    >
      Turn MDM on for Windows hosts with fleetd.
    </SectionCard>
  );
};

interface ITurnOffWindowsMdmCardProps {
  onClickEdit: () => void;
}

const TurnOffWindowsMdmCard = ({
  onClickEdit,
}: ITurnOffWindowsMdmCardProps) => {
  return (
    <SectionCard
      iconName="success"
      cta={
        <Button onClick={onClickEdit} variant="text-icon">
          <Icon name="pencil" />
          Edit
        </Button>
      }
    >
      Windows MDM turned on (servers excluded).
    </SectionCard>
  );
};

interface IWindowsMdmCardProps {
  turnOnWindowsMdm: () => void;
  editWindowsMdm: () => void;
}

const WindowsMdmCard = ({
  turnOnWindowsMdm,
  editWindowsMdm,
}: IWindowsMdmCardProps) => {
  const { config } = useContext(AppContext);

  const isWindowsMdmEnabled =
    config?.mdm?.windows_enabled_and_configured ?? false;

  return isWindowsMdmEnabled ? (
    <TurnOffWindowsMdmCard onClickEdit={editWindowsMdm} />
  ) : (
    <TurnOnWindowsMdmCard onClickTurnOn={turnOnWindowsMdm} />
  );
};

export default WindowsMdmCard;
