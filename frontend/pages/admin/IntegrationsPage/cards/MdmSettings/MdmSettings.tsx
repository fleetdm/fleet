import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";

import mdmAppleAPI from "services/entities/mdm_apple";
import { IMdmApple } from "interfaces/mdm";

import { readableDate } from "utilities/helpers";
import PATHS from "router/paths";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import RequestCSRModal from "./components/RequestCSRModal";
import EndUserMigrationSection from "./components/EndUserMigrationSection/EndUserMigrationSection";
import WindowsMdmSection from "./components/WindowsMdmSection/WindowsMdmSection";

const baseClass = "mdm-settings";

interface IMdmSettingsProps {
  router: InjectedRouter;
}

const MdmSettings = ({ router }: IMdmSettingsProps) => {
  const { isPremiumTier, config } = useContext(AppContext);

  const [showRequestCSRModal, setShowRequestCSRModal] = useState(false);

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

  const toggleRequestCSRModal = () => {
    setShowRequestCSRModal(!showRequestCSRModal);
  };

  const navigateToWindowsMdm = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM_WINDOWS);
  };

  // The API returns a 404 error if APNs is not configured yet, in that case we
  // want to prompt the user to download the certs and keys to configure the
  // server instead of the default error message.
  const showMdmAppleError = errorMdmApple && errorMdmApple.status !== 404;

  const renderMdmAppleSection = () => {
    if (showMdmAppleError) {
      return <DataError />;
    }

    if (!appleAPNInfo) {
      return (
        <>
          <div className={`${baseClass}__section-description`}>
            Connect Fleet to Apple Push Certificates Portal to change settings
            and install software on your macOS hosts.
          </div>
          <div className={`${baseClass}__section-instructions`}>
            <p>
              1. Request a certificate signing request (CSR) and key for Apple
              Push Notification Service (APNs) and a certificate and key for
              Simple Certificate Enrollment Protocol (SCEP).
            </p>
            <Button onClick={toggleRequestCSRModal} variant="brand">
              Request
            </Button>
            <p>2. Go to your email to download your CSR.</p>
            <p>
              3.{" "}
              <CustomLink
                url="https://identity.apple.com/pushcert/"
                text="Sign in to Apple Push Certificates Portal"
                newTab
              />
              <br />
              If you don’t have an Apple ID, select <b>Create yours now</b>.
            </p>
            <p>
              4. In Apple Push Certificates Portal, select{" "}
              <b>Create a Certificate</b>, upload your CSR, and download your
              APNs certificate.
            </p>
            <p>
              5. Deploy Fleet with <b>mdm</b> configuration.{" "}
              <CustomLink
                url="https://fleetdm.com/docs/deploying/configuration#mobile-device-management-mdm"
                text="See how"
                newTab
              />
            </p>
          </div>
        </>
      );
    }

    return (
      <>
        <div className={`${baseClass}__section-description`}>
          To change settings and install software on your macOS hosts, Apple
          Inc. requires an Apple Push Notification service (APNs) certificate.
        </div>
        <div className={`${baseClass}__section-information`}>
          <h4>Common name (CN)</h4>
          <p>{appleAPNInfo.common_name}</p>
          <h4>Serial number</h4>
          <p>{appleAPNInfo.serial_number}</p>
          <h4>Issuer</h4>
          <p>{appleAPNInfo.issuer}</p>
          <h4>Renew date</h4>
          <p>{readableDate(appleAPNInfo.renew_date)}</p>
        </div>
      </>
    );
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <h2>Apple Push Certificates Portal</h2>
        {isLoadingMdmApple ? <Spinner /> : renderMdmAppleSection()}
      </div>
      {/* TODO: remove conditional rendering when windows MDM is released. */}
      {config?.mdm_enabled && (
        <WindowsMdmSection
          turnOnWindowsMdm={navigateToWindowsMdm}
          editWindowsMdm={navigateToWindowsMdm}
        />
      )}
      {isPremiumTier && (
        <>
          <div className={`${baseClass}__section`}>
            <EndUserMigrationSection router={router} />
          </div>
        </>
      )}
      {showRequestCSRModal && (
        <RequestCSRModal onCancel={toggleRequestCSRModal} />
      )}
    </div>
  );
};

export default MdmSettings;
