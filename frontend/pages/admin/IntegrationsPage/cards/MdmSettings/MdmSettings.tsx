import { AxiosError } from "axios";
import React from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { IConfig } from "interfaces/config";
import { IMdmApple } from "interfaces/mdm";
import mdmAPI, { IEulaMetadataResponse } from "services/entities/mdm";
import mdmAppleAPI, {
  IGetVppTokensResponse,
} from "services/entities/mdm_apple";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import AndroidZeroTouchSection from "./components/AndroidZeroTouchSection";
import AppleBusinessManagerSection from "./components/AppleBusinessManagerSection";
import EndUserMigrationSection from "./components/EndUserMigrationSection";
import EulaSection from "./components/EulaSection";
import MdmSettingsSection from "./components/MdmSettingsSection";
import MicrosoftEntraSection from "./components/MicrosoftEntraSection";

const baseClass = "mdm-settings";

interface IMdmSettingsProps {
  router: InjectedRouter;
  appConfig?: IConfig;
  isPremiumTier?: boolean;
}

const MdmSettings = ({
  router,
  appConfig,
  isPremiumTier = false,
}: IMdmSettingsProps) => {
  const isMdmEnabled = !!appConfig?.mdm.enabled_and_configured;
  const isWindowsMdmEnabled = !!appConfig?.mdm.windows_enabled_and_configured;
  // The macOS EULA is shown by Apple's automated enrollment, and end user
  // migration relies on it too, so both need Apple MDM and Apple Business
  // Manager.
  const isAppleBusinessSetUp =
    isMdmEnabled && !!appConfig?.mdm.apple_bm_enabled_and_configured;

  // Currently the status of this API call is what determines various UI states on
  // this page. Because of this we will not render any of this components UI until this API
  // call has completed.
  const {
    data: APNSInfo,
    isLoading: isLoadingAPNSInfo,
    isError: isAPNSInfoError,
    error: errorAPNSInfo,
  } = useQuery<IMdmApple, AxiosError, IMdmApple>(
    ["appleAPNInfo", { isMdmEnabled }],
    () => mdmAppleAPI.getAppleAPNInfo(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: (tries, error) => error.status !== 404 && tries <= 3,
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
    ["vppInfo", { isMdmEnabled }],
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
    ["eula-metadata", { isMdmEnabled }],
    () => mdmAPI.getEULAMetadata(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      enabled: isPremiumTier && isMdmEnabled,
    }
  );

  const {
    data: windowsEulaMetadata,
    isLoading: isLoadingWindowsEula,
    isError: isWindowsEulaError,
    error: windowsEulaError,
    refetch: refetchWindowsEulaMetadata,
  } = useQuery<IEulaMetadataResponse, AxiosError>(
    ["windows-eula-metadata", { isWindowsMdmEnabled }],
    () => mdmAPI.getWindowsEULAMetadata(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      enabled: isPremiumTier && isWindowsMdmEnabled,
    }
  );

  // we use this to determine if we any of the request are still in progress
  // and should show a spinner.
  const isLoading =
    isLoadingAPNSInfo || isLoadingVpp || isLoadingEula || isLoadingWindowsEula;

  const noVppTokenUploaded = !vppData || !vppData.vpp_tokens.length;
  const hasVppError = isVppError && !noVppTokenUploaded;

  // We are relying on the API to give us a 404 to
  // tell use the user has not uploaded a eula.
  const noEulaUploaded = eulaError && eulaError.status === 404;
  const hasEulaError = isEulaError && !noEulaUploaded;
  const noWindowsEulaUploaded =
    windowsEulaError && windowsEulaError.status === 404;
  const hasWindowsEulaError = isWindowsEulaError && !noWindowsEulaUploaded;

  // we use this to determine if there was any errors when getting any of the
  // data we depend on to render the page. We will not include the VPP or EULA
  // 404 errors. We only want to show an error if there was a "real" error
  // (e.g.non 404 error).
  const hasError =
    isAPNSInfoError || hasVppError || hasEulaError || hasWindowsEulaError;

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
          <AndroidZeroTouchSection
            router={router}
            isPremiumTier={isPremiumTier}
          />
          <AppleBusinessManagerSection
            router={router}
            isPremiumTier={isPremiumTier}
            isVppOn={!noVppTokenUploaded}
          />
          <MicrosoftEntraSection
            router={router}
            windowsMdmEnabled={!!appConfig?.mdm.windows_enabled_and_configured}
            tenantAdded={!!appConfig?.mdm.windows_entra_tenant_ids?.length}
            isPremiumTier={isPremiumTier}
          />
          {/* react-query keeps the last metadata when a refetch fails, so the
          404 that follows a delete has to hide it. */}
          {isPremiumTier && (isAppleBusinessSetUp || isWindowsMdmEnabled) && (
            <EulaSection
              eulas={{
                darwin: {
                  isAvailable: isAppleBusinessSetUp,
                  metadata: noEulaUploaded ? undefined : eulaMetadata,
                  onChange: refetchEulaMetadata,
                },
                windows: {
                  isAvailable: isWindowsMdmEnabled,
                  metadata: noWindowsEulaUploaded
                    ? undefined
                    : windowsEulaMetadata,
                  onChange: refetchWindowsEulaMetadata,
                },
              }}
            />
          )}
          {isPremiumTier && isAppleBusinessSetUp && (
            <EndUserMigrationSection router={router} />
          )}
        </>
      )}
    </div>
  );
};

export default MdmSettings;
