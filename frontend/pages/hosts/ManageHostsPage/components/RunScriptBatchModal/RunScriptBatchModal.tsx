import React, { useState } from "react";
import { useQuery } from "react-query";

import { IScript } from "interfaces/script";

import Modal from "components/Modal";

import scriptAPI, {
  IListScriptsQueryKey,
  IScriptsResponse,
} from "services/entities/scripts";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import Spinner from "components/Spinner";
import EmptyTable from "components/EmptyTable";
import Button from "components/buttons/Button";

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
  // just used to get total number of scripts, could be optimized by implementing a dedicated scriptsCount endpoint
  const { data: scripts } = useQuery<
    IScriptsResponse,
    Error,
    IScript[],
    IListScriptsQueryKey[]
  >(
    [
      {
        scope: "scripts",
        team_id: teamId,
      },
    ],
    ({ queryKey }) => {
      return scriptAPI.getScripts(queryKey[0]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      keepPreviousData: true,
      select: (data) => data.scripts || [],
    }
  );

  const renderModalContent = () => {
    if (scripts === undefined) {
      return <Spinner />;
    }
    if (!scripts.length) {
      return (
        <EmptyTable
          header="No scripts available for this team"
          info={
            <>
              You can add saved scripts{" "}
              <a href={`/controls/scripts?team_id=${teamId}`}>here</a>.
            </>
          }
        />
      );
    }
    return (
      <>
        <p>
          Will run on <b>{selectedHostsCount} hosts</b>. You can see individual
          script results on the host details page.
        </p>
        <RunScriptBatchPaginatedList
          onRunScript={onRunScript}
          isUpdating={isUpdating}
          teamId={teamId}
          scriptCount={scripts.length}
        />
      </>
    );
  };

  return (
    <Modal
      title="Run script"
      onExit={onCancel}
      onEnter={onCancel}
      className={`${baseClass}`}
      isLoading={isUpdating}
    >
      <>
        {renderModalContent()}
        <div className="modal-cta-wrap">
          <Button onClick={onCancel}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default RunScriptBatchModal;
