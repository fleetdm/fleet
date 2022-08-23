import React from "react";

import FleetAce from "components/FleetAce";
import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "show-query-modal";

interface IShowQueryModalProps {
  onCancel: () => void;
  query?: string;
}

const ShowQueryModal = ({
  query,
  onCancel,
}: IShowQueryModalProps): JSX.Element => {
  return (
    <Modal
      title={"Query"}
      onExit={onCancel}
      onEnter={onCancel}
      className={baseClass}
    >
      <div className={baseClass}>
        <FleetAce
          value={query}
          name="Scheduled query"
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
