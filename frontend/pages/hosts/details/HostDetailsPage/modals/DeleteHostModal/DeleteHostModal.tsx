import React from "react";
import Modal from "components/Modal";

import Button from "components/buttons/Button";
import { ITeam } from "interfaces/team";

interface IDeleteHostModal {
  onSubmit: (team: ITeam) => void;
  onCancel: () => void;
  hostName?: string;
  isUpdatingHost: boolean;
}

const baseClass = "delete-host-modal";

const DeleteHostModal = ({
  onCancel,
  onSubmit,
  hostName,
  isUpdatingHost,
}: IDeleteHostModal): JSX.Element => {
  return (
    <Modal
      title="Delete host"
      onExit={onCancel}
      className={`${baseClass}__modal`}
    >
      <>
        <p>
          This action will delete the host <strong>{hostName}</strong> from
          Fleet.
        </p>
        <p>The host comes back online, it will automatically re-enroll.</p>
        <p>To prevent re-enrollment, uninstall osquery on the host.</p>
        <div className="modal-cta-wrap">
          <Button
            onClick={onSubmit}
            variant="alert"
            className="delete-loading"
            isLoading={isUpdatingHost}
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
