import React from "react";

import Card from "components/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";

const baseClass = "automatic-enrollment-card";

interface IAppleAutomaticEnrollmentCardProps {
  isAppleMdmOn: boolean;
  viewDetails: () => void;
  configured?: boolean;
}

const AppleAutomaticEnrollmentCard = ({
  isAppleMdmOn,
  viewDetails,
  configured,
}: IAppleAutomaticEnrollmentCardProps) => {
  let icon = "";
  let msg =
    "To enable automatic enrollment for macOS, iOS, and iPadOS hosts, first turn on Apple MDM.";
  if (isAppleMdmOn && !configured) {
    msg =
      "Add an Apple Business Manager (ABM) connection to automatically enroll newly " +
      "purchased Apple hosts when they're first unboxed and set up by your end users.";
  } else if (isAppleMdmOn && configured) {
    msg = "Automatic enrollment for Apple (macOS, iOS, iPadOS) hosts enabled.";
    icon = "success";
  }

  return (
    <Card
      className={`${baseClass} ${
        !isAppleMdmOn ? `${baseClass}__turn-on-mdm` : ""
      }`}
      color="gray"
    >
      <div>
        {!icon && (
          <h3>Automatic enrollment for Apple (macOS, iOS, iPadOS) hosts.</h3>
        )}
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
      {isAppleMdmOn && !configured && (
        <Button
          className="apple-details-button"
          onClick={viewDetails}
          variant="brand"
        >
          Add ABM
        </Button>
      )}
      {isAppleMdmOn && configured && (
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
