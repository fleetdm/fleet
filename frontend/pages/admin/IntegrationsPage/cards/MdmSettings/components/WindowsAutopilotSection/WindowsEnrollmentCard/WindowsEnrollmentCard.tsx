import React from "react";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import SectionCard from "../../SectionCard";

interface IWindowsEnrollmentCardProps {
  viewDetails: () => void;
}

const WindowsEnrollmentCard = ({
  viewDetails,
}: IWindowsEnrollmentCardProps) => {
  return (
    <SectionCard
      header="Windows enrollment"
      cta={
        <Button onClick={viewDetails} variant="inverse" iconStroke>
          Details <Icon name="chevron-right" color="ui-fleet-black-75" />
        </Button>
      }
    >
      To use automatic enrollment for Windows hosts and Windows Autopilot you
      need to connect Fleet to Azure AD first.
    </SectionCard>
  );
};

export default WindowsEnrollmentCard;
