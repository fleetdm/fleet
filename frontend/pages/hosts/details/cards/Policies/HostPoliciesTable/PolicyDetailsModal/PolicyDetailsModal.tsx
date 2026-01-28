import React from "react";
import Button from "components/buttons/Button";
import Modal from "components/Modal";

import { IHostPolicy } from "interfaces/policy";
import ClickableUrls from "components/ClickableUrls/ClickableUrls";

interface IPolicyDetailsProps {
  onCancel: () => void;
  policy: IHostPolicy | null;
  onResolveLater?: () => void;
}

const baseClass = "policy-details-modal";

const PolicyDetailsModal = ({
  onCancel,
  policy,
  onResolveLater,
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
          <Button onClick={onCancel}>Done</Button>
          {policy?.conditional_access_enabled &&
            policy.response === "fail" &&
            onResolveLater && (
              <Button onClick={onResolveLater} variant="inverse">
                Resolve later
              </Button>
            )}
        </div>
      </div>
    </Modal>
  );
};

export default PolicyDetailsModal;
