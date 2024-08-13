import React from "react";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";

import PATHS from "router/paths";
import { IMdmApple } from "interfaces/mdm";

import Spinner from "components/Spinner";
import SettingsSection from "pages/admin/components/SettingsSection";

import AppleMdmCard from "./AppleMdmCard";
import WindowsMdmCard from "./WindowsMdmCard";

const baseClass = "mdm-settings-section";

interface IMdmSectionProps {
  appleAPNInfo?: IMdmApple;
  appleAPNError: AxiosError | null;
  isLoading: boolean;
  router: InjectedRouter;
}

const MdmSettingsSection = ({
  router,
  appleAPNInfo,
  appleAPNError,
  isLoading,
}: IMdmSectionProps) => {
  const navigateToAppleMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_APPLE);
  };

  const navigateToWindowsMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_WINDOWS);
  };

  return (
    <SettingsSection
      title="Mobile device management (MDM)"
      className={baseClass}
    >
      {isLoading ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__content`}>
          <AppleMdmCard
            appleAPNInfo={appleAPNInfo}
            errorData={appleAPNError}
            turnOnAppleMdm={navigateToAppleMdm}
            viewDetails={navigateToAppleMdm}
          />
          <WindowsMdmCard
            turnOnWindowsMdm={navigateToWindowsMdm}
            editWindowsMdm={navigateToWindowsMdm}
          />
        </div>
      )}
    </SettingsSection>
  );
};

export default MdmSettingsSection;
