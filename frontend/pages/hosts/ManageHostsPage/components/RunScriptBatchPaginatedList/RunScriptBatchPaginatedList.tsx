import React, { useCallback, useState } from "react";
import { useQuery } from "react-query";

import scriptAPI, { IScriptsResponse } from "services/entities/scripts";

import { addTeamIdCriteria, IScript } from "interfaces/script";

import PaginatedList from "components/PaginatedList";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "run-script-batch-paginated-list";

export interface IPaginatedListScript extends IScript {
  hasRun?: boolean;
}

interface IRunScriptBatchPaginatedList {
  onRunScript: (script: IPaginatedListScript) => void;
  isUpdating: boolean;
  teamId: number;
  isFreeTier?: boolean;
  scriptCount: number;
  setScriptForDetails: (script: IPaginatedListScript) => void;
}

export const SCRIPT_BATCH_PAGE_SIZE = 6;

const RunScriptBatchPaginatedList = ({
  onRunScript: _onRunScript,
  isUpdating,
  isFreeTier,
  teamId,
  scriptCount,
  setScriptForDetails,
}: IRunScriptBatchPaginatedList) => {
  const [pageNumber, setPageNumber] = useState(0);

  // Fetch a single page of scripts.
  const queryKey = addTeamIdCriteria(
    {
      scope: "scripts",
      page: pageNumber,
      per_page: SCRIPT_BATCH_PAGE_SIZE,
    },
    teamId,
    isFreeTier
  );

  const { data, isFetching } = useQuery<
    IScriptsResponse,
    Error,
    IScriptsResponse
  >([queryKey], () => {
    return scriptAPI.getScripts(queryKey);
  });

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
        variant="inverse"
        iconStroke={!script.hasRun}
        onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
          e.stopPropagation();
          onRunScript(script, onChange);
        }}
      >
        {script.hasRun ? (
          <>
            Run again
            <Icon name="refresh" color="ui-fleet-black-75" />
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
        data={data?.scripts || []}
        onChangePage={setPageNumber}
        currentPage={pageNumber}
        onClickRow={onClickScriptRow}
        setDirtyOnClickRow={false}
        pageSize={SCRIPT_BATCH_PAGE_SIZE}
        disabled={isUpdating}
        useCheckBoxes={false}
        isLoading={isUpdating || isFetching}
      />
    </div>
  );
};

export default RunScriptBatchPaginatedList;
