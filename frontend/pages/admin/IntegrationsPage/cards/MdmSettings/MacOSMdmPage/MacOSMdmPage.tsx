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
import TurnOffMacOsMdmModal from "./components/modals/TurnOffMacOsMdmModal";

export const baseClass = "mac-os-mdm-page";

const MacOSMdmPage = ({ router }: { router: InjectedRouter }) => {
  const { config } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [isTurningOff, setIsTurningOff] = useState(false);
  const [showRenewCertModal, setShowRenewCertModal] = useState(false);
  const [showTurnOffMdmModal, setShowTurnOffMdmModal] = useState(false);

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

  const toggleRenewCertModal = () => {
    setShowRenewCertModal((prevState) => !prevState);
  };

  const toggleTurnOffMdmModal = () => {
    setShowTurnOffMdmModal((prevState) => !prevState);
  };

  const turnOffMdm = useCallback(async () => {
    setIsTurningOff(true);
    toggleTurnOffMdmModal();
    console.log("Turn off MDM confirmed");
    try {
      // TODO: handle submission
      // await mdmApi.TurnOffMacOsMdm();
      renderFlash("success", "macOS MDM turned off successfully.");
      router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
    } catch (e) {
      renderFlash("error", "Couldn’t turn off MDM. Please try again.");
      setIsTurningOff(false);
    }
  }, [renderFlash, router]);

  // The API returns a 404 error if APNs is not configured yet, in that case we
  // want to prompt the user to download the certs and keys to configure the
  // server instead of the default error message.
  const isMdmAppleError = errorMdmApple && errorMdmApple.status !== 404;

  const showSpinner = isLoadingMdmApple || isTurningOff;
  const showError = !config || isMdmAppleError;
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
              // onClickRequest={toggleRequestCSRModal}
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
          <RenewCertModal onCancel={toggleRenewCertModal} />
        )}
        {showTurnOffMdmModal && (
          <TurnOffMacOsMdmModal
            onCancel={toggleTurnOffMdmModal}
            onConfirm={turnOffMdm}
          />
        )}
      </>
    </MainContent>
  );
};

export default MacOSMdmPage;
