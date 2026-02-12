import React from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SettingsSection from "pages/admin/components/SettingsSection";

import WindowsAutomaticEnrollmentCard from "./WindowsAutomaticEnrollmentCard";

const baseClass = "microsoft-entra-section";

interface IMicrosoftEntraSectionProps {
  router: InjectedRouter;
  windowsMdmEnabled: boolean;
  isPremiumTier: boolean;
}

const MicrosoftEntraSection = ({
  router,
  windowsMdmEnabled,
  isPremiumTier,
}: IMicrosoftEntraSectionProps) => {
  const navigateToWindowsEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS);
  };

  return (
    <SettingsSection title="Microsoft Entra" className={baseClass}>
      {!isPremiumTier ? (
        <PremiumFeatureMessage alignment="left" />
      ) : (
        <div className={`${baseClass}__content`}>
          <WindowsAutomaticEnrollmentCard
            windowsMdmEnabled={windowsMdmEnabled}
            tenantAdded
            viewDetails={navigateToWindowsEnrollment}
          />
        </div>
      )}
    </SettingsSection>
  );
};

export default MicrosoftEntraSection;
