import React from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

interface DeleteBootstrapPackageModalProps {
  onCancel: () => void;
  onDelete: () => void;
}

const baseClass = "delete-bootstrap-package-modal";

const DeleteBootstrapPackageModal = ({
  onCancel,
  onDelete,
}: DeleteBootstrapPackageModalProps) => {
  return (
    <Modal
      className={baseClass}
      title="Delete bootstrap package"
      onExit={onCancel}
      onEnter={() => onDelete()}
    >
      <>
        <p>
          Package won&apos;t be uninstalled from existing macOS hosts. Installs
          or uninstalls currently running on a host will still complete.
        </p>
        <p>
          Option to install Fleet&apos;s agent (fleetd) manually will be
          disabled, so agent will be installed automatically during automatic
          enollment of macOS hosts.
        </p>
        <div className="modal-cta-wrap">
          <Button type="button" onClick={() => onDelete()} variant="alert">
            Delete
          </Button>
          <Button onClick={onCancel} variant="inverse-alert">
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteBootstrapPackageModal;
