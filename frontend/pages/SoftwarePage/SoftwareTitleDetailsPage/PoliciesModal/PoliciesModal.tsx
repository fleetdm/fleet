import React from "react";

import { ISoftwareInstallPolicyUI } from "interfaces/software";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InstallerPoliciesTable from "../SoftwareInstallerCard/InstallerPoliciesTable";

const baseClass = "policies-modal";

interface IPoliciesModalProps {
  policies: ISoftwareInstallPolicyUI[];
  teamId?: number;
  onExit: () => void;
}

// TODO: Marko to update the design of this modal — layout, plus a
// description that accounts for patch policies that may not auto-update.
const PoliciesModal = ({ policies, teamId, onExit }: IPoliciesModalProps) => {
  return (
    <Modal className={baseClass} title="Policies" onExit={onExit}>
      <>
        {policies.length === 0 ? (
          <p className={`${baseClass}__empty`}>
            No policies are linked to this software.
          </p>
        ) : (
          <InstallerPoliciesTable
            teamId={teamId}
            policies={policies}
            hideCount
          />
        )}
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default PoliciesModal;
