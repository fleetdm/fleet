import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import { AxiosError } from "axios";

import PATHS from "router/paths";
import mdmAppleAPI from "services/entities/mdm_apple";
import { IMdmApple, getMdmServerUrl } from "interfaces/mdm";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import DataError from "components/DataError";
import Spinner from "components/Spinner";

import ApplePushCertSetup from "./components/content/ApplePushCertSetup";
import ApplePushCertInfo from "./components/content/ApplePushCertInfo";

import RenewCertModal from "./components/modals/RenewCertModal";
import TurnOffAppleMdmModal from "./components/modals/TurnOffAppleMdmModal";

export const baseClass = "apple-mdm-page";

const AppleMdmPage = ({ router }: { router: InjectedRouter }) => {
  const { config } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [isUpdating, setIsUpdating] = useState(false);
  const [showRenewCertModal, setShowRenewCertModal] = useState(false);
  const [showTurnOffMdmModal, setShowTurnOffMdmModal] = useState(false);

  // Currently the status of this API call is what determines various UI states on
  // this page. Because of this we will not render any of this components UI until this API
  // call has completed.
  const {
    data: appleAPNInfo,
    isLoading,
    isRefetching,
    refetch,
    error: errorMdmApple,
  } = useQuery<IMdmApple, AxiosError, IMdmApple>(
    ["appleAPNInfo"],
    () => mdmAppleAPI.getAppleAPNInfo(),
    {
      retry: (tries, error) => error.status !== 404 && tries <= 3,
      enabled: config?.mdm.enabled_and_configured,
      staleTime: 5000,
      refetchOnWindowFocus: false,
      onSettled: () => setIsUpdating(false),
    }
  );

  const toggleRenewCertModal = () => {
    setShowRenewCertModal((prevState) => !prevState);
  };

  const toggleTurnOffMdmModal = () => {
    setShowTurnOffMdmModal((prevState) => !prevState);
  };

  const turnOffMdm = useCallback(async () => {
    setIsUpdating(true);
    toggleTurnOffMdmModal();
    try {
      await mdmAppleAPI.deleteApplePushCertificate();
      router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
      renderFlash("success", "MDM turned off successfully.");
    } catch (e) {
      renderFlash("error", "Couldn't turn off MDM. Please try again.");
      setIsUpdating(false);
    }
  }, [renderFlash, router]);

  const onRenewCert = useCallback(() => {
    refetch();
    toggleRenewCertModal();
  }, [refetch]);

  const onSetupSuccess = useCallback(() => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  }, [router]);

  // The API returns a 404 error if APNs is not configured yet, in that case we
  // want to prompt the user to configure the server instead of the default error message.
  const isMdmNotConfigured = errorMdmApple && errorMdmApple.status !== 404;

  const showSpinner = isLoading || isUpdating || isRefetching;
  const showError = !config || isMdmNotConfigured;
  const showContent = !showSpinner && !showError;

  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to MDM"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-mdm`}
        />
        <h1>Apple Push Certificate Portal</h1>
        {showSpinner && <Spinner />}
        {showError && <DataError />}
        {showContent &&
          (!appleAPNInfo ? (
            <ApplePushCertSetup
              baseClass={baseClass}
              onSetupSuccess={onSetupSuccess}
            />
          ) : (
            <ApplePushCertInfo
              baseClass={baseClass}
              appleAPNInfo={appleAPNInfo}
              orgName={config.org_info.org_name}
              serverUrl={getMdmServerUrl(config.server_settings)}
              onClickRenew={toggleRenewCertModal}
              onClickTurnOff={toggleTurnOffMdmModal}
            />
          ))}
        {showRenewCertModal && (
          <RenewCertModal
            onCancel={toggleRenewCertModal}
            onRenew={onRenewCert}
          />
        )}
        {showTurnOffMdmModal && (
          <TurnOffAppleMdmModal
            onCancel={toggleTurnOffMdmModal}
            onConfirm={turnOffMdm}
          />
        )}
      </>
    </MainContent>
  );
};

export default AppleMdmPage;
