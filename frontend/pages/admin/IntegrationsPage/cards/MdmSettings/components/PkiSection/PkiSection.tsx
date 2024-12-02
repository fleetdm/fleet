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

const baseClass = "pki-section";

interface IPkiCardProps {
  isAppleMdmOn: boolean;
  isPkiOn: boolean;
  router: InjectedRouter;
}

export const PKI_TIP_CONTENT = <>Fleet currently supports DigiCert as a PKI.</>;

const DIGICERT_PKI_ADDED_MESSAGE = "DigiCert added as your PKI."; // TODO: confirm this message

const PkiCard = ({ isAppleMdmOn, isPkiOn, router }: IPkiCardProps) => {
  const navigateToPkiSetup = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_PKI);
  };

  const appleMdmDisabledCard = (
    <SectionCard className={baseClass} header="Public key infrastructure (PKI)">
      <p>
        To help your end users connect to Wi-Fi by adding your{" "}
        <TooltipWrapper tipContent={PKI_TIP_CONTENT}>PKI</TooltipWrapper>, first
        turn on Apple (macOS, iOS, iPadOS) MDM.
      </p>
    </SectionCard>
  );

  const isPkiOnCard = (
    <SectionCard
      className={baseClass}
      iconName="success"
      // cta={
      //   <Button onClick={navigateToPkiSetup} variant="text-icon">
      //     <Icon name="pencil" />
      //     Edit
      //   </Button>
      // }
      cta={
        <Button
          className="windows-details-button"
          onClick={navigateToPkiSetup}
          variant="text-icon"
        >
          Details <Icon name="chevron-right" color="core-fleet-blue" />
        </Button>
      }
    >
      {DIGICERT_PKI_ADDED_MESSAGE}
    </SectionCard>
  );

  const isPkiOffCard = (
    <SectionCard
      className={baseClass}
      header="Public key infrastructure (PKI)"
      cta={
        <Button
          className={`${baseClass}__add-scep-button`}
          onClick={navigateToPkiSetup}
          variant="brand"
        >
          Add PKI
        </Button>
      }
    >
      <div>
        To help your end users connect to Wi-Fi, you can add your{" "}
        <TooltipWrapper tipContent={PKI_TIP_CONTENT}>PKI</TooltipWrapper>.
      </div>
    </SectionCard>
  );

  if (!isAppleMdmOn) {
    return appleMdmDisabledCard;
  }

  return isPkiOn ? isPkiOnCard : isPkiOffCard;
};

interface IPkiSectionProps {
  router: InjectedRouter;
  isPkiOn: boolean;
  isPremiumTier: boolean;
}

const PkiSection = ({ router, isPkiOn, isPremiumTier }: IPkiSectionProps) => {
  const { config } = useContext(AppContext);

  return (
    <SettingsSection
      title="Public key infrastructure (PKI)"
      className={baseClass}
    >
      {!isPremiumTier ? (
        <PremiumFeatureMessage alignment="left" />
      ) : (
        <PkiCard
          isAppleMdmOn={!!config?.mdm.enabled_and_configured}
          isPkiOn={isPkiOn}
          router={router}
        />
      )}
    </SettingsSection>
  );
};

export default PkiSection;
