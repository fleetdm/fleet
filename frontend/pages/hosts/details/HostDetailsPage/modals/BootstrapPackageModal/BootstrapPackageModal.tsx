import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "bootstrap-package-modal";

interface IBootstrapPackageModalProps {
  packageName: string;
  details: string;
  onClose: () => void;
}

const BootstrapPackageModal = ({
  packageName,
  details,
  onClose,
}: IBootstrapPackageModalProps) => {
  return (
    <Modal
      title="Bootstrap package"
      onExit={onClose}
      onEnter={onClose}
      className={baseClass}
    >
      <>
        <p className={`${baseClass}__package-name`}>
          The <b>{packageName}</b> failed to install with the following error:
        </p>
        <p className={`${baseClass}__details`}>{details}</p>

        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onClose}
            variant="brand"
            className="delete-loading"
          >
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default BootstrapPackageModal;
