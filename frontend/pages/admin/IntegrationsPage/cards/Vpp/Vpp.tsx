import React from "react";

import Card from "components/Card";
import SectionHeader from "components/SectionHeader";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "vpp";

interface IVppCardProps {
  isOn: boolean;
  onTurnOnVpp: () => void;
  onEditVpp: () => void;
}

const VppCard = ({ isOn, onTurnOnVpp, onEditVpp }: IVppCardProps) => {
  const icon = isOn ? "success" : "";

  const isOnContent = (
    <>
      <p>
        <span>
          <Icon name="success" />
          Volume Purchasing Program (VPP) enabled.
        </span>
      </p>
      <Button onClick={onEditVpp} variant="text-icon">
        <Icon name="pencil" />
        Edit
      </Button>
    </>
  );

  const isOffContent = (
    <>
      <div>
        <h3>Volume Purchasing Program (VPP)</h3>
        <p>
          Install apps from Apple&apos;s App Store purchased through Apple
          Business Manager.
        </p>
      </div>
      <Button onClick={onTurnOnVpp} variant="brand">
        Enable
      </Button>
    </>
  );

  return (
    <Card className={`${baseClass}__card`} color="gray">
      {isOn ? isOnContent : isOffContent}
    </Card>
  );
};

interface IVppProps {}

const Vpp = ({}: IVppProps) => {
  return (
    <div className={baseClass}>
      <SectionHeader title="Volume Purchasing Program (VPP)" />
      <VppCard isOn />
    </div>
  );
};

export default Vpp;
