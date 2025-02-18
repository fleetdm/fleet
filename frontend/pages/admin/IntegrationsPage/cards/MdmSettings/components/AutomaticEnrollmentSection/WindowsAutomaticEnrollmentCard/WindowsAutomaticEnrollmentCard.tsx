import React from "react";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import SectionCard from "../../SectionCard";

interface IWindowsAutomaticEnrollmentCardProps {
  viewDetails: () => void;
}

const WindowsAutomaticEnrollmentCard = ({
  viewDetails,
}: IWindowsAutomaticEnrollmentCardProps) => {
  return (
    <SectionCard
      header="Windows automatic enrollment"
      cta={
        <Button
          className="windows-details-button"
          onClick={viewDetails}
          variant="text-icon"
        >
          Details <Icon name="chevron-right" color="core-fleet-blue" />
        </Button>
      }
    >
      To use automatic enrollment for Windows hosts and Windows Autopilot you
      need to connect Fleet to Azure AD first.
    </SectionCard>
  );
};

export default WindowsAutomaticEnrollmentCard;
