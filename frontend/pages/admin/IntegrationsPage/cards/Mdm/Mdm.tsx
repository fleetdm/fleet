import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import FileSaver from "file-saver";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import mdmAppleAPI from "services/entities/mdm_apple";
import mdmAppleBmAPI from "services/entities/mdm_apple_bm";
import { IMdmApple, IMdmAppleBm } from "interfaces/mdm";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

import RequestModal from "./components/RequestModal";
import EditTeamModal from "./components/EditTeamModal";

// MDM TODO: key validation?
// import { isValidKeys } from "../../..";

const baseClass = "mdm-integrations";

const readableDate = (date: string) => {
  const dateString = new Date(date);

  return new Intl.DateTimeFormat(navigator.language, {
    year: "numeric",
    month: "long",
    day: "numeric",
  }).format(dateString);
};

const Mdm = (): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  const [showRequestModal, setShowRequestModal] = useState(false);
  const [showEditTeamModal, setShowEditTeamModal] = useState(false);

  const {
    data: mdmApple,
    isLoading: isLoadingMdmApple,
    error: errorMdmApple,
  } = useQuery<IMdmApple, Error, IMdmApple>(
    ["mdmAppleAPI"],
    () => mdmAppleAPI.loadAll(),
    {
      enabled: isPremiumTier,
      staleTime: 5000,
    }
  );

  const {
    data: mdmAppleBm,
    isLoading: isLoadingMdmAppleBm,
    error: errorMdmAppleBm,
  } = useQuery<IMdmAppleBm, Error, IMdmAppleBm>(
    ["mdmAppleBmAPI"],
    () => mdmAppleBmAPI.loadAll(),
    {
      enabled: isPremiumTier,
      staleTime: 5000,
    }
  );

  // MDM TODO: Test manually after backend is merged
  const {
    data: keys,
    error: fetchKeysError,
    isFetching: isFetchingKeys,
  } = useQuery<string, Error>(["keys"], () => mdmAppleBmAPI.loadKeys(), {
    enabled: isPremiumTier,
    refetchOnWindowFocus: false,
  });

  const toggleRequestModal = () => {
    setShowRequestModal(!showRequestModal);
  };

  const toggleEditTeamModal = () => {
    setShowEditTeamModal(!showEditTeamModal);
  };

  const onDownloadKeys = (evt: React.MouseEvent) => {
    evt.preventDefault();

    // MDM TODO: Confirm error flash message
    if (isFetchingKeys || fetchKeysError) {
      renderFlash(
        "error",
        "Your MDM business manager keys could not be downloaded. Please try again."
      );
      return false;
    }

    if (keys) {
      // MDM TODO: Validate keys like we validate certificates?
      // if (keys && isValidKeys(keys)) {
      const filename = "fleet.pem";
      const file = new global.window.File([keys], filename, {
        type: "application/x-pem-file",
      });

      FileSaver.saveAs(file);
    } else {
      renderFlash(
        "error",
        "Your MDM business manager keys could not be downloaded. Please TODO ACTION."
      );
    }
    return false;
  };

  const renderMdmAppleSection = () => {
    if (errorMdmApple) {
      return <DataError />;
    }

    if (!mdmApple) {
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
            <Button onClick={toggleRequestModal} variant="brand">
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
              <CustomLink url="https://www.youtube.com" text="See how" newTab />
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
          <p>{mdmApple.common_name}</p>
          <h4>Serial number</h4>
          <p>{mdmApple.serial_number}</p>
          <h4>Issuer</h4>
          <p>{mdmApple.issuer}</p>
          <h4>Renew date</h4>
          <p>{readableDate(mdmApple.renew_date)}</p>
        </div>
      </>
    );
  };

  const renderMdmAppleBm = () => {
    if (errorMdmAppleBm) {
      return <DataError />;
    }

    if (!mdmAppleBm) {
      return (
        <>
          <div className={`${baseClass}__section-description`}>
            Connect Fleet to your Apple Business Manager account to
            automatically enroll macOS hosts to Fleet when they’re first
            unboxed.
          </div>
          <div className={`${baseClass}__section-instructions`}>
            <p>1. Download your public and private keys.</p>
            <Button onClick={onDownloadKeys} variant="brand">
              Download
            </Button>
            <p>
              2. Sign in to{" "}
              <CustomLink
                url="https://business.apple.com/"
                text="Apple Business Manager"
                newTab
              />
              <br />
              If your organization doesn’t have an account, select{" "}
              <b>Enroll now</b>.
            </p>
            <p>
              3. In Apple Business Manager, upload your public key and download
              your server token.
            </p>
            <p>
              4. Deploy Fleet with <b>mdm</b> configuration.{" "}
              <CustomLink
                url="https://business.apple.com/"
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
          To use automatically enroll macOS hosts to Fleet when they’re first
          unboxed, Apple Inc. requires a server token.
        </div>
        <div className={`${baseClass}__section-information`}>
          <h4>
            <TooltipWrapper tipContent="macOS hosts will be added to this team when they’re first unboxed.">
              Team
            </TooltipWrapper>
          </h4>
          <p>
            {mdmAppleBm.default_team || "No team"}{" "}
            <Button
              className={`${baseClass}__edit-team-btn`}
              onClick={toggleEditTeamModal}
              variant="text-icon"
            >
              Edit <Icon name="pencil" />
            </Button>
          </p>
          <h4>Apple ID</h4>
          <p>{mdmAppleBm.apple_id}</p>
          <h4>Organization name</h4>
          <p>{mdmAppleBm.organization_name}</p>
          <h4>MDM Server URL</h4>
          <p>{mdmAppleBm.mdm_server_url}</p>
          <h4>Renew date</h4>
          <p>{readableDate(mdmAppleBm.renew_date)}</p>
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
      {isPremiumTier && (
        <div className={`${baseClass}__section`}>
          <h2>Apple Business Manager</h2>
          {isLoadingMdmAppleBm ? <Spinner /> : renderMdmAppleBm()}
        </div>
      )}
      {showRequestModal && (
        <RequestModal
          onCancel={toggleRequestModal}
          onRequest={toggleRequestModal}
        />
      )}
      {showEditTeamModal && (
        <EditTeamModal
          onCancel={toggleEditTeamModal}
          onEdit={toggleEditTeamModal}
        />
      )}
    </div>
  );
};

export default Mdm;
