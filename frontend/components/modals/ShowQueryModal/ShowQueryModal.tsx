import React, { useContext } from "react";
import { QueryContext } from "context/query";
import { PolicyContext } from "context/policy";

import FleetAce from "components/FleetAce";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "show-query-modal";

interface IShowQueryModalProps {
  onCancel: () => void;
  query?: string;
  liveQuery?: boolean;
  livePolicy?: boolean;
}

const ShowQueryModal = ({
  query,
  liveQuery,
  livePolicy,
  onCancel,
}: IShowQueryModalProps): JSX.Element => {
  const { lastEditedQueryBody } = useContext(QueryContext);
  const { lastEditedQueryBody: lastEditedPolicyQueryBody } = useContext(
    PolicyContext
  );

  const querySql = () => {
    if (liveQuery) {
      return lastEditedQueryBody;
    }
    if (livePolicy) {
      return lastEditedPolicyQueryBody;
    }
    return query;
  };

  return (
    <Modal
      title={"Query"}
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <div className={baseClass}>
        <FleetAce
          value={querySql()}
          name="Query"
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
          wrapEnabled
          readOnly
        />
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ShowQueryModal;
