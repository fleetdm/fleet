import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import mdmAppleAPI from "services/entities/mdm_apple";
import mdmAppleBmAPI from "services/entities/mdm_apple_bm";
import { IMdmApple, IMdmAppleBm } from "interfaces/mdm";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import RequestModal from "./components/RequestModal";

const baseClass = "mdm-integrations";

const readableDate = (date: string) => {
  const dateString = new Date(date);

  return new Intl.DateTimeFormat("en-US", {
    year: "numeric",
    month: "long",
    day: "numeric",
  }).format(dateString);
};

const Mdm = (): JSX.Element => {
  const { isPremiumTier, isMdmEnabled } = useContext(AppContext);

  const [showRequestModal, setShowRequestModal] = useState(false);

  const {
    data: mdmApple,
    isLoading: isLoadingMdmApple,
    error: errorMdmApple,
  } = useQuery<IMdmApple, Error, IMdmApple>(
    ["mdmAppleAPI"],
    () => mdmAppleAPI.loadAll(),
    {
      enabled: !!isMdmEnabled && isPremiumTier,
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
      enabled: !!isMdmEnabled && isPremiumTier,
      staleTime: 5000,
    }
  );

  const toggleRequestModal = () => {
    setShowRequestModal(!showRequestModal);
  };

  const downloadKeys = () => {
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
            <p>2. Go to your email to download to download your CSR.</p>
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
          <p>
            <b>Common name (CN)</b>
            <br />
            {mdmApple.common_name}
          </p>
          <p>
            <b>Serial number</b>
            <br />
            {mdmApple.serial_number}
          </p>
          <p>
            <b>Issuer</b>
            <br />
            {mdmApple.issuer}
          </p>
          <p>
            <b>Renew date</b>
            <br />
            {readableDate(mdmApple.renew_date)}
          </p>
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
            <Button onClick={downloadKeys} variant="brand">
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
          <p>
            <b>Team</b>
            <br />
            {mdmAppleBm.default_team || "No team"}
          </p>
          <p>
            <b>Apple ID</b>
            <br />
            {mdmAppleBm.apple_id}
          </p>
          <p>
            <b>Organization name</b>
            <br />
            {mdmAppleBm.organization_name}
          </p>
          <p>
            <b>MDM Server URL</b>
            <br />
            {mdmAppleBm.mdm_server_url}
          </p>
          <p>
            <b>Renew date</b>
            <br />
            {readableDate(mdmAppleBm.renew_date)}
          </p>
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
    </div>
  );
};

export default Mdm;
