import React, { useContext } from "react";
import { QueryContext } from "context/query";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import FleetAce from "components/FleetAce";

const baseClass = "show-query-modal";

interface IShowQueryModalProps {
  onCancel: () => void;
}

const ShowQueryModal = ({ onCancel }: IShowQueryModalProps): JSX.Element => {
  const { lastEditedQueryBody } = useContext(QueryContext);

  return (
    <Modal title={"Query"} onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__show-query-modal`}>
        <FleetAce
          value={lastEditedQueryBody}
          name="query editor"
          wrapperClassName={`${baseClass}__text-editor-wrapper`}
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
