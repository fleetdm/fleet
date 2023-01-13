import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import CustomLink from "components/CustomLink";

export interface IInfoModalProps {
  onCancel: () => void;
}

const baseClass = "device-user-info";

const InfoModal = ({ onCancel }: IInfoModalProps): JSX.Element => {
  return (
    <Modal
      title="Turn on MDM"
      onExit={onCancel}
      className={`${baseClass}__modal`}
    >
      <div>
        <p>
          To turn on MDM, Apple Inc. requires that you download and install a
          profile.
        </p>
        <p>1. Download your profile.</p>
        <p>
          <Button
            variant="brand"
            onClick={() => console.log(true)}
            isLoading // TODO: piping
          >
            Download
          </Button>
        </p>
        <p>
          2. From the Apple menu in the top left corner of your screen, select{" "}
          <strong>System Settings</strong> or{" "}
          <strong>System Preferences</strong>.
        </p>
        <p>
          3. In the search bar, type “Profiles.” Select Profiles, find and
          select <strong>Enrollment Profile</strong>, and select{" "}
          <strong>Install</strong>.
        </p>
        <p>
          4. Enter your password, and select <strong>Enroll</strong>.
        </p>
        <p>
          5. Close this window and select <strong>Refetch</strong> on your My
          device page to tell your organization that MDM is on.
        </p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default InfoModal;
