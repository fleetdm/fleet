import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "install-all-in-category-modal";

export interface IInstallAllInCategoryModalProps {
  count: number;
  isSubmitting?: boolean;
  onConfirm: () => void;
  onExit: () => void;
}

const InstallAllInCategoryModal = ({
  count,
  isSubmitting,
  onConfirm,
  onExit,
}: IInstallAllInCategoryModalProps) => {
  return (
    <Modal
      className={baseClass}
      title="Install all"
      onExit={onExit}
      isContentDisabled={isSubmitting}
    >
      <>
        <p>
          {count} new app{count === 1 ? "" : "s"} will be installed. Apps
          already installed won&apos;t be re-installed.
        </p>
        <div className="modal-cta-wrap">
          <Button onClick={onConfirm} isLoading={isSubmitting}>
            Install all
          </Button>
          <Button variant="secondary" onClick={onExit} disabled={isSubmitting}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default InstallAllInCategoryModal;
