import React from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";

import { IHostPolicy } from "interfaces/policy";

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
      title={`${policy?.name || "Policy name"}`}
      onExit={onCancel}
      className={baseClass}
    >
      <div className={`${baseClass}__modal-body`}>
        <p>{policy?.description}</p>
        {policy?.resolution && (
          <div className={`${baseClass}__resolution`}>
            <span className={`${baseClass}__resolve-header`}> Resolve:</span>
            <br />
            {policy?.resolution}
          </div>
        )}
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
