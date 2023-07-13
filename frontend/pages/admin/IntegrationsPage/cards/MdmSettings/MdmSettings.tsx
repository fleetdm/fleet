import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";

import mdmAppleAPI from "services/entities/mdm_apple";
import { IMdmApple } from "interfaces/mdm";

import PATHS from "router/paths";

import Spinner from "components/Spinner";
import DataError from "components/DataError/DataError";
import EndUserMigrationSection from "./components/EndUserMigrationSection/EndUserMigrationSection";
import WindowsMdmSection from "./components/WindowsMdmSection/WindowsMdmSection";
import MacOSMdmCard from "./components/MacOSMdmCard/MacOSMdmCard";

const baseClass = "mdm-settings";

interface IMdmSettingsProps {
  router: InjectedRouter;
}

const MdmSettings = ({ router }: IMdmSettingsProps) => {
  const { isPremiumTier, config } = useContext(AppContext);

  // Currently the status of this API call is what determines various UI states on
  // this page. Because of this we will not render any of this components UI until this API
  // call has completed.
  const {
    data: appleAPNInfo,
    isLoading: isLoadingMdmApple,
    error: errorMdmApple,
  } = useQuery<IMdmApple, AxiosError, IMdmApple>(
    ["appleAPNInfo"],
    () => mdmAppleAPI.getAppleAPNInfo(),
    {
      retry: (tries, error) => error.status !== 404 && tries <= 3,
      enabled: config?.mdm.enabled_and_configured,
      staleTime: 5000,
    }
  );

  const navigateToMacOSMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_MAC);
  };

  const navigateToWindowsMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_WINDOWS);
  };

  const renderMacOSMdmCard = () => {
    // The API returns a 404 error if APNS is not configured yet. If there is any
    // other error we will show the DataError component.
    const showMdmAppleError = errorMdmApple && errorMdmApple.status !== 404;

    if (showMdmAppleError) {
      return <DataError />;
    }

    return (
      <MacOSMdmCard
        isEnabled={appleAPNInfo !== undefined}
        turnOnMacOSMdm={navigateToMacOSMdm}
        viewDetails={navigateToMacOSMdm}
      />
    );
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section ${baseClass}__mdm-section`}>
        <h2>Mobile device management (MDM)</h2>
        {isLoadingMdmApple ? (
          <Spinner />
        ) : (
          <>
            {renderMacOSMdmCard()}
            {/* TODO: remove conditional rendering when windows MDM is released. */}
            {config?.mdm_enabled && (
              <WindowsMdmSection
                turnOnWindowsMdm={navigateToWindowsMdm}
                editWindowsMdm={navigateToWindowsMdm}
              />
            )}
          </>
        )}
      </div>
      {isPremiumTier && appleAPNInfo && (
        <div className={`${baseClass}__section`}>
          <EndUserMigrationSection router={router} />
        </div>
      )}
    </div>
  );
};

export default MdmSettings;
