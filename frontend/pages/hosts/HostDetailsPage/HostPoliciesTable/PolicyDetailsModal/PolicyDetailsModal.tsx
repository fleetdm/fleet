import React from "react";
import Button from "components/buttons/Button";
import Modal from "components/modals/Modal";

import { IHostPolicy } from "interfaces/host_policy";

interface IPolicyDetailsProps {
  onCancel: () => void;
  policy: IHostPolicy | null;
}

const baseClass = "policy-details-modal";

const PolicyDetailsModal = ({
  onCancel,
  policy,
}: IPolicyDetailsProps): JSX.Element => {
  return (
    <Modal
      title={`${policy?.query_name || "Query name"}`}
      onExit={onCancel}
      className={baseClass}
    >
      <div className={`${baseClass}__modal-body`}>
        <p>{policy?.query_description}</p>
        <div className={`${baseClass}__btn-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onCancel}
            variant="brand"
          >
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default PolicyDetailsModal;
