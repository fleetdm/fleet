import React from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";

import { IHostPolicy } from "interfaces/policy";
import ClickableUrls from "components/ClickableUrls/ClickableUrls";

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
      onEnter={onCancel}
      className={baseClass}
    >
      <div className={`${baseClass}__body`}>
        <span className={`${baseClass}__description`}>
          {policy?.description}
        </span>
        {policy?.resolution && (
          <div className={`${baseClass}__resolution`}>
            <span className={`${baseClass}__resolution-header`}>Resolve:</span>
            {policy?.resolution && <ClickableUrls text={policy?.resolution} />}
          </div>
        )}
        <div className="modal-cta-wrap">
          <Button onClick={onCancel} variant="brand">
            Done
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default PolicyDetailsModal;
