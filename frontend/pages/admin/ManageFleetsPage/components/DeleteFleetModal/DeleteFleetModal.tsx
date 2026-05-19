import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-fleet-modal";

interface IDeleteFleetModalProps {
  name: string;
  isUpdatingFleets: boolean;
  onSubmit: () => void;
  onCancel: () => void;
}

const DeleteFleetModal = ({
  name,
  isUpdatingFleets,
  onSubmit,
  onCancel,
}: IDeleteFleetModalProps): JSX.Element => {
  return (
    <Modal
      title="Delete fleet"
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <p>
        This will delete the{" "}
        <span className={`${baseClass}__name`}>{name}</span> fleet.
      </p>
      <p>
        Users on this fleet who are not assigned to other fleets won&apos;t be
        able to log in.
      </p>
      <div className="modal-cta-wrap">
        <Button
          type="button"
          onClick={onSubmit}
          variant="alert"
          className="delete-loading"
          isLoading={isUpdatingFleets}
        >
          Delete
        </Button>
        <Button onClick={onCancel} variant="inverse-alert">
          Cancel
        </Button>
      </div>
    </Modal>
  );
};

export default DeleteFleetModal;
