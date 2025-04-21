import React, { useState } from "react";

import { IScript } from "interfaces/script";

import Modal from "components/Modal";

import RunScriptBatchPaginatedList from "../RunScriptBatchPaginatedList";

const baseClass = "run-script-batch-modal";

interface IRunScriptBatchModal {
  selectedHostsCount: number;
  onRunScript: (script: IScript) => Promise<void>;
  onCancel: () => void;
  isUpdating: boolean;
  teamId?: number;
}

const RunScriptBatchModal = ({
  selectedHostsCount,
  onRunScript,
  onCancel,
  isUpdating,
  teamId,
}: IRunScriptBatchModal) => {
  return (
    <Modal
      title="Run script"
      onExit={onCancel}
      onEnter={onCancel}
      className={`${baseClass}`}
      isLoading={isUpdating}
    >
      <>
        <p>
          Will run on <b>{selectedHostsCount} hosts</b>. You can see individual
          script results on the host details page.
        </p>
        <RunScriptBatchPaginatedList
          onRunScript={onRunScript}
          isUpdating={isUpdating}
          teamId={teamId}
        />
      </>
    </Modal>
  );
};

export default RunScriptBatchModal;
