import React, { useCallback } from "react";
import { useQueryClient } from "react-query";

import scriptAPI, { IScriptsResponse } from "services/entities/scripts";

import { IScript } from "interfaces/script";

import PaginatedList from "components/PaginatedList";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "run-script-batch-paginated-list";

export interface IPaginatedListScript extends IScript {
  hasRun?: boolean;
}

interface IRunScriptBatchPaginatedList {
  onRunScript: (script: IPaginatedListScript) => Promise<void>;
  isUpdating: boolean;
  teamId: number;
  scriptCount: number;
  setScriptForDetails: (script: IPaginatedListScript) => void;
}

export const SCRIPT_BATCH_PAGE_SIZE = 6;

const RunScriptBatchPaginatedList = ({
  onRunScript: _onRunScript,
  isUpdating,
  teamId,
  scriptCount,
  setScriptForDetails,
}: IRunScriptBatchPaginatedList) => {
  // Fetch a single page of scripts.
  const queryClient = useQueryClient();

  const fetchPage = useCallback(
    (pageNumber: number) => {
      // scripts not supported for All teams
      const fetchPromise = queryClient.fetchQuery(
        [
          {
            scope: "scripts",
            team_id: teamId,
            page: pageNumber,
            per_page: SCRIPT_BATCH_PAGE_SIZE,
          },
        ],
        ({ queryKey }) => {
          return scriptAPI.getScripts(queryKey[0]);
        }
      );

      return fetchPromise.then(({ scripts }: IScriptsResponse) => {
        return scripts || [];
      });
    },
    [queryClient, teamId]
  );

  const onRunScript = useCallback(
    (
      script: IPaginatedListScript,
      onChange: (script: IPaginatedListScript) => void
    ) => {
      _onRunScript(script);
      onChange({ hasRun: true, ...script });
      return script;
    },
    [_onRunScript]
  );

  const onClickScriptRow = useCallback((script: IPaginatedListScript) => {
    setScriptForDetails(script);
    return script;
  }, []);

  const renderScriptRow = (
    script: IPaginatedListScript,
    onChange: (script: IPaginatedListScript) => void
  ) => (
    <>
      <a>{script.name}</a>
      <Button
        variant="text-icon"
        iconStroke={!script.hasRun}
        onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
          e.stopPropagation();
          onRunScript(script, onChange);
        }}
      >
        {script.hasRun ? (
          <>
            Run again
            <Icon name="refresh" color="core-fleet-blue" />
          </>
        ) : (
          <>
            Run script
            <Icon name="run" />
          </>
        )}
      </Button>
    </>
  );

  return (
    <div className={`${baseClass}`}>
      <PaginatedList<IPaginatedListScript>
        renderItemRow={renderScriptRow}
        count={scriptCount}
        fetchPage={fetchPage}
        onClickRow={onClickScriptRow}
        setDirtyOnClickRow={false}
        pageSize={SCRIPT_BATCH_PAGE_SIZE}
        disabled={isUpdating}
        useCheckBoxes={false}
        ancestralUpdating={isUpdating}
      />
    </div>
  );
};

export default RunScriptBatchPaginatedList;
