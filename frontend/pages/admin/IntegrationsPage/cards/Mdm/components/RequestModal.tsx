import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface IRequestModal {
  onCancel: () => void;
  onRequest: () => void;
}

const baseClass = "request-modal";

const RequestModal = ({ onCancel, onRequest }: IRequestModal): JSX.Element => {
  return (
    <Modal title="Request" onExit={onCancel} className={baseClass}>
      <>
        Cool beans
        <div className="modal-cta-wrap">
          <Button onClick={onRequest} variant="brand">
            Request
          </Button>
          <Button onClick={onCancel} variant="inverse">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default RequestModal;
