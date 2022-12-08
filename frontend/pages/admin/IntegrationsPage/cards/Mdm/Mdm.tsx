import React, { useContext, useState } from "react";

import Button from "components/buttons/Button";

import CustomLink from "components/CustomLink";
import { AppContext } from "context/app";

const baseClass = "mdm-integrations";

const Mdm = (): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);

  const [showRequestModal, setShowRequestModal] = useState(false);

  const toggleRequestModal = () => {
    setShowRequestModal(!showRequestModal);
  };

  const downloadKeys = () => {
    return false;
  };

  return (
    <>
      <div className={`${baseClass}__section`}>
        <h2>Apple Push Certificates Portal</h2>
        <div className={`${baseClass}__description`}>
          Connect Fleet to Apple Push Certificates Portal to change settings and
          install software on your macOS hosts.
        </div>
        <div className={`${baseClass}__instructions`}>
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
            <b>Create a Certificate</b>, upload your CSR, and download your APNs
            certificate.
          </p>
          <p>
            5. Deploy Fleet with mdm configuration.{" "}
            <CustomLink url="https://www.youtube.com" text="See how" newTab />
          </p>
        </div>
      </div>
      {isPremiumTier && (
        <div className={`${baseClass}__section`}>
          <h2>Apple Business Manager</h2>
          <div className={`${baseClass}__description`}>
            Connect Fleet to your Apple Business Manager account to
            automatically enroll macOS hosts to Fleet when they’re first
            unboxed.
          </div>
          <div className={`${baseClass}__instructions`}>
            <p>1. Download your public and private keys.</p>
            <Button onClick={downloadKeys} variant="brand">
              Download
            </Button>
            <p>
              2. Sign in to{" "}
              <CustomLink
                url="https://business.apple.com/"
                text="Apple Business Manager"
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
              <CustomLink url="https://business.apple.com/" text="See how" />
            </p>
          </div>
        </div>
      )}
    </>
  );
};

export default Mdm;
