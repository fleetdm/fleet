import React, { useCallback, useContext, useState } from "react";

import softwareAPI from "services/entities/software";
import { NotificationContext } from "context/notification";

import { getErrorReason } from "interfaces/errors";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "delete-software-modal";

const DELETE_SW_USED_BY_POLICY_ERROR_MSG =
  "Couldn't delete. Policy automation uses this software. Please disable policy automation for this software and try again.";
const DELETE_SW_INSTALLED_DURING_SETUP_ERROR_MSG =
  "Couldn't delete. This software is installed when new Macs boot. Please remove software in Controls > Setup experience and try again.";

interface IDeleteSoftwareModalProps {
  softwareId: number;
  teamId: number;
  softwarePackageName?: string;
  onExit: () => void;
  onSuccess: () => void;
}

const DeleteSoftwareModal = ({
  softwareId,
  teamId,
  softwarePackageName,
  onExit,
  onSuccess,
}: IDeleteSoftwareModalProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isDeleting, setIsDeleting] = useState(false);

  const onDeleteSoftware = useCallback(async () => {
    setIsDeleting(true);
    try {
      await softwareAPI.deleteSoftwarePackage(softwareId, teamId);
      renderFlash("success", "Software deleted successfully!");
      onSuccess();
    } catch (error) {
      const reason = getErrorReason(error);
      if (reason.includes("Policy automation uses this software")) {
        renderFlash("error", DELETE_SW_USED_BY_POLICY_ERROR_MSG);
      } else if (reason.includes("This software is installed when")) {
        renderFlash("error", DELETE_SW_INSTALLED_DURING_SETUP_ERROR_MSG);
      } else {
        renderFlash("error", "Couldn't delete. Please try again.");
      }
    }
    setIsDeleting(false);
    onExit();
  }, [softwareId, teamId, renderFlash, onSuccess, onExit]);

  return (
    <Modal
      className={baseClass}
      title="Delete software"
      onExit={onExit}
      isContentDisabled={isDeleting}
    >
      <>
        <p>
          Software won&apos;t be uninstalled from existing hosts, but any
          pending pending installs and uninstalls{" "}
          {softwarePackageName ? (
            <>
              for <b> {softwarePackageName}</b>{" "}
            </>
          ) : (
            ""
          )}
          will be canceled.
        </p>
        <p>
          Installs or uninstalls currently running on a host will still
          complete, but results won&apos;t appear in Fleet.
        </p>
        <p>You cannot undo this action.</p>
        <div className="modal-cta-wrap">
          <Button
            variant="alert"
            onClick={onDeleteSoftware}
            isLoading={isDeleting}
          >
            Delete
          </Button>
          <Button variant="inverse-alert" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default DeleteSoftwareModal;
