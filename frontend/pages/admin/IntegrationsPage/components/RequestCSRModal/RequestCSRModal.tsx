import React, { useState, useEffect } from "react";

import Modal from "components/Modal";
// @ts-ignore
import RequestCSRForm from "../RequestCSRForm";

const DESTINATION = "https://www.imatest.url";

const baseClass = " modal request-csr-modal";

interface IRequestCSRModalProps {
  onSubmit: () => void;
  onCancel: () => void;
  userEmail: string;
  orgName: string;
}

const RequestCSRModal = ({
  onCancel,
  onSubmit,
  userEmail,
  orgName,
}: IRequestCSRModalProps): JSX.Element => {
  return (
    <Modal title="Request" onExit={onCancel} className={baseClass}>
      <>
        <p>
          A CSR and key for APNs and a certificate and key for SCEP are required
          to connect Fleet to Apple Developer. Apple Inc. requires the following
          information. <br />
          <br />
          fleetdm.com will send your CSR to the below email. Your certificate
          and key for SCEP will be downloaded in the browser.
        </p>
        <RequestCSRForm
          onCancel={onCancel}
          onSubmit={onSubmit}
          userEmail={userEmail}
          currentOrgName={orgName}
          requestCSRDestination={DESTINATION}
        />
      </>
    </Modal>
  );
};

export default RequestCSRModal;
