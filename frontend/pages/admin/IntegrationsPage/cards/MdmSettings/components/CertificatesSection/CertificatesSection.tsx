import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";

import Card from "components/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import SettingsSection from "pages/admin/components/SettingsSection";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

const baseClass = "certificates-section";

interface IScepCardProps {
  isAppleMdmOn: boolean;
  isScepOn: boolean;
  router: InjectedRouter;
}

const ScepCard = ({ isAppleMdmOn, isScepOn, router }: IScepCardProps) => {
  const navigateToScepSetup = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_SCEP_SETUP);
  };

  const appleMdmDiabledContent = (
    <div className={`${baseClass}__mdm-disabled-content`}>
      <div>
        <h3>Simple Certificate Enrollment Protocol (SCEP)</h3>
        <p>
          To enable Fleet to get SCEP certificates from your custom SCEP server
          and install them on macOS hosts, first turn on Apple (macOS, iOS,
          iPadOS) MDM.
        </p>
        <p>
          Fleet currently supports Microsoft&apos;s Network Device Enrollment
          Service (NDES) as a custom SCEP server.
        </p>
      </div>
    </div>
  );

  const isScepOnContent = (
    <div className={`${baseClass}__scep-on-content`}>
      <p>
        <span>
          <Icon name="success" />
          Volume Purchasing Program (VPP) is enabled. TODO THIS IS NOT ON FIGMA
          YET?
        </span>
      </p>
      <Button onClick={navigateToScepSetup} variant="text-icon">
        <Icon name="pencil" />
        Edit
      </Button>
    </div>
  );

  const isScepOffContent = (
    <div className={`${baseClass}__scep-off-content`}>
      <div>
        <h3>Simple Certificate Enrollment Protocol (SCEP)</h3>
        <p>
          Add a SCEP connection to enable Fleet to get SCEP certificates from
          your custom SCEP server and install them on macOS hosts.{" "}
        </p>
        <p>
          Fleet currently supports Microsoft&apos;s Network Device Enrollment
          Service (NDES) as a custom SCEP server.
        </p>
      </div>
      <Button
        className={`${baseClass}__add-scep-button`}
        onClick={navigateToScepSetup}
        variant="brand"
      >
        Add SCEP
      </Button>
    </div>
  );

  const renderCardContent = () => {
    if (!isAppleMdmOn) {
      return appleMdmDiabledContent;
    }

    return isScepOn ? isScepOnContent : isScepOffContent;
  };

  return (
    <Card className={`${baseClass}__card`} color="gray">
      {renderCardContent()}
    </Card>
  );
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
