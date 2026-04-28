import React, { useCallback, useContext, useState } from "react";

import teamPoliciesAPI from "services/entities/team_policies";
import { NotificationContext } from "context/notification";

import { getErrorReason } from "interfaces/errors";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "add-patch-policy-modal";

const EXISTING_PATCH_POLICY_ERROR_MSG = `Couldn't add patch policy. Specified "patch_software_title_id" already has a policy with "type" set to "patch".`;

interface IAddPatchPolicyModal {
  softwareId: number;
  teamId: number;
  onExit: () => void;
  onSuccess: () => void;
}

const AddPatchPolicyModal = ({
  softwareId,
  teamId,
  onExit,
  onSuccess,
}: IAddPatchPolicyModal) => {
  const { renderFlash } = useContext(NotificationContext);
  const [isAddingPatchPolicy, setIsAddingPatchPolicy] = useState(false);

  const onAddPatchPolicy = useCallback(async () => {
    setIsAddingPatchPolicy(true);
    try {
      await teamPoliciesAPI.create({
        type: "patch",
        patch_software_title_id: softwareId,
        team_id: teamId,
      });
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
          This creates a read-only policy. Later, to enforce remediation, head
          to this policy&apos;s page.
        </p>
        <div className="modal-cta-wrap">
          <GitOpsModeTooltipWrapper
            entityType="software"
            renderChildren={(disableChildren) => (
              <Button
                onClick={onAddPatchPolicy}
                isLoading={isAddingPatchPolicy}
                disabled={disableChildren}
              >
                Add
              </Button>
            )}
          />
          <Button variant="inverse" onClick={onExit}>
            Cancel
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default AddPatchPolicyModal;
