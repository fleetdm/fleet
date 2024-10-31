import React from "react";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";

import SectionCard from "../../SectionCard";

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
  const appleMdmDisabledCard = (
    <SectionCard header="Automatic enrollment for Apple (macOS, iOS, iPadOS) hosts.">
      To enable automatic enrollment for macOS, iOS, and iPadOS hosts, first
      turn on Apple MDM.
    </SectionCard>
  );

  const isAbmConfiguredCard = (
    <SectionCard
      iconName="success"
      cta={
        <Button onClick={viewDetails} variant="text-icon">
          <Icon name="pencil" />
          Edit
        </Button>
      }
    >
      Automatic enrollment for Apple (macOS, iOS, iPadOS) is enabled.
    </SectionCard>
  );

  const isAbmNotConfiguredCard = (
    <SectionCard
      header="Automatic enrollment for Apple (macOS, iOS, iPadOS) hosts."
      cta={
        <Button
          className="add-abm-button"
          onClick={viewDetails}
          variant="brand"
        >
          Add ABM
        </Button>
      }
    >
      Add an Apple Business Manager (ABM) connection to automatically enroll
      newly purchased Apple hosts when they&apos;re first unboxed and set up by
      your end users.
    </SectionCard>
  );

  if (!isAppleMdmOn) {
    return appleMdmDisabledCard;
  }

  return configured ? isAbmConfiguredCard : isAbmNotConfiguredCard;
};

export default AppleAutomaticEnrollmentCard;
