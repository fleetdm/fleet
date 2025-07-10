import React from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SettingsSection from "pages/admin/components/SettingsSection";

import WindowsAutomaticEnrollmentCard from "./WindowsAutomaticEnrollmentCard";

const baseClass = "windows-autopilot-section";

interface IWindowsAutopilotSectionProps {
  router: InjectedRouter;
  isPremiumTier: boolean;
}

const WindowsAutopilotSection = ({
  router,
  isPremiumTier,
}: IWindowsAutopilotSectionProps) => {
  const navigateToWindowsAutomaticEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS);
  };

  return (
    <SettingsSection title="Windows Autopilot" className={baseClass}>
      {!isPremiumTier ? (
        <PremiumFeatureMessage alignment="left" />
      ) : (
        <div className={`${baseClass}__content`}>
          <WindowsAutomaticEnrollmentCard
            viewDetails={navigateToWindowsAutomaticEnrollment}
          />
        </div>
      )}
    </SettingsSection>
  );
};

export default WindowsAutopilotSection;
