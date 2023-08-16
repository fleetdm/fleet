import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import PATHS from "router/paths";
import mdmAppleAPI from "services/entities/mdm_apple";
import { IMdmApple } from "interfaces/mdm";
import { readableDate } from "utilities/helpers";
import { AppContext } from "context/app";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import RequestCSRModal from "../components/RequestCSRModal";

const baseClass = "mac-os-mdm-page";

interface IApplePuushCertificatePortalSetupProps {
  onClickRequest: () => void;
}

const ApplePushCertificatePortalSetup = ({
  onClickRequest,
}: IApplePuushCertificatePortalSetupProps) => {
  return (
    <div className={`${baseClass}__page-content ${baseClass}__setup-content`}>
      <p className={`${baseClass}__setup-description`}>
        Connect Fleet to Apple Push Certificates Portal to change settings and
        install software on your macOS hosts.
      </p>
      <ol className={`${baseClass}__setup-instructions-list`}>
        <li>
          <p>
            1. Request a certificate signing request (CSR) and key for Apple
            Push Notification Service (APNs) and a certificate and key for
            Simple Certificate Enrollment Protocol (SCEP).
          </p>
          <Button
            className={`${baseClass}__request-button`}
            onClick={onClickRequest}
            variant="brand"
          >
            Request
          </Button>
        </li>
        <li>
          <p>2. Go to your email to download your CSR.</p>
        </li>
        <li>
          <p>
            3.{" "}
            <CustomLink
              url="https://identity.apple.com/pushcert/"
              text="Sign in to Apple Push Certificates Portal"
              newTab
            />
            <br />
            If you don&apos;t have an Apple ID, select <b>Create yours now</b>.
          </p>
        </li>
        <li>
          <p>
            4. In Apple Push Certificates Portal, select{" "}
            <b>Create a Certificate</b>, upload your CSR, and download your APNs
            certificate.
          </p>
        </li>
        <li>
          <p>
            5. Deploy Fleet with <b>mdm</b> configuration.{" "}
            <CustomLink
              url="https://fleetdm.com/docs/deploying/configuration#mobile-device-management-mdm"
              text="See how"
              newTab
            />
          </p>
        </li>
      </ol>
    </div>
  );
};

interface IApplePushCertificatePortalSetupInfoProps {
  appleAPNInfo: IMdmApple;
}

const ApplePushCertificatePortalSetupInfo = ({
  appleAPNInfo,
}: IApplePushCertificatePortalSetupInfoProps) => {
  return (
    <dl className={`${baseClass}__page-content ${baseClass}__apc-info`}>
      <div>
        <dt>Common name (CN)</dt>
        <dd>{appleAPNInfo.common_name}</dd>
      </div>
      <div>
        <dt>Serial number</dt>
        <dd>{appleAPNInfo.serial_number}</dd>
      </div>
      <div>
        <dt>Issuer</dt>
        <dd>{appleAPNInfo.issuer}</dd>
      </div>
      <div>
        <dt>Renew date</dt>
        <dd>{readableDate(appleAPNInfo.renew_date)}</dd>
      </div>
    </dl>
  );
};

const MacOSMdmPage = () => {
  const { config } = useContext(AppContext);
  const [showRequestCSRModal, setShowRequestCSRModal] = useState(false);

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

  const toggleRequestCSRModal = () => {
    setShowRequestCSRModal((prevState) => !prevState);
  };

  const renderPageContent = () => {
    // The API returns a 404 error if APNs is not configured yet, in that case we
    // want to prompt the user to download the certs and keys to configure the
    // server instead of the default error message.
    const showMdmAppleError = errorMdmApple && errorMdmApple.status !== 404;

    if (showMdmAppleError) {
      return <DataError />;
    }

    if (!appleAPNInfo) {
      return (
        <ApplePushCertificatePortalSetup
          onClickRequest={toggleRequestCSRModal}
        />
      );
    }

    return <ApplePushCertificatePortalSetupInfo appleAPNInfo={appleAPNInfo} />;
  };

  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to MDM"
          path={PATHS.ADMIN_INTEGRATIONS_MDM}
          className={`${baseClass}__back-to-mdm`}
        />
        <h1>Apple Push Certificate Portal</h1>
        {isLoadingMdmApple ? <Spinner /> : renderPageContent()}
        {showRequestCSRModal && (
          <RequestCSRModal onCancel={toggleRequestCSRModal} />
        )}
      </>
    </MainContent>
  );
};

export default MacOSMdmPage;
