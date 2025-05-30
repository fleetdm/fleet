import React, { useContext } from "react";

import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

import SectionCard from "../../SectionCard";

const baseClass = "android-mdm-card";

interface ITurnOnAndroidMdmCardProps {
  onClickTurnOn: () => void;
}

const TurnOnAndroidMdmCard = ({
  onClickTurnOn,
}: ITurnOnAndroidMdmCardProps) => {
  return (
    <SectionCard
      className={baseClass}
      header="Turn on Android MDM"
      cta={<Button onClick={onClickTurnOn}>Turn on</Button>}
    >
      Enforce settings, OS updates, and more.
    </SectionCard>
  );
};

interface ITurnOffAndroidMdmCardProps {
  onClickEdit: () => void;
}

const TurnOffAndroidMdmCard = ({
  onClickEdit,
}: ITurnOffAndroidMdmCardProps) => {
  return (
    <SectionCard
      className={baseClass}
      iconName="success"
      cta={
        <Button onClick={onClickEdit} variant="text-icon">
          <Icon name="pencil" />
          Edit
        </Button>
      }
    >
      Android MDM turned on.
    </SectionCard>
  );
};

interface IAndroidMdmCardProps {
  turnOffAndroidMdm: () => void;
  editAndroidMdm: () => void;
}

const AndroidMdmCard = ({
  turnOffAndroidMdm,
  editAndroidMdm,
}: IAndroidMdmCardProps) => {
  const { isAndroidMdmEnabledAndConfigured } = useContext(AppContext);

  if (isAndroidMdmEnabledAndConfigured === undefined) {
    return null;
  }

  return isAndroidMdmEnabledAndConfigured ? (
    <TurnOffAndroidMdmCard onClickEdit={editAndroidMdm} />
  ) : (
    <TurnOnAndroidMdmCard onClickTurnOn={turnOffAndroidMdm} />
  );
};

export default AndroidMdmCard;
