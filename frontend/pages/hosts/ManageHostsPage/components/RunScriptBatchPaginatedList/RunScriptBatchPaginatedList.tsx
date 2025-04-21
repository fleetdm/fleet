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
import { useQueryClient } from "react-query";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

const baseClass = "run-script-batch-paginated-list";

interface IRunScriptBatchPaginatedList {
  onRunScript: (script: IScript) => Promise<void>;
  isUpdating: boolean;
  teamId?: number;
}

const PAGE_SIZE = 6;

const RunScriptBatchPaginatedList = ({
  onRunScript: _onRunScript,
  isUpdating,
  teamId,
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
        // TODO - use `meta` to determine enable/disable of next/previous buttons
        return scripts;
      });
    },
    [queryClient, teamId]
  );

  const onRunScript = useCallback(
    (script: IScript) => {
      _onRunScript(script);
      // regardless of result of async trigger of batch script run, consider script "dirty" and
      // display "run again"
      return script;
    },
    [_onRunScript]
  );

  const onClickScriptRow = useCallback((script: IScript) => {
    // TODO - open script preview modal, maintain current modal state, incorporate into `renderItemRow`
  }, []);

  return (
    <div className={`${baseClass}`}>
      <PaginatedList<IScript>
        // ref
        fetchPage={fetchPage}
        // TODO - use dirtyItems to determine if script has been run
        onFireItemPrimaryAction={onRunScript}
        pageSize={PAGE_SIZE}
        disabled={isUpdating}
        // TODO - heading prop, use for ordering by name?
      />
    </div>
  );
};

export default RunScriptBatchPaginatedList;
