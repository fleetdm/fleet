import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

interface IAutoEnrollMdmModalProps {
  onCancel: () => void;
}

const baseClass = "auto-enroll-mdm-modal";

const AutoEnrollMdmModal = ({
  onCancel,
}: IAutoEnrollMdmModalProps): JSX.Element => {
  return (
    <Modal
      title="Turn on MDM"
      onExit={onCancel}
      className={baseClass}
      width="xlarge"
    >
      <div>
        <p className={`${baseClass}__description`}>
          To turn on MDM, Apple Inc. requires that you install a profile.
        </p>
        <ol>
          <li>
            From the Apple menu in the top left corner of your screen, select{" "}
            <b>System Settings</b> or <b>System Preferences</b>.
          </li>
          <li>
            In the search bar, type “Profiles”. Select <b>Profiles</b>, double
            click on <b>Enrollment Profile</b>, and select <b>Install</b>.
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

export default AutoEnrollMdmModal;
