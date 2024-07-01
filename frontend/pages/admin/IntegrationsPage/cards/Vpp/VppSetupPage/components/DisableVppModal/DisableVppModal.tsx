import React, { useState } from "react";
import { noop } from "lodash";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "diable-vpp-modal";

interface IDisableVppModalProps {
  onExit: () => void;
}

const DisableVppModal = ({ onExit }: IDisableVppModalProps) => {
  const [isDisabling, setIsDisabling] = useState(false);

  const onDisableVpp = () => {
    // TODO: API integration
  };

  return (
    <Modal
      title="Disable Volume Purchasing Program (VPP)"
      onExit={noop}
      className={baseClass}
    >
      <>
        <p>
          Apps purchased in Apple Business Manager won&apos;t appear in Fleet.
          Apps won&apos;t be uninstalled from hosts. If you want to enable
          integration again, you&apos;ll have to upload a new content token.
        </p>
        <div className="modal-cta-wrap">
          <Button
            variant="alert"
            onClick={onDisableVpp}
            isLoading={isDisabling}
          >
            Disable
          </Button>
          <Button onClick={onExit} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DisableVppModal;
