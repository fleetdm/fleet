import React from "react";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";
import SectionCard from "../../SectionCard";

interface IWindowsAutomaticEnrollmentCardProps {
  windowsMdmEnabled: boolean;
  viewDetails: () => void;
}

const WindowsAutomaticEnrollmentCard = ({
  windowsMdmEnabled,
  viewDetails,
}: IWindowsAutomaticEnrollmentCardProps) => {
  const contentText = (
    <>
      To enable end users to enroll to Fleet via Microsoft Entra (e.g.
      Autopilot),{" "}
      {windowsMdmEnabled
        ? "you need to connect Fleet to Entra first."
        : "first turn on Windows MDM."}{" "}
    </>
  );

  return (
    <SectionCard
      header="Windows enrollment"
      cta={
        windowsMdmEnabled ? (
          <Button onClick={viewDetails} variant="inverse" iconStroke>
            Details <Icon name="chevron-right" color="ui-fleet-black-75" />
          </Button>
        ) : undefined
      }
    >
      {contentText}
    </SectionCard>
  );
};

export default WindowsAutomaticEnrollmentCard;
