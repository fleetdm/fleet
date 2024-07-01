import React, { useState } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

import VppSetupSteps from "../VppSetupSteps";

const baseClass = "renew-vpp-token-modal";

interface IRenewVppTokenModalProps {
  onExit: () => void;
}

const RenewVppTokenModal = ({ onExit }: IRenewVppTokenModalProps) => {
  const [isRenewing, setIsRenewing] = useState(false);

  const onRenewToken = () => {
    // TODO: API integration
  };

  return (
    <Modal title="Renew token" className={baseClass} onExit={onExit}>
      <>
        <VppSetupSteps />
        <div className="modal-cta-wrap">
          <Button variant="brand" onClick={onRenewToken} isLoading={isRenewing}>
            Renew token
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default RenewVppTokenModal;
