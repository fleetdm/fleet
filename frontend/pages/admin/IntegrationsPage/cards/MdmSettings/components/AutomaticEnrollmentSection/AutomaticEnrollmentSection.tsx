import React, { useContext } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";

import { AppContext } from "context/app";

import SettingsSection from "pages/admin/components/SettingsSection";

import AppleAutomaticEnrollmentCard from "./AppleAutomaticEnrollmentCard";
import WindowsAutomaticEnrollmentCard from "./WindowsAutomaticEnrollmentCard";

const baseClass = "automatic-enrollment-section";

interface IAutomaticEnrollmentSectionProps {
  router: InjectedRouter;
}

const AutomaticEnrollmentSection = ({
  router,
}: IAutomaticEnrollmentSectionProps) => {
  const { config } = useContext(AppContext);

  const navigateToWindowsAutomaticEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_WINDOWS);
  };

  const navigateToAppleAutomaticEnrollment = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT_APPLE);
  };

  const navigateToApplePushCertSetup = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  return (
    <SettingsSection title="Automatic Enrollment" className={baseClass}>
      <AppleAutomaticEnrollmentCard
        viewDetails={navigateToAppleAutomaticEnrollment}
        turnOn={
          !config?.mdm.enabled_and_configured
            ? navigateToApplePushCertSetup
            : undefined
        }
        configured={!!config?.mdm.apple_bm_enabled_and_configured}
      />
      <WindowsAutomaticEnrollmentCard
        viewDetails={navigateToWindowsAutomaticEnrollment}
      />
    </SettingsSection>
  );
};

export default AutomaticEnrollmentSection;
