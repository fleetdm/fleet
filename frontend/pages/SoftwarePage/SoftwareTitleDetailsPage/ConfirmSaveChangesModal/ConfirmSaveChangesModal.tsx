import React from "react";

import { InstallerType } from "interfaces/software";

import Button from "components/buttons/Button";
import Modal from "components/Modal";

const baseClass = "save-changes-modal";

export interface IConfirmSaveChangesModalProps {
  onSaveChanges: () => void;
  softwareInstallerName?: string;
  installerType: InstallerType;
  onClose: () => void;
  isLoading: boolean;
}

const ConfirmSaveChangesModal = ({
  onSaveChanges,
  softwareInstallerName,
  installerType,
  onClose,
  isLoading,
}: IConfirmSaveChangesModalProps) => {
  const warningText =
    installerType === "package" ? (
      <>
        <p>
          The changes you are making will cancel any pending installs and
          uninstalls
          {softwareInstallerName ? (
            <>
              {" "}
              for <b> {softwareInstallerName}</b>
            </>
          ) : (
            ""
          )}
          .
        </p>
        <p>
          Installs or uninstalls currently running on a host will still
          complete, but results won&apos;t appear in Fleet.
        </p>
        <p>You cannot undo this action.</p>
      </>
    ) : (
      <p>When targets change, pending installs will still complete.</p>
    );

  return (
    <Modal title="Save changes?" onExit={onClose}>
      <form className={`${baseClass}__form`}>
        {warningText}
        <div className="modal-cta-wrap">
          <Button
            type="button"
            onClick={onSaveChanges}
            className="save-loading"
            isLoading={isLoading}
          >
            Save
          </Button>
          <Button onClick={onClose} variant="inverse">
            Cancel
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export default ConfirmSaveChangesModal;
