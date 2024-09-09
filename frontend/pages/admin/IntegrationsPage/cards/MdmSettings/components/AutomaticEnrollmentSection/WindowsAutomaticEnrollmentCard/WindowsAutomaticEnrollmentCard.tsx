import React from "react";

import Card from "components/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";

const baseClass = "automatic-enrollment-card";

interface IWindowsAutomaticEnrollmentCardProps {
  viewDetails: () => void;
}

const WindowsAutomaticEnrollmentCard = ({
  viewDetails,
}: IWindowsAutomaticEnrollmentCardProps) => {
  return (
    <Card className={baseClass} color="gray">
      <div>
        <h3>Windows automatic enrollment</h3>
        <p>
          To use automatic enrollment for Windows hosts and Windows Autopilot
          you need to connect Fleet to Azure AD first.
        </p>
      </div>
      <Button
        className="windows-details-button"
        onClick={viewDetails}
        variant="text-icon"
      >
        Details <Icon name="chevron-right" color="core-fleet-blue" />
      </Button>
    </Card>
  );
};

export default WindowsAutomaticEnrollmentCard;
