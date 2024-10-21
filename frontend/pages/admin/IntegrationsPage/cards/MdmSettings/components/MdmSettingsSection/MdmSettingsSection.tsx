import React from "react";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";

import PATHS from "router/paths";
import { IMdmApple } from "interfaces/mdm";

import Spinner from "components/Spinner";
import DataError from "components/DataError";
import SettingsSection from "pages/admin/components/SettingsSection";

import AppleMdmCard from "./AppleMdmCard";
import WindowsMdmCard from "./WindowsMdmCard";

const baseClass = "mdm-settings-section";

interface IMdmSectionProps {
  isLoading: boolean;
  isError: boolean;
  appleAPNSError: AxiosError | null;
  router: InjectedRouter;
  appleAPNSInfo?: IMdmApple;
}

const MdmSettingsSection = ({
  isLoading,
  isError,
  appleAPNSError,
  router,
  appleAPNSInfo,
}: IMdmSectionProps) => {
  const navigateToAppleMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_APPLE);
  };

  const navigateToWindowsMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_WINDOWS);
  };

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <DataError />;
    }

    return (
      <div className={`${baseClass}__content`}>
        <AppleMdmCard
          appleAPNSInfo={appleAPNSInfo}
          errorData={appleAPNSError}
          turnOnAppleMdm={navigateToAppleMdm}
          viewDetails={navigateToAppleMdm}
        />
        <WindowsMdmCard
          turnOnWindowsMdm={navigateToWindowsMdm}
          editWindowsMdm={navigateToWindowsMdm}
        />
      </div>
    );
  };

  return (
    <SettingsSection
      title="Mobile device management (MDM)"
      className={baseClass}
    >
      {renderContent()}
    </SettingsSection>
  );
};

export default MdmSettingsSection;
