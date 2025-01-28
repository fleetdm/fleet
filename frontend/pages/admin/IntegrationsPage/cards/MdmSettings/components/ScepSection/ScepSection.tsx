import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

import SettingsSection from "pages/admin/components/SettingsSection";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import TooltipWrapper from "components/TooltipWrapper";

import SectionCard from "../SectionCard";

const baseClass = "scep-section";

interface IScepCardProps {
  isAppleMdmOn: boolean;
  isScepOn: boolean;
  router: InjectedRouter;
}

export const SCEP_SERVER_TIP_CONTENT = (
  <>
    Fleet currently supports Microsoft&apos;s Network Device
    <br />
    Enrollment Service (NDES) as a SCEP server.
  </>
);

const ScepCard = ({ isAppleMdmOn, isScepOn, router }: IScepCardProps) => {
  const navigateToScepSetup = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_SCEP);
  };

  const appleMdmDisabledCard = (
    <SectionCard
      className={baseClass}
      header="Simple Certificate Enrollment Protocol (SCEP)"
    >
      <p>
        To help your end users connect to Wi-Fi by adding your{" "}
        <TooltipWrapper tipContent={SCEP_SERVER_TIP_CONTENT}>
          SCEP server
        </TooltipWrapper>
        , first turn on Apple (macOS, iOS, iPadOS) MDM.
      </p>
    </SectionCard>
  );

  const isScepOnCard = (
    <SectionCard
      className={baseClass}
      iconName="success"
      cta={
        <Button onClick={navigateToScepSetup} variant="text-icon">
          <Icon name="pencil" />
          Edit
        </Button>
      }
    >
      Microsoft&apos;s Network Device Enrollment Service (NDES) added as your
      SCEP server. Your end users can connect to Wi-Fi.
    </SectionCard>
  );

  const isScepOffCard = (
    <SectionCard
      className={baseClass}
      header="Simple Certificate Enrollment Protocol (SCEP)"
      cta={
        <Button
          className={`${baseClass}__add-scep-button`}
          onClick={navigateToScepSetup}
          variant="brand"
        >
          Add SCEP
        </Button>
      }
    >
      <div>
        To help your end users connect to Wi-Fi, you can add your{" "}
        <TooltipWrapper tipContent={SCEP_SERVER_TIP_CONTENT}>
          SCEP server
        </TooltipWrapper>
        .
      </div>
    </SectionCard>
  );

  if (!isAppleMdmOn) {
    return appleMdmDisabledCard;
  }

  return isScepOn ? isScepOnCard : isScepOffCard;
};

interface IScepSectionProps {
  router: InjectedRouter;
  isScepOn: boolean;
  isPremiumTier: boolean;
}

const ScepSection = ({
  router,
  isScepOn,
  isPremiumTier,
}: IScepSectionProps) => {
  const { config } = useContext(AppContext);

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage alignment="left" />;
    }

    return (
      <ScepCard
        isAppleMdmOn={!!config?.mdm.enabled_and_configured}
        isScepOn={isScepOn}
        router={router}
      />
    );
  };

  return (
    <SettingsSection
      title="Simple Certificate Enrollment Protocol (SCEP)"
      className={baseClass}
    >
      <>{renderContent()}</>
    </SettingsSection>
  );
};

export default ScepSection;
