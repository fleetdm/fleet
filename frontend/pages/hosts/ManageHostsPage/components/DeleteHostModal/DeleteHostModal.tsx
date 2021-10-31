import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-host-modal";

interface IDeleteHostModalProps {
  selectedHostIds: number[];
  onSubmit: () => void;
  onCancel: () => void;
  isAllMatchingHostsSelected: boolean;
}

const DeleteHostModal = ({
  selectedHostIds,
  onSubmit,
  onCancel,
  isAllMatchingHostsSelected,
}: IDeleteHostModalProps): JSX.Element => {
  return (
    <Modal title={"Delete host"} onExit={onCancel} className={baseClass}>
      <form className={`${baseClass}__form`}>
        <p>
          This action will delete{" "}
          <b>
            {selectedHostIds.length}
            {isAllMatchingHostsSelected && "+"}{" "}
            {selectedHostIds.length === 1 ? "host" : "hosts"}
          </b>{" "}
          from your Fleet instance.
        </p>
        <p>If the hosts come back online, they will automatically re-enroll.</p>
        <p>
          To prevent re-enrollment, you can disable or uninstall osquery on
          these hosts.
        </p>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            onClick={onSubmit}
            variant="alert"
          >
            Delete
          </Button>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse-alert"
          >
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default DeleteHostModal;
