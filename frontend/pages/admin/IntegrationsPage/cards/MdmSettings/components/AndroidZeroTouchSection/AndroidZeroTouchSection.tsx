import React, { useContext } from "react";
import { InjectedRouter } from "react-router";

import Button from "components/buttons/Button";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import { AppContext } from "context/app";
import SettingsSection from "pages/admin/components/SettingsSection";
import PATHS from "router/paths";

import SectionCard from "../SectionCard";

const baseClass = "android-zero-touch-section";

interface IAndroidZeroTouchSectionProps {
  router: InjectedRouter;
  isPremiumTier: boolean;
}

const AndroidZeroTouchSection = ({
  router,
  isPremiumTier,
}: IAndroidZeroTouchSectionProps) => {
  const { isAndroidMdmEnabledAndConfigured } = useContext(AppContext);

  const navigateToSetup = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_ANDROID_ZERO_TOUCH);
  };

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }

    if (!isAndroidMdmEnabledAndConfigured) {
      return (
        <SectionCard header="Android enrollment">
          To enable end users to enroll to Fleet via Android zero-touch, first
          turn on Android MDM.
        </SectionCard>
      );
    }

    return (
      <SectionCard
        cta={
          <Button
            onClick={navigateToSetup}
            variant="subdued"
            icon="chevron-right"
            iconPosition="right"
          >
            Setup
          </Button>
        }
      >
        To automatically enroll company-owned Android hosts when they&apos;re
        first unboxed, connect Fleet to Android zero-touch.
      </SectionCard>
    );
  };

  return (
    <SettingsSection title="Android zero-touch" className={baseClass}>
      {renderContent()}
    </SettingsSection>
  );
};

export default AndroidZeroTouchSection;
