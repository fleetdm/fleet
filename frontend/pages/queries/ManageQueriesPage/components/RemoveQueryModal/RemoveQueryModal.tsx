import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "remove-query-modal";

interface IRemoveQueryModalProps {
  isLoading: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

const RemoveQueryModal = ({
  isLoading,
  onCancel,
  onSubmit,
}: IRemoveQueryModalProps): JSX.Element => {
  return (
    <Modal
      title={"Delete query"}
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <>
        {isLoading ? (
          <Spinner />
        ) : (
          <div className={baseClass}>
            Are you sure you want to delete the selected queries?
            <div className="modal-cta-wrap">
              <Button onClick={onCancel} variant="inverse-alert">
                Cancel
              </Button>
              <Button type="button" variant="alert" onClick={onSubmit}>
                Delete
              </Button>
            </div>
          </div>
        )}
      </>
    </Modal>
  );
};

export default RemoveQueryModal;
