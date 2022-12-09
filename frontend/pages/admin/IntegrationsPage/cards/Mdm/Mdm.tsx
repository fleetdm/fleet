import React, { useContext, useState } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import appleMdmAPI from "services/entities/appleMdm";
import { IAppleMdm } from "interfaces/mdm";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import RequestModal from "./components/RequestModal";

const baseClass = "mdm-integrations";

const Mdm = (): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);

  // TODO: Where are we setting MDM enabled?
  const mdmEnabled = true;

  const [showRequestModal, setShowRequestModal] = useState(false);

  const {
    data: appleMDM,
    isLoading: isLoadingAppleMdm,
    error: errorAbm,
  } = useQuery<IAppleMdm, Error, IAppleMdm>(
    ["appleMdmAPI"],
    () => appleMdmAPI.loadAll(),
    {
      enabled: !!mdmEnabled && isPremiumTier,
      staleTime: 5000,
    }
  );
  const toggleRequestModal = () => {
    setShowRequestModal(!showRequestModal);
  };

  const downloadKeys = () => {
    return false;
  };

  console.log("appleMDM", appleMDM);
  const renderApnSection = () => {
    if (!appleMDM) {
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
              5. Deploy Fleet with mdm configuration.{" "}
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
            {appleMDM.apn.commonName}
          </p>
          <p>
            <b>Serial number</b>
            <br />
            {appleMDM.apn.serialNumber}
          </p>
          <p>
            <b>Issuer</b>
            <br />
            {appleMDM.apn.issuer}
          </p>
          <p>
            <b>Renew data</b>
            <br />
            {appleMDM.apn.renewDate}
          </p>
        </div>
      </>
    );
  };

  const renderAbmSection = () => {
    if (!appleMDM) {
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
              If your organization doesn’t have an account, select Enroll now.
            </p>
            <p>
              3. In Apple Business Manager, upload your public key and download
              your server token.
            </p>
            <p>
              4. Deploy Fleet with mdm configuration.{" "}
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
            {appleMDM.abm.team || "No team"}
          </p>
          <p>
            <b>Apple ID</b>
            <br />
            {appleMDM.abm.appleId}
          </p>
          <p>
            <b>Organization name</b>
            <br />
            {appleMDM.abm.organizationName}
          </p>
          <p>
            <b>MDM Server URL</b>
            <br />
            {appleMDM.abm.mdmServerUrl}
          </p>
          <p>
            <b>Renew date</b>
            <br />
            {appleMDM.abm.renewDate}
          </p>
        </div>
      </>
    );
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <h2>Apple Push Certificates Portal</h2>
        {isLoadingAppleMdm ? <Spinner /> : renderApnSection()}
      </div>
      {isPremiumTier && (
        <div className={`${baseClass}__section`}>
          <h2>Apple Business Manager</h2>
          {isLoadingAppleMdm ? <Spinner /> : renderAbmSection()}
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
