import { AxiosError } from "axios";
import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import BackButton from "components/BackButton";
import CopyButton from "components/buttons/CopyButton";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import FleetsDropdown from "components/FleetsDropdown";
import InputField from "components/forms/fields/InputField";
import MainContent from "components/MainContent";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import { AppContext } from "context/app";
import {
  APP_CONTEXT_NO_TEAM_ID,
  APP_CONTEXT_NO_TEAM_SUMMARY,
} from "interfaces/team";
import PATHS from "router/paths";
import mdmAndroidAPI, {
  IGetZeroTouchConfigurationResponse,
} from "services/entities/mdm_android";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

const baseClass = "android-zero-touch-page";

const AndroidZeroTouchPage = () => {
  const {
    currentUser,
    availableTeams,
    isPremiumTier,
    isAndroidMdmEnabledAndConfigured,
  } = useContext(AppContext);

  const [selectedFleetId, setSelectedFleetId] = useState(
    APP_CONTEXT_NO_TEAM_ID
  );
  const selectedFleetName =
    availableTeams?.find((fleet) => fleet.id === selectedFleetId)?.name ??
    APP_CONTEXT_NO_TEAM_SUMMARY.name;

  // `isPremiumTier` is undefined until the config request resolves, and the
  // route renders as soon as `currentUser` is set. Without this the paywall
  // flashes on Premium and the premium-only request fires on Free.
  const isTierKnown = isPremiumTier !== undefined;

  const { data: zeroTouchConfig, isLoading, isFetching, isError } = useQuery<
    IGetZeroTouchConfigurationResponse,
    AxiosError
  >(
    // Scoped to the user: the query client is module-scoped and survives SPA
    // logout, and the DPC extras embed a reusable enrollment token.
    ["android-zero-touch-configuration", currentUser?.id, selectedFleetId],
    () => mdmAndroidAPI.getZeroTouchConfiguration(selectedFleetId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled:
        !!isPremiumTier && !!isAndroidMdmEnabledAndConfigured && !!currentUser,
    }
  );

  const hasConfig = !isError && !!zeroTouchConfig;
  const dpcExtrasText = hasConfig
    ? JSON.stringify(zeroTouchConfig, null, 2)
    : "";

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
        <code>{dpcExtrasText}</code>
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
        <div className={`${baseClass}__description`}>
          To connect Fleet to Android zero-touch, go to the{" "}
          <CustomLink
            url="https://fleetdm.com/learn-more-about/android-zero-touch-portal"
            text="Android zero-touch portal"
            newTab
          />
        </div>
        <div className={`${baseClass}__fleet-picker`}>
          <span>
            Pick the fleet that Android hosts will automatically enroll into:
          </span>
          <FleetsDropdown
            asFormField
            currentUserFleets={availableTeams || []}
            selectedFleetId={selectedFleetId}
            includeAllFleets={false}
            includeUnassigned
            isDisabled={isFetching}
            onChange={setSelectedFleetId}
          />
        </div>
        <div className={`${baseClass}__field`}>
          <div className={`${baseClass}__field-header`}>
            <span className={`${baseClass}__field-label`}>Name</span>
            <CopyButton
              copyText={selectedFleetName}
              variant="secondary"
              ariaLabel="Copy name"
              disabled={isFetching}
            />
          </div>
          <InputField
            name="android-zero-touch-name"
            value={selectedFleetName}
            readOnly
            inputOptions={{ "aria-label": "Name" }}
            helpText={
              <>
                For your configuration, use this name and pick{" "}
                <b>Android Device Policy</b> as your <b>EMM DPC</b>.
              </>
            }
          />
        </div>
        <div className={`${baseClass}__field`}>
          <div className={`${baseClass}__field-header`}>
            <span className={`${baseClass}__field-label`}>DPC extras</span>
            <CopyButton
              copyText={dpcExtrasText}
              variant="secondary"
              ariaLabel="Copy DPC extras"
              disabled={isFetching || !hasConfig}
            />
          </div>
          {renderCodeBlock()}
        </div>
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
