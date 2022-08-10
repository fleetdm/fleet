import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";

const baseClass = "remove-scheduled-query-modal";

interface IRemoveScheduledQueryModalProps {
  isUpdatingScheduledQuery: boolean;
  onCancel: () => void;
  onSubmit: () => void;
}

const RemoveScheduledQueryModal = ({
  isUpdatingScheduledQuery,
  onCancel,
  onSubmit,
}: IRemoveScheduledQueryModalProps): JSX.Element => {
  return (
    <Modal
      title={"Remove scheduled query"}
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <div className={baseClass}>
        Are you sure you want to remove the selected queries from the schedule?
        <div className="modal-cta-wrap">
          <Button
            type="button"
            variant="alert"
            onClick={onSubmit}
            className="remove-loading"
            isLoading={isUpdatingScheduledQuery}
          >
            Remove
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default RemoveScheduledQueryModal;
