import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import SettingsSection from "pages/admin/components/SettingsSection";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SectionCard from "../SectionCard";

const baseClass = "vpp-section";

interface IVppCardProps {
  isAppleMdmOn: boolean;
  isVppOn: boolean;
  router: InjectedRouter;
}

const VppCard = ({ isAppleMdmOn, isVppOn, router }: IVppCardProps) => {
  const navigateToVppSetup = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_VPP_SETUP);
  };

  const appleMdmDisabledCard = (
    <SectionCard header="Volume Purchasing Program (VPP)">
      To enable Volume Purchasing Program (VPP), first turn on Apple (macOS,
      iOS, iPadOS) MDM.
    </SectionCard>
  );

  const isVppOnCard = (
    <SectionCard
      iconName="success"
      cta={
        <Button onClick={navigateToVppSetup} variant="text-icon">
          <Icon name="pencil" />
          Edit
        </Button>
      }
    >
      Volume Purchasing Program (VPP) is enabled.
    </SectionCard>
  );

  const isVppOffCard = (
    <SectionCard
      header="Volume Purchasing Program (VPP)"
      cta={
        <Button
          className={`${baseClass}__add-vpp-button`}
          onClick={navigateToVppSetup}
          variant="brand"
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
    return appleMdmDisabledCard;
  }

  return isVppOn ? isVppOnCard : isVppOffCard;
};

interface IVppSectionProps {
  router: InjectedRouter;
  isVppOn: boolean;
  isPremiumTier: boolean;
}

const VppSection = ({ router, isVppOn, isPremiumTier }: IVppSectionProps) => {
  const { config } = useContext(AppContext);

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage alignment="left" />;
    }

    return (
      <VppCard
        isAppleMdmOn={!!config?.mdm.enabled_and_configured}
        isVppOn={isVppOn}
        router={router}
      />
    );
  };

  return (
    <SettingsSection
      title="Volume Purchasing Program (VPP)"
      className={baseClass}
    >
      <>{renderContent()}</>
    </SettingsSection>
  );
};

export default VppSection;
