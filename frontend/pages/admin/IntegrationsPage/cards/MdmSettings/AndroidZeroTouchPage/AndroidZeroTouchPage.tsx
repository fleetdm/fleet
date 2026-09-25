import { AxiosError } from "axios";
import React, { useContext } from "react";
import { useQuery } from "react-query";

import BackButton from "components/BackButton";
import CopyButton from "components/buttons/CopyButton";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import MainContent from "components/MainContent";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import { AppContext } from "context/app";
import PATHS from "router/paths";
import mdmAndroidAPI, {
  IGetZeroTouchConfigurationResponse,
} from "services/entities/mdm_android";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

const baseClass = "android-zero-touch-page";

const AndroidZeroTouchPage = () => {
  const {
    currentUser,
    isPremiumTier,
    isAndroidMdmEnabledAndConfigured,
  } = useContext(AppContext);

  // `isPremiumTier` is undefined until the config request resolves, and the
  // route renders as soon as `currentUser` is set. Without this the paywall
  // flashes on Premium and the premium-only request fires on Free.
  const isTierKnown = isPremiumTier !== undefined;

  const { data: zeroTouchConfig, isLoading, isError } = useQuery<
    IGetZeroTouchConfigurationResponse,
    AxiosError
  >(
    // Scoped to the user: the query client is module-scoped and survives SPA
    // logout, and the DPC extras embed a reusable enrollment token.
    ["android-zero-touch-configuration", currentUser?.id],
    () => mdmAndroidAPI.getZeroTouchConfiguration(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled:
        !!isPremiumTier && !!isAndroidMdmEnabledAndConfigured && !!currentUser,
    }
  );

  const hasConfig = !isError && !!zeroTouchConfig;

  const renderCodeBlock = () => {
    if (isLoading) {
      return (
        <div
          className={`${baseClass}__dpc-extras-code ${baseClass}__dpc-extras-code--loading`}
        >
          <Spinner />
        </div>
      );
    }

    if (!hasConfig) {
      return <DataError />;
    }

    return (
      <pre className={`${baseClass}__dpc-extras-code`}>
        <code>{zeroTouchConfig.dpc_extras}</code>
      </pre>
    );
  };

  const renderContent = () => {
    if (!isTierKnown) {
      return <Spinner />;
    }

    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }

    if (!isAndroidMdmEnabledAndConfigured) {
      return (
        <p className={`${baseClass}__prerequisite`}>
          To enable end users to enroll to Fleet via Android zero-touch, first
          turn on Android MDM.
        </p>
      );
    }

    return (
      <>
        <p className={`${baseClass}__description`}>
          To connect Fleet to Android zero-touch, go to the{" "}
          <CustomLink
            url="https://fleetdm.com/learn-more-about/android-zero-touch-portal"
            text="Android zero-touch portal"
            newTab
          />
        </p>
        <p className={`${baseClass}__enrollment-info`}>
          Android hosts will automatically enroll to the <b>Unassigned</b>{" "}
          fleet. Changing fleets is coming soon.
        </p>
        <div className={`${baseClass}__dpc-extras`}>
          <div className={`${baseClass}__dpc-extras-header`}>
            <span className={`${baseClass}__dpc-extras-label`}>DPC extras</span>
            {hasConfig && (
              <CopyButton
                copyText={zeroTouchConfig.dpc_extras}
                variant="secondary"
              />
            )}
          </div>
          {renderCodeBlock()}
        </div>
        {hasConfig && (
          <p className={`${baseClass}__instructions`}>
            Select <b>Add configuration</b>, pick <b>Android Device Policy</b>{" "}
            as your <b>EMM DPC</b>, and paste this JSON into <b>DPC extras</b>.
          </p>
        )}
      </>
    );
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__header-links`}>
        <BackButton
          text="Back to MDM"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-mdm`}
        />
      </div>
      <h1>Android zero-touch</h1>
      {renderContent()}
    </MainContent>
  );
};

export default AndroidZeroTouchPage;
