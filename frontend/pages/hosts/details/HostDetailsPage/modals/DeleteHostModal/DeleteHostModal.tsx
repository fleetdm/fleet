import React from "react";
import Modal from "components/Modal";

import Button from "components/buttons/Button";
import { ITeam } from "interfaces/team";

interface IDeleteHostModal {
  onSubmit: (team: ITeam) => void;
  onCancel: () => void;
  hostName?: string;
  isDeletingHost: boolean;
}

const baseClass = "delete-host-modal";

const DeleteHostModal = ({
  onCancel,
  onSubmit,
  hostName,
  isDeletingHost,
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
        <div className="modal-cta-wrap">
          <Button
            onClick={onSubmit}
            variant="alert"
            className="delete-loading"
            spinner={isDeletingHost}
          >
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteHostModal;
