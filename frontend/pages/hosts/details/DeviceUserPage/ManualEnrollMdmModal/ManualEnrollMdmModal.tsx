import React, { useState } from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

import mdmAPI from "services/entities/mdm";

export interface IInfoModalProps {
  onCancel: () => void;
}

const baseClass = "manual-enroll-mdm-modal";

const ManualEnrollMdmModal = ({ onCancel }: IInfoModalProps): JSX.Element => {
  const [showDownloading, setshowDownloading] = useState(false);

  const handleDownload = async () => {
    setshowDownloading(true);
    setTimeout(() => {
      setshowDownloading(false);
    }, 1000);

    try {
      await mdmAPI.downloadEnrollmentProfile();
    } catch (e) {
      console.error("error downloading profile:", e);
    }
  };

  return (
    <Modal title="Turn on MDM" onExit={onCancel} className={baseClass}>
      <div>
        <p className={`${baseClass}__description`}>
          To turn on MDM, Apple Inc. requires that you download and install a
          profile
        </p>
        <ol>
          <li>
            <span>Download your profile.</span>
            <Button
              type="button"
              onClick={handleDownload}
              variant="brand"
              isLoading={showDownloading}
              className={`${baseClass}__download-button`}
            >
              Download
            </Button>
          </li>
          <li>
            From the Apple menu in the top left corner of your screen, select{" "}
            <b>System Settings</b> or <b>System Preferences</b>.
          </li>
          <li>
            In the search bar, type “Profiles.” Select <b>Profiles</b>, find and
            select <b>Enrollment Profile</b>, and select <b>Install</b>.
          </li>
          <li>
            Enter your password, and select <b>Enroll</b>.
          </li>
          <li>
            Close this window and select <b>Refetch</b> on your My device page
            to tell your organization that MDM is on.
          </li>
        </ol>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ManualEnrollMdmModal;
