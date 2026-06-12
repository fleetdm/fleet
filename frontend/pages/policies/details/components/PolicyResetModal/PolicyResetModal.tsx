import React from "react";

import { IPolicy, OtherAutomationType } from "interfaces/policy";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { PolicyAutomationsList } from "pages/policies/components";
import { mapAutomationRows } from "pages/policies/components/PolicyAutomationsList/PolicyAutomationsList";

const baseClass = "policy-reset-modal";

interface IPolicyResetModalProps {
  policy?: IPolicy;
  hostDisplayName?: string;
  currentAutomatedPolicies: number[];
  otherAutomationType?: OtherAutomationType;
  isResetting: boolean;
  onSubmit: () => void;
  onCancel: () => void;
}

const PolicyResetModal = ({
  policy,
  hostDisplayName,
  currentAutomatedPolicies,
  otherAutomationType,
  isResetting,
  onSubmit,
  onCancel,
}: IPolicyResetModalProps): JSX.Element => {
  const hasAutomations =
    !!policy &&
    mapAutomationRows(policy, currentAutomatedPolicies, otherAutomationType)
      .length > 0;

  return (
    <Modal title="Reset policy" onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__modal-content`}>
        <p>
          Resetting this policy will clear pass/fail results for{" "}
          {hostDisplayName ? <b>{hostDisplayName}</b> : "all hosts"} until its
          next check in.
        </p>
        <div>
          <span>
            {hasAutomations
              ? "Automations will re-run if the host fails the policy:"
              : "Automations will re-run if the host fails the policy."}
          </span>
          {policy && hasAutomations && (
            <PolicyAutomationsList
              storedPolicy={policy}
              currentAutomatedPolicies={currentAutomatedPolicies}
              otherAutomationType={otherAutomationType}
            />
          )}
        </div>
        <div className="modal-cta-wrap">
          <Button
            onClick={onSubmit}
            isLoading={isResetting}
            disabled={isResetting}
          >
            Reset
          </Button>
          <Button variant="inverse" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default PolicyResetModal;
