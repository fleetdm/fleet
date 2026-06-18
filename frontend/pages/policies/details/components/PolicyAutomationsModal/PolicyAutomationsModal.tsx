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
        <div className={`${baseClass}__automations`}>
          <PolicyAutomationsList
            storedPolicy={storedPolicy}
            currentAutomatedPolicies={currentAutomatedPolicies}
            otherAutomationType={otherAutomationType}
          />
          <p className={`${baseClass}__footer-text`}>
            {storedPolicy.continuous_automations_enabled ? (
              <>
                Software and script automations run <b>every time</b> Fleet
                receives a failing response.
                <br />
                All other automations run on a host&apos;s first failure, or
                when a host&apos;s response changes from pass to fail.
              </>
            ) : (
              "Automations run on a host's first failure, or when a host's response changes from pass to fail."
            )}
          </p>
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onClose}>Done</Button>
        </div>
      </div>
    </Modal>
  );
};

export default PolicyAutomationsModal;
