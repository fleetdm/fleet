import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";

import Button from "components/buttons/Button";
import Icon from "components/Icon";

import SettingsSection from "pages/admin/components/SettingsSection";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SectionCard from "../SectionCard";

const baseClass = "certificates-section";

interface IScepCardProps {
  isAppleMdmOn: boolean;
  isScepOn: boolean;
  router: InjectedRouter;
}

const ScepCard = ({ isAppleMdmOn, isScepOn, router }: IScepCardProps) => {
  const navigateToScepSetup = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_SCEP);
  };

  const appleMdmDiabledCard = (
    <SectionCard header="Simple Certificate Enrollment Protocol (SCEP)">
      <p>
        To enable Fleet to get SCEP certificates from your custom SCEP server
        and install them on macOS hosts, first turn on Apple (macOS, iOS,
        iPadOS) MDM.
      </p>
      <p>
        Fleet currently supports Microsoft&apos;s Network Device Enrollment
        Service (NDES) as a custom SCEP server.
      </p>
    </SectionCard>
  );

  const isScepOnCard = (
    <SectionCard
      iconName="success"
      cta={
        <Button onClick={navigateToScepSetup} variant="text-icon">
          <Icon name="pencil" />
          Edit
        </Button>
      }
    >
      TODO: Need Figma design for this
    </SectionCard>
  );

  const isScepOffCard = (
    <SectionCard
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
      <p>
        Add a SCEP connection to enable Fleet to get SCEP certificates from your
        custom SCEP server and install them on macOS hosts.{" "}
      </p>
      <p>
        Fleet currently supports Microsoft&apos;s Network Device Enrollment
        Service (NDES) as a custom SCEP server.
      </p>
    </SectionCard>
  );

  if (!isAppleMdmOn) {
    return appleMdmDiabledCard;
  }

  return !isScepOn ? isScepOnCard : isScepOffCard;
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
    <SettingsSection title="Certificates" className={baseClass}>
      <>{renderContent()}</>
    </SettingsSection>
  );
};

export default ScepSection;
