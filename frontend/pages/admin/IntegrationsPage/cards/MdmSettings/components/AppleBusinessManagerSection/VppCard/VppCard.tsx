import React from "react";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

import SectionCard from "../../SectionCard";

const baseClass = "vpp-card";

interface IVppCardProps {
  isAppleMdmOn: boolean;
  isVppOn: boolean;
  viewDetails: () => void;
}

const VppCard = ({ isAppleMdmOn, isVppOn, viewDetails }: IVppCardProps) => {
  const AppleMdmDisabledCard = (
    <SectionCard header="Volume Purchasing Program (VPP)">
      To enable Volume Purchasing Program (VPP), first turn on Apple (macOS,
      iOS, iPadOS) MDM.
    </SectionCard>
  );

  const VppOnCard = (
    <SectionCard
      iconName="success"
      cta={
        <Button onClick={viewDetails} variant="text-icon">
          <Icon name="pencil" />
          Edit
        </Button>
      }
    >
      Volume Purchasing Program (VPP) is enabled.
    </SectionCard>
  );

  const VppOffCard = (
    <SectionCard
      header="Volume Purchasing Program (VPP)"
      cta={
        <Button
          className={`${baseClass}__add-vpp-button`}
          onClick={viewDetails}
        >
          Add VPP
        </Button>
      }
    >
      Add a VPP connection to install Apple App Store apps purchased through
      Apple Business Manager.
    </SectionCard>
  );

  if (!isAppleMdmOn) {
    return AppleMdmDisabledCard;
  }

  return isVppOn ? VppOnCard : VppOffCard;
};

export default VppCard;
