import React, { useContext } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

import { AppContext } from "context/app";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SettingsSection from "pages/admin/components/SettingsSection";

import AppleAutomaticEnrollmentCard from "./AppleAutomaticEnrollmentCard";
import VppCard from "./VppCard/VppCard";

const baseClass = "apple-business-manager-section";

interface IAutomaticEnrollmentSectionProps {
  router: InjectedRouter;
  isPremiumTier: boolean;
  isVppOn: boolean;
}

const AppleBusinessManagerSection = ({
  router,
  isPremiumTier,
  isVppOn,
}: IAutomaticEnrollmentSectionProps) => {
  const { config } = useContext(AppContext);

  const navigateToAppleAutomaticEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_APPLE_BUSINESS_MANAGER);
  };

  const navigateToVppSetup = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_VPP_SETUP);
  };

  return (
    <SettingsSection title="Apple Business Manger (ABM)" className={baseClass}>
      {!isPremiumTier ? (
        <PremiumFeatureMessage alignment="left" />
      ) : (
        <div className={`${baseClass}__content`}>
          <AppleAutomaticEnrollmentCard
            viewDetails={navigateToAppleAutomaticEnrollment}
            isAppleMdmOn={!!config?.mdm.enabled_and_configured}
            configured={!!config?.mdm.apple_bm_enabled_and_configured}
          />
          <VppCard
            viewDetails={navigateToVppSetup}
            isAppleMdmOn={!!config?.mdm.enabled_and_configured}
            isVppOn={isVppOn}
          />
        </div>
      )}
    </SettingsSection>
  );
};

export default AppleBusinessManagerSection;
