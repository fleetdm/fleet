import React from "react";

import endpoints from "utilities/endpoints";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const { DEVICE_USER_MDM_ENROLLMENT_PROFILE } = endpoints;

interface IManualEnrollMdmModalProps {
  onCancel: () => void;
  token?: string;
}

const baseClass = "manual-enroll-mdm-modal";

const ManualEnrollMdmModal = ({
  onCancel,
  token = "",
}: IManualEnrollMdmModalProps): JSX.Element => {
  const renderModalBody = () => {
    const downloadUrl = `/api${DEVICE_USER_MDM_ENROLLMENT_PROFILE(token)}`;

    return (
      <div>
        <p className={`${baseClass}__description`}>
          To turn on MDM, Apple Inc. requires that you download and install a
          profile.
        </p>
        <ol>
          <li>
            <span>Download your profile.</span>
            <br />
            {/* TODO: make a link component that appears as a button. */}
            <a
              className={`${baseClass}__download-link`}
              href={downloadUrl}
              download
            >
              Download
            </a>
          </li>
          <li>Open the profile you just downloaded.</li>
          <li>
            From the Apple menu in the top left corner of your screen, select{" "}
            <b>System Settings</b>.
          </li>
          <li>
            In the search bar, type “Profiles”. Select <b>Profiles</b>, find and
            double click the <br /> <b>[Organization name] enrollment</b>{" "}
            profile.
          </li>
          <li>
            Select <b>Enroll</b> then enter your password.
          </li>
          <li>
            Select <b>Done</b> to close this window and select <b>Refetch</b> on
            your My device page to tell <br /> your organization that MDM is on.
          </li>
        </ol>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </div>
    );
  };

  return (
    <Modal
      title="Turn on MDM"
      onExit={onCancel}
      className={baseClass}
      width="xlarge"
    >
      {renderModalBody()}
    </Modal>
  );
};

export default ManualEnrollMdmModal;
