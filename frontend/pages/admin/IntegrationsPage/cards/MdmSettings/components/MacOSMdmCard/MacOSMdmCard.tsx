import React from "react";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Card from "components/Card";

const baseClass = "mac-os-mdm-card";

interface ITurnOnMacOSMdmProps {
  onClickTurnOn: () => void;
}

const TurnOnMacOSMdm = ({ onClickTurnOn }: ITurnOnMacOSMdmProps) => {
  return (
    <div className={`${baseClass}__turn-on-mac-os`}>
      <div>
        <h3>Turn on macOS MDM</h3>
        <p>
          Connect Fleet to Apple Push Certificates Portal to change settings and
          install software on your macOS hosts.
        </p>
      </div>
      <Button onClick={onClickTurnOn}>Connect APNS</Button>
    </div>
  );
};

interface ITurnOffMacOSMdmProps {
  onClickDetails: () => void;
}

const TurnOffMacOSMdm = ({ onClickDetails }: ITurnOffMacOSMdmProps) => {
  return (
    <div className={`${baseClass}__turn-off-mac-os`}>
      <div>
        <Icon name="success" />
        <p>macOS MDM turned on</p>
      </div>
      <Button onClick={onClickDetails} variant="text-icon">
        Details
        <Icon name="chevron" direction="right" color="core-fleet-blue" />
      </Button>
    </div>
  );
};

interface IMacOSMdmCardProps {
  isEnabled: boolean;
  turnOnMacOSMdm: () => void;
  viewDetails: () => void;
}

const MacOSMdmCard = ({
  isEnabled,
  turnOnMacOSMdm,
  viewDetails,
}: IMacOSMdmCardProps) => {
  return (
    <Card className={baseClass} color="gray">
      {isEnabled ? (
        <TurnOffMacOSMdm onClickDetails={viewDetails} />
      ) : (
        <TurnOnMacOSMdm onClickTurnOn={turnOnMacOSMdm} />
      )}
    </Card>
  );
};

export default MacOSMdmCard;
