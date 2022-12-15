import React from "react";

import Modal from "components/Modal";
import RequestCSRForm from "../RequestCSRForm";

const DESTINATION = "https://www.imatest.url";

const baseClass = " modal request-csr-modal";

const RequestCSRModal = (): JSX.Element => {
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
        <RequestCSRForm requestCSRDestination={DESTINATION} />
      </>
    </Modal>
  );
};

export default RequestCSRModal;
