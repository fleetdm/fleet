import React from "react";
import Modal from "components/Modal";

import Button from "components/buttons/Button";
import { ITeam } from "interfaces/team";

interface IDeleteHostModal {
  onSubmit: (team: ITeam) => void;
  onCancel: () => void;
  hostName?: string;
}

const baseClass = "delete-host-modal";

const DeleteHostModal = ({
  onCancel,
  onSubmit,
  hostName,
}: IDeleteHostModal): JSX.Element => {
  return (
    <Modal
      title="Delete host"
      onExit={onCancel}
      className={`${baseClass}__modal`}
    >
      <>
        <p>
          This action will delete the host <strong>{hostName}</strong> from your
          Fleet instance.
        </p>
        <p>
          The host will automatically re-enroll when it checks back into Fleet.
        </p>
        <p>
          To prevent re-enrollment, you can uninstall osquery on the host or
          revoke the host&apos;s enroll secret.
        </p>
        <div className={`${baseClass}__button-wrap modal-btn-wrap`}>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
          <Button onClick={onSubmit} variant="alert">
            Delete
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteHostModal;
