import React, { useContext } from "react";
import FileSaver from "file-saver";

import mdmAPI from "services/entities/mdm";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { NotificationContext } from "context/notification";

interface IManualEnrollMdmModalProps {
  onCancel: () => void;
  token?: string;
}

const baseClass = "manual-enroll-mdm-modal";

const ManualEnrollMdmModal = ({
  onCancel,
  token = "",
}: IManualEnrollMdmModalProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const onDownload = async () => {
    try {
      const profileContent = await mdmAPI.downloadManualEnrollmentProfile(
        token
      );
      const file = new File(
        [profileContent],
        "fleet-mdm-enrollment-profile.mobileconfig"
      );
      FileSaver.saveAs(file);
    } catch (e) {
      renderFlash("error", "Failed to download the profile. Please try again.");
    }
  };

  const renderModalBody = () => {
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
            <Button
              className={`${baseClass}__download-button`}
              onClick={onDownload}
            >
              Download
            </Button>
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
            Select <b>Install...</b> then confirm again clicking <b>Install</b>.
          </li>
          <li>Enter your password when you get a prompt.</li>
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
      onEnter={onCancel}
      className={baseClass}
      width="xlarge"
    >
      {renderModalBody()}
    </Modal>
  );
};

export default ManualEnrollMdmModal;
