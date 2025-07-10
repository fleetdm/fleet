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
  const AppleMdmDisabledCard = (
    <SectionCard header="Automatic enrollment for Apple (macOS, iOS, iPadOS) hosts.">
      To enable automatic enrollment for macOS, iOS, and iPadOS hosts, first
      turn on Apple MDM.
    </SectionCard>
  );

  const AbmConfiguredCard = (
    <SectionCard
      iconName="success"
      cta={
        <Button onClick={viewDetails} variant="text-icon">
          <Icon name="pencil" />
          Edit
        </Button>
      }
    >
      Company-owned (ADE) and personal (BYOD) enrollment for Apple (macOS, iOS,
      iPadOS) is enabled.
    </SectionCard>
  );

  const AbmNotConfiguredCard = (
    <SectionCard
      header="Apple (macOS, iOS, iPadOS) company-owned and personal hosts enrollment"
      cta={
        <Button className="add-abm-button" onClick={viewDetails}>
          Add ABM
        </Button>
      }
    >
      Company-owned Apple hosts will enroll with Automated Device Enrollment
      (ADE) when they&apos;re first unboxed. Personal (BYOD) hosts will enroll
      when end users sign in with Managed Apple Account.
    </SectionCard>
  );

  if (!isAppleMdmOn) {
    return AppleMdmDisabledCard;
  }

  return configured ? AbmConfiguredCard : AbmNotConfiguredCard;
};

export default AppleAutomaticEnrollmentCard;
