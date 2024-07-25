import React from "react";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Card from "components/Card";
import DataError from "components/DataError";
import { AxiosError } from "axios";
import { IMdmApple } from "interfaces/mdm";

const baseClass = "apple-mdm-card";

interface ITurnOnAppleMdmProps {
  onClickTurnOn: () => void;
}

const TurnOnAppleMdm = ({ onClickTurnOn }: ITurnOnAppleMdmProps) => {
  return (
    <div className={`${baseClass}__turn-on-apple-mdm`}>
      <div>
        <h3>Turn on Apple (macOS, iOS, iPadOS) MDM</h3>
        <p>Enforce settings, OS updates, disk encryption, and more.</p>
      </div>
      <Button variant="brand" onClick={onClickTurnOn}>
        Turn on
      </Button>
    </div>
  );
};

interface ITurnOffAppleMdmProps {
  onClickDetails: () => void;
}

const SeeDetailsAppleMdm = ({ onClickDetails }: ITurnOffAppleMdmProps) => {
  return (
    <div className={`${baseClass}__turn-off-mac-os`}>
      <div>
        <Icon name="success" />
        <p>Apple (macOS, iOS, iPadOS) MDM turned on.</p>
      </div>
      <Button onClick={onClickDetails} variant="text-icon">
        <Icon name="pencil" />
        Edit
      </Button>
    </div>
  );
};

interface IAppleMdmCardProps {
  appleAPNInfo: IMdmApple | undefined;
  errorData: AxiosError | null;
  turnOnAppleMdm: () => void;
  viewDetails: () => void;
}

/**
 * This component is responsible for showing the correct UI for the Apple MDM card.
 * We pass in the appleAPNInfo and errorData from the MdmSettings component because
 * we need to make that API call higher up in the component tree to correctly show
 * loading states on the page.
 */
const AppleMdmCard = ({
  appleAPNInfo,
  errorData,
  turnOnAppleMdm,
  viewDetails,
}: IAppleMdmCardProps) => {
  // The API returns an error if MDM is turned off or APNS is not configured yet.
  // If there is any other error we will show the DataError component.
  const showError =
    errorData !== null && errorData.status !== 404 && errorData.status !== 400;

  if (showError) {
    return <DataError />;
  }

  return (
    <Card className={baseClass} color="gray">
      {appleAPNInfo !== undefined ? (
        <SeeDetailsAppleMdm onClickDetails={viewDetails} />
      ) : (
        <TurnOnAppleMdm onClickTurnOn={turnOnAppleMdm} />
      )}
    </Card>
  );
};

export default AppleMdmCard;
