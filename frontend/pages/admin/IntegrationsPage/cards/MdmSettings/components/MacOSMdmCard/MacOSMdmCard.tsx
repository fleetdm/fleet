import React from "react";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Card from "components/Card";
import DataError from "components/DataError";
import { AxiosError } from "axios";
import { IMdmApple } from "interfaces/mdm";

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
      <Button variant="brand" onClick={onClickTurnOn}>
        Connect APNS
      </Button>
    </div>
  );
};

interface ITurnOffMacOSMdmProps {
  onClickDetails: () => void;
}

const SeeDetailsMacOSMdm = ({ onClickDetails }: ITurnOffMacOSMdmProps) => {
  return (
    <div className={`${baseClass}__turn-off-mac-os`}>
      <div>
        <Icon name="success" />
        <p>macOS MDM turned on</p>
      </div>
      <Button onClick={onClickDetails} variant="text-icon">
        Details
        <Icon name="chevron-right" color="core-fleet-blue" />
      </Button>
    </div>
  );
};

interface IMacOSMdmCardProps {
  appleAPNInfo: IMdmApple | undefined;
  errorData: AxiosError | null;
  turnOnMacOSMdm: () => void;
  viewDetails: () => void;
}

/**
 * This compoent is responsible for showing the correct UI for the macOS MDM card.
 * We pass in the appleAPNInfo and errorData from the MdmSettings component because
 * we need to make that API call higher up in the component tree to correctly show
 * loading states on the page.
 */
const MacOSMdmCard = ({
  appleAPNInfo,
  errorData,
  turnOnMacOSMdm,
  viewDetails,
}: IMacOSMdmCardProps) => {
  // The API returns a 404 error if APNS is not configured yet. If there is any
  // other error we will show the DataError component.
  const showError = errorData !== null && errorData.status !== 404;

  if (showError) {
    return <DataError />;
  }

  return (
    <Card className={baseClass} color="gray">
      {appleAPNInfo !== undefined ? (
        <SeeDetailsMacOSMdm onClickDetails={viewDetails} />
      ) : (
        <TurnOnMacOSMdm onClickTurnOn={turnOnMacOSMdm} />
      )}
    </Card>
  );
};

export default MacOSMdmCard;
