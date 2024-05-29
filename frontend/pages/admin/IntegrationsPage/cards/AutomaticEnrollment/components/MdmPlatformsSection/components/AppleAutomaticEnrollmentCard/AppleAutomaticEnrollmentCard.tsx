import React from "react";

import Card from "components/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";

const baseClass = "automatic-enrollment-card";

interface IAppleAutomaticEnrollmentCardProps {
  viewDetails: () => void;
  turnOn?: () => void;
  configured?: boolean;
}

const AppleAutomaticEnrollmentCard = ({
  viewDetails,
  turnOn,
  configured,
}: IAppleAutomaticEnrollmentCardProps) => {
  let icon = "";
  let msg =
    "To enable automatic enrollment for macOS devices, first turn on macOS MDM.";
  if (!turnOn && !configured) {
    msg =
      "Automatically enroll newly purchased macOS devices when they’re first unboxed and set up by your end user.";
  } else if (!turnOn && configured) {
    msg = "Automatic enrollment for macOS enabled.";
    icon = "success";
  }

  return (
    <Card
      className={`${baseClass} ${turnOn ? `${baseClass}__turn-on-mdm` : ""}`}
      color="gray"
    >
      <div>
        {!icon && <h3>Automatic enrollment for macOS hosts</h3>}
        <p>
          {icon ? (
            <span>
              <Icon name="success" />
              {msg}
            </span>
          ) : (
            msg
          )}
        </p>
      </div>
      {turnOn && (
        <Button
          className="apple-details-button"
          onClick={turnOn}
          variant="text-icon"
        >
          Turn on MDM
        </Button>
      )}
      {!turnOn && !configured && (
        <Button
          className="apple-details-button"
          onClick={viewDetails}
          variant="brand"
        >
          Enable
        </Button>
      )}
      {!turnOn && configured && (
        <Button
          className="apple-details-button"
          onClick={viewDetails}
          variant="text-icon"
        >
          <Icon name="pencil" />
          Edit
        </Button>
      )}
    </Card>
  );
};

export default AppleAutomaticEnrollmentCard;
