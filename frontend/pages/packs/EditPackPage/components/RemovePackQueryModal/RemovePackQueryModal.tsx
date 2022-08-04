import React, { useEffect } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { IScheduledQuery } from "interfaces/scheduled_query";

const baseClass = "remove-pack-query-modal";

interface IRemovePackQueryModalProps {
  onCancel: () => void;
  onSubmit: () => void;
  selectedQuery?: IScheduledQuery;
  selectedQueryIds: number[];
}

const RemovePackQueryModal = ({
  onCancel,
  onSubmit,
  selectedQuery,
  selectedQueryIds,
}: IRemovePackQueryModalProps): JSX.Element => {
  const queryOrQueries =
    selectedQuery || selectedQueryIds?.length === 1 ? "query" : "queries";
  return (
    <Modal
      title={"Remove queries"}
      onExit={onCancel}
      onEnter={onSubmit}
      className={baseClass}
    >
      <div className={baseClass}>
        Are you sure you want to remove the selected {queryOrQueries} from your
        pack?
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
          <Button type="button" variant="alert" onClick={onSubmit}>
            Remove
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default RemovePackQueryModal;
