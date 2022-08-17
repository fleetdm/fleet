import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "remove-scheduled-query-modal";

interface IShowQueryModalProps {
  isLoading: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

const ShowQueryModal = ({
  isLoading,
  onCancel,
}: IShowQueryModalProps): JSX.Element => {
  return (
    <Modal title={"Query"} onExit={onCancel} className={baseClass}>
      {isLoading ? (
        <Spinner />
      ) : (
        <div className={baseClass}>
          TODO: Put the SQL editor here.
          <div className="modal-cta-wrap">
            <Button onClick={onCancel} variant="inverse-alert">
              Done
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
};

export default ShowQueryModal;
