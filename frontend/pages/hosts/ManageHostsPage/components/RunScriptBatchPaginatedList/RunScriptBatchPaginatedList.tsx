import { createMockScript } from "__mocks__/scriptMock";
import scriptAPI, {
  IListScriptsQueryKey,
  IScriptsResponse,
} from "services/entities/scripts";
import PaginatedList from "components/PaginatedList";
import { IScript } from "interfaces/script";
import {
  APP_CONTEXT_ALL_TEAMS_ID as APP_CONTEXT_NO_TEAM,
  APP_CONTEXT_NO_TEAM_ID,
} from "interfaces/team";
import React, { useState, useCallback } from "react";
import { useQuery, useQueryClient } from "react-query";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import ActionButton from "components/TableContainer/DataTable/ActionButton";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

const baseClass = "run-script-batch-paginated-list";

interface IPaginatedListScript extends IScript {
  hasRun?: boolean;
}

interface IRunScriptBatchPaginatedList {
  onRunScript: (script: IPaginatedListScript) => Promise<void>;
  isUpdating: boolean;
  teamId?: number;
  scriptCount: number;
}

const PAGE_SIZE = 6;

const RunScriptBatchPaginatedList = ({
  onRunScript: _onRunScript,
  isUpdating,
  teamId,
  scriptCount,
}: IRunScriptBatchPaginatedList) => {
  // const fetchPage = useCallback((pageNumber: number) => {
  // TODO - Scott's implementation on PoliciesPaginatedList uses UseQuery underlying query client,
  // but seems like simplest and most Fleet-idiomatic approach would be to pass current scripts as
  // a prop, and have the child just set the page number to trigger updates

  // return Promise.resolve([createMockScript()]);
  // }, []);

  // Fetch a single page of scripts.
  const queryClient = useQueryClient();

  const fetchPage = useCallback(
    (pageNumber: number) => {
      // scripts not supported for All teams
      const fetchPromise = queryClient.fetchQuery(
        [
          {
            scope: "scripts",
            // TODO - check this covers No team correctly
            team_id: teamId,
            page: pageNumber,
            perPage: PAGE_SIZE,
            // TODO - allow changing order direction
          },
        ],
        ({ queryKey }) => {
          return scriptAPI.getScripts(queryKey[0]);
        }
      );

      return fetchPromise.then(({ scripts, meta }: IScriptsResponse) => {
        // TODO - use `meta` to determine enable/disable of next/previous buttons? currently
        // calculated within paginatedlist
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
      // regardless of result of async trigger of batch script run, consider script "dirty" and
      // display "run again"
      onChange({ hasRun: true, ...script });
      return script;
    },
    [_onRunScript]
  );

  const onClickScriptRow = useCallback((script: IPaginatedListScript) => {
    // TODO - open script preview modal, maintain current modal state, incorporate into `renderItemRow`
  }, []);

  const toggleScriptPreview = useCallback((script: IPaginatedListScript) => {
    // TODO - call for script details, render in preview modal
    alert("script contents");
    return script;
  }, []);

  const renderScriptRow = (
    script: IPaginatedListScript,
    onChange: (script: IPaginatedListScript) => void
  ) => (
    <>
      <span>{script.name}</span>
      {/* TODO - only show button on over */}
      <Button
        variant="text-icon"
        // prevent filling in icon background on hover
        iconStroke
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

  // TODO - implement grayed overlay with Spinner when `isUpdating`
  return (
    <div className={`${baseClass}`}>
      <PaginatedList<IPaginatedListScript>
        renderItemRow={renderScriptRow}
        count={scriptCount}
        fetchPage={fetchPage}
        onClickRow={toggleScriptPreview}
        // TODO - more elegant way?
        setDirtyOnClickRow={false}
        pageSize={PAGE_SIZE}
        disabled={isUpdating}
        // TODO - heading prop, use for ordering by name?
        useCheckBoxes={false}
      />
    </div>
  );
};

export default RunScriptBatchPaginatedList;
