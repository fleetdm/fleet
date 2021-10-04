import React from "react";

import Modal from "components/modals/Modal";
import Button from "components/buttons/Button";
import { IScheduledQuery } from "interfaces/scheduled_query";

const baseClass = "remove-pack-query-modal";

interface IRemovePackQueryModalProps {
  onCancel: () => void;
  onSubmit: () => void;
  selectedQuery?: IScheduledQuery;
  selectedQueryIds: number[];
}

const RemovePackQueryModal = (
  props: IRemovePackQueryModalProps
): JSX.Element => {
  const { onCancel, onSubmit, selectedQuery, selectedQueryIds } = props;
  console.log(selectedQuery);
  console.log(selectedQueryIds);
  const queryOrQueries =
    selectedQuery || selectedQueryIds?.length === 1 ? "query" : "queries";
  return (
    <Modal title={"Remove queries"} onExit={onCancel} className={baseClass}>
      <div className={baseClass}>
        Are you sure you want to remove the selected {queryOrQueries} from your
        pack?
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="inverse"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__btn`}
            type="button"
            variant="alert"
            onClick={onSubmit}
          >
            Remove
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default RemovePackQueryModal;
