import React, { useContext } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { InjectedRouter } from "react-router";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { AppContext } from "context/app";
import { IMdmApple } from "interfaces/mdm";
import mdmAppleAPI, {
  IGetVppTokensResponse,
} from "services/entities/mdm_apple";
import mdmAPI, { IEulaMetadataResponse } from "services/entities/mdm";

import MdmSettingsSection from "./components/MdmSettingsSection";
import AutomaticEnrollmentSection from "./components/AutomaticEnrollmentSection";
import VppSection from "./components/VppSection";
import IdpSection from "./components/IdpSection";
import EulaSection from "./components/EulaSection";
import EndUserMigrationSection from "./components/EndUserMigrationSection";
import ScepSection from "./components/ScepSection/ScepSection";
import PkiSection from "./components/PkiSection/PkiSection";

const baseClass = "mdm-settings";

interface IMdmSettingsProps {
  router: InjectedRouter;
}

const MdmSettings = ({ router }: IMdmSettingsProps) => {
  const { isPremiumTier, config } = useContext(AppContext);

  const isMdmEnabled = !!config?.mdm.enabled_and_configured;

  // Currently the status of this API call is what determines various UI states on
  // this page. Because of this we will not render any of this components UI until this API
  // call has completed.
  const {
    data: APNSInfo,
    isLoading: isLoadingAPNSInfo,
    isError: isAPNSInfoError,
    error: errorAPNSInfo,
  } = useQuery<IMdmApple, AxiosError, IMdmApple>(
    ["appleAPNInfo"],
    () => mdmAppleAPI.getAppleAPNInfo(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: (tries, error) =>
        error.status !== 404 && error.status !== 400 && tries <= 3,
      // TODO: There is a potential race condition here immediately after MDM is turned off. This
      // component gets remounted and stale config data is used to determine it this API call is
      // enabled, resulting in a 400 response. The race really should  be fixed higher up the chain where
      // we're fetching and setting the config, but for now we'll just assume that any 400 response
      // means that MDM is not enabled and we'll show the "Turn on MDM" button.
      staleTime: 5000,
      enabled: isMdmEnabled,
    }
  );

  // get the vpp info
  const {
    data: vppData,
    isLoading: isLoadingVpp,
    isError: isVppError,
  } = useQuery<IGetVppTokensResponse, AxiosError>(
    "vppInfo",
    () => mdmAppleAPI.getVppTokens(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      enabled: isPremiumTier && isMdmEnabled,
    }
  );

  // get the eula metadata
  const {
    data: eulaMetadata,
    isLoading: isLoadingEula,
    isError: isEulaError,
    error: eulaError,
    refetch: refetchEulaMetadata,
  } = useQuery<IEulaMetadataResponse, AxiosError>(
    ["eula-metadata"],
    () => mdmAPI.getEULAMetadata(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      enabled: isPremiumTier && isMdmEnabled,
    }
  );

  // we use this to determine if we any of the request are still in progress
  // and should show a spinner.
  const isLoading = isLoadingAPNSInfo || isLoadingVpp || isLoadingEula;

  const noVppTokenUploaded = !vppData || !vppData.vpp_tokens.length;
  const hasVppError = isVppError && !noVppTokenUploaded;

  const noScepCredentials = !config?.integrations.ndes_scep_proxy;

  const noPki = !config?.integrations.digicert_pki?.length;

  // We are relying on the API to give us a 404 to
  // tell use the user has not uploaded a eula.
  const noEulaUploaded = eulaError && eulaError.status === 404;
  const hasEulaError = isEulaError && !noEulaUploaded;

  // we use this to determine if there was any errors when getting any of the
  // data we depend on to render the page. We will not include the VPP or EULA
  // 404 errors. We only want to show an error if there was a "real" error
  // (e.g.non 404 error).
  const hasError = isAPNSInfoError || hasVppError || hasEulaError;

  // we use this to determine if we have all the data we need to render the UI.
  // Notice that we do not need VPP or EULA data to render this page.
  const hasAllData = !isMdmEnabled || !!APNSInfo;

  return (
    <div className={baseClass}>
      {/* The MDM settings section component handles showing the pages overall
       * loading and error states */}
      <MdmSettingsSection
        isLoading={isLoading}
        isError={hasError}
        appleAPNSInfo={APNSInfo}
        appleAPNSError={errorAPNSInfo}
        router={router}
      />
      {!isLoading && !hasError && hasAllData && (
        <>
          <AutomaticEnrollmentSection
            router={router}
            isPremiumTier={!!isPremiumTier}
          />
          <VppSection
            router={router}
            isVppOn={!noVppTokenUploaded}
            isPremiumTier={!!isPremiumTier}
          />
          <PkiSection
            router={router}
            isPremiumTier={!!isPremiumTier}
            isPkiOn={!noPki}
          />
          <ScepSection
            router={router}
            isScepOn={!noScepCredentials}
            isPremiumTier={!!isPremiumTier}
          />
          {isPremiumTier && !!config?.mdm.apple_bm_enabled_and_configured && (
            <>
              <IdpSection />
              <EulaSection
                eulaMetadata={eulaMetadata}
                isEulaUploaded={!noEulaUploaded}
                onUpload={refetchEulaMetadata}
                onDelete={refetchEulaMetadata}
              />
              <EndUserMigrationSection router={router} />
            </>
          )}
        </>
      )}
    </div>
  );
};

export default MdmSettings;
