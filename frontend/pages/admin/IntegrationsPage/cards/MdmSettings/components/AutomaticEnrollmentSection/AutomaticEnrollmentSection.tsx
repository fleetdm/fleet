import React, { useContext } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

import { AppContext } from "context/app";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SettingsSection from "pages/admin/components/SettingsSection";

import AppleAutomaticEnrollmentCard from "./AppleAutomaticEnrollmentCard";
import WindowsAutomaticEnrollmentCard from "./WindowsAutomaticEnrollmentCard";

const baseClass = "automatic-enrollment-section";

interface IAutomaticEnrollmentSectionProps {
  router: InjectedRouter;
  isPremiumTier: boolean;
}

const AutomaticEnrollmentSection = ({
  router,
  isPremiumTier,
}: IAutomaticEnrollmentSectionProps) => {
  const { config } = useContext(AppContext);

  const navigateToWindowsAutomaticEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS);
  };

  const navigateToAppleAutomaticEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_APPLE_BUSINESS_MANAGER);
  };

  return (
    <SettingsSection title="Automatic enrollment" className={baseClass}>
      {!isPremiumTier ? (
        <PremiumFeatureMessage alignment="left" />
      ) : (
        <div className={`${baseClass}__content`}>
          <AppleAutomaticEnrollmentCard
            viewDetails={navigateToAppleAutomaticEnrollment}
            isAppleMdmOn={!!config?.mdm.enabled_and_configured}
            configured={!!config?.mdm.apple_bm_enabled_and_configured}
          />
          <WindowsAutomaticEnrollmentCard
            viewDetails={navigateToWindowsAutomaticEnrollment}
          />
        </div>
      )}
    </SettingsSection>
  );
};

export default AutomaticEnrollmentSection;
