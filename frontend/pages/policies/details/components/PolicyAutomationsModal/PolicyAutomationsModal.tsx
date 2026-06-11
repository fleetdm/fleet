import React from "react";

import { IPolicy, OtherAutomationType } from "interfaces/policy";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import { PolicyAutomationsList } from "pages/policies/components";

const baseClass = "policy-automations-modal";

interface IPolicyAutomationsModalProps {
  storedPolicy: IPolicy;
  currentAutomatedPolicies: number[];
  otherAutomationType?: OtherAutomationType;
  onClose: () => void;
}

const PolicyAutomationsModal = ({
  storedPolicy,
  currentAutomatedPolicies,
  otherAutomationType,
  onClose,
}: IPolicyAutomationsModalProps): JSX.Element => {
  return (
    <Modal
      title="Automations"
      onExit={onClose}
      onEnter={onClose}
      className={baseClass}
    >
      <div className={baseClass}>
        <PolicyAutomationsList
          storedPolicy={storedPolicy}
          currentAutomatedPolicies={currentAutomatedPolicies}
          otherAutomationType={otherAutomationType}
        />
        <div className="modal-cta-wrap">
          <Button onClick={onClose}>Done</Button>
        </div>
      </div>
    </Modal>
  );
};

export default PolicyAutomationsModal;
