import React, { useContext } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";

import mdmAppleAPI from "services/entities/mdm_apple";
import { IMdmApple } from "interfaces/mdm";

import PATHS from "router/paths";

import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";

import EndUserMigrationSection from "./components/EndUserMigrationSection/EndUserMigrationSection";
import WindowsMdmCard from "./components/WindowsMdmCard/WindowsMdmCard";
import AppleMdmCard from "./components/AppleMdmCard/AppleMdmCard";

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
      retry: (tries, error) =>
        error.status !== 404 && error.status !== 400 && tries <= 3,
      // TODO: There is a potential race condition here immediately after MDM is turned off. This
      // component gets remounted and stale config data is used to determine it this API call is
      // enabled, resulting in a 400 response. The race really should  be fixed higher up the chain where
      // we're fetching and setting the config, but for now we'll just assume that any 400 response
      // means that MDM is not enabled and we'll show the "Turn on MDM" button.
      enabled: !!config?.mdm.enabled_and_configured,
      staleTime: 5000,
    }
  );

  const navigateToAppleMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_APPLE);
  };

  const navigateToWindowsMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_WINDOWS);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader title="Mobile device management (MDM)" />
        {isLoadingMdmApple ? (
          <Spinner />
        ) : (
          <div className={`${baseClass}__section ${baseClass}__mdm-section`}>
            <AppleMdmCard
              appleAPNInfo={appleAPNInfo}
              errorData={errorMdmApple}
              turnOnAppleMdm={navigateToAppleMdm}
              viewDetails={navigateToAppleMdm}
            />
            <WindowsMdmCard
              turnOnWindowsMdm={navigateToWindowsMdm}
              editWindowsMdm={navigateToWindowsMdm}
            />
          </div>
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
