import React from "react";

import Button from "components/buttons/Button";
import Modal from "components/Modal";
import { IPolicy, OtherAutomationType } from "interfaces/policy";
import {
  PolicyAutomationsList,
  mapAutomationRows,
} from "pages/policies/components";

const baseClass = "policy-reset-modal";

interface IPolicyResetModalProps {
  policy: IPolicy;
  /** Set when the reset is scoped to a single host. */
  host?: { displayName: string };
  currentAutomatedPolicies: number[];
  otherAutomationType?: OtherAutomationType;
  isResetting: boolean;
  onSubmit: () => void;
  onCancel: () => void;
}

const PolicyResetModal = ({
  policy,
  host,
  currentAutomatedPolicies,
  otherAutomationType,
  isResetting,
  onSubmit,
  onCancel,
}: IPolicyResetModalProps): JSX.Element => {
  // The modal opens in two modes. From the table's "Reset policy" button it's a
  // generic, policy-wide reset: no host name and no automations list. From a
  // specific automation run (the activity details modal) it's host-scoped: show
  // the host name and the automations that will re-run for that host. Scope
  // comes from `host`, not its name, since the display name can be empty.
  const isHostScoped = !!host;
  let target: React.ReactNode = "all hosts";
  if (host) {
    target = host.displayName ? <b>{host.displayName}</b> : "this host";
  }
  const hasAutomations =
    isHostScoped &&
    mapAutomationRows(policy, currentAutomatedPolicies, otherAutomationType)
      .length > 0;

  return (
    <Modal title="Reset policy" onExit={onCancel} className={baseClass}>
      <div className={`${baseClass}__modal-content`}>
        <p>
          Resetting this policy will clear pass/fail results for {target} until
          its next check in.
        </p>
        <div>
          {hasAutomations ? (
            <>
              <span>Automations will re-run if the host fails the policy:</span>
              <PolicyAutomationsList
                storedPolicy={policy}
                currentAutomatedPolicies={currentAutomatedPolicies}
                otherAutomationType={otherAutomationType}
              />
            </>
          ) : (
            <span>Automations will re-run if the host fails the policy.</span>
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
          <Button variant="secondary" onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default PolicyResetModal;
