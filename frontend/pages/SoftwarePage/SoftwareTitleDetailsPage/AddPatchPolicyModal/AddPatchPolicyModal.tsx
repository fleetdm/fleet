import React, { useCallback, useContext, useState } from "react";

import softwareAPI from "services/entities/software";
import { NotificationContext } from "context/notification";

import { getErrorReason } from "interfaces/errors";

import Modal from "components/Modal";
import Button from "components/buttons/Button";

const baseClass = "add-patch-policy-modal";

const EXISTING_PATCH_POLICY_ERROR_MSG = `Couldn't add patch policy. Specified "patch_software_title_id" already has a policy with "type" set to "patch".`;

interface IAddPatchPolicyModal {
  softwareId: number;
  teamId: number;
  onExit: () => void;
  onSuccess: () => void;
  gitOpsModeEnabled?: boolean;
}

const AddPatchPolicyModal = ({
  softwareId,
  teamId,
  onExit,
  onSuccess,
  gitOpsModeEnabled,
}: IAddPatchPolicyModal) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isAddingPatchPolicy, setIsAddingPatchPolicy] = useState(false);

  const onAddPatchPolicy = useCallback(async () => {
    setIsAddingPatchPolicy(true);
    try {
      await softwareAPI.addPatchPolicy(softwareId, teamId);
      renderFlash("success", "Successfully added patch policy.");
      onSuccess();
    } catch (error) {
      const reason = getErrorReason(error);
      if (reason.includes("already has a policy")) {
        renderFlash("error", EXISTING_PATCH_POLICY_ERROR_MSG);
      }
      renderFlash("error", "Couldn't add patch policy. Please try again.");
    }
    setIsAddingPatchPolicy(false);
    onExit();
  }, [softwareId, teamId, renderFlash, onSuccess, onExit]);

  return (
    <Modal
      className={baseClass}
      title="Add patch policy"
      onExit={onExit}
      isContentDisabled={isAddingPatchPolicy}
    >
      <>
        <p>
          Later you can add software automation for this software on{" "}
          <strong>
            Policies &gt; Manage automations &gt; Install software
          </strong>
          .
        </p>
        <div className="modal-cta-wrap">
          {/* TODO: Disable here in gitops mode? */}
          <Button onClick={onAddPatchPolicy} isLoading={isAddingPatchPolicy}>
            Add
          </Button>
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default AddPatchPolicyModal;
