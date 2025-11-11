import React from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SettingsSection from "pages/admin/components/SettingsSection";

import WindowsEnrollmentCard from "./WindowsEnrollmentCard";

const baseClass = "microsoft-entra-section";

interface IMicrosoftEntraSectionProps {
  router: InjectedRouter;
  isPremiumTier: boolean;
}

const MicrosoftEntraSection = ({
  router,
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
          <WindowsEnrollmentCard viewDetails={navigateToWindowsEnrollment} />
        </div>
      )}
    </SettingsSection>
  );
};

export default MicrosoftEntraSection;
