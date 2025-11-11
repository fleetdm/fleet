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
      header="Windows enrollment"
      cta={
        <Button onClick={viewDetails} variant="inverse" iconStroke>
          Details <Icon name="chevron-right" color="ui-fleet-black-75" />
        </Button>
      }
    >
      To enable end users to enroll to Fleet via Microsoft Entra (e.g.
      Autopilot), you need to connect Fleet to Entra first.
    </SectionCard>
  );
};

export default WindowsAutomaticEnrollmentCard;
