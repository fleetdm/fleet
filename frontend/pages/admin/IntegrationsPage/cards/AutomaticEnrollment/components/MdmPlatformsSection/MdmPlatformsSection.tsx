import React, { useContext } from "react";
import { InjectedRouter } from "react-router";
import PATHS from "router/paths";
import { AppContext } from "context/app";
import SectionHeader from "components/SectionHeader";

import WindowsAutomaticEnrollmentCard from "./components/WindowsAutomaticEnrollmentCard";
import AppleAutomaticEnrollmentCard from "./components/AppleAutomaticEnrollmentCard";

const baseClass = "mdm-platforms-section";

interface IMdmPlatformsSectionProps {
  router: InjectedRouter;
}

const MdmPlatformsSection = ({ router }: IMdmPlatformsSectionProps) => {
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
    <div className={baseClass}>
      <SectionHeader title="Apple Business Manager" />
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
    </div>
  );
};

export default MdmPlatformsSection;
