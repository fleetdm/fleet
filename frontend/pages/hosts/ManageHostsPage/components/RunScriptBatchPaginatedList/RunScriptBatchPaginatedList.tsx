import PaginatedList from "components/PaginatedList";
import { IScript } from "interfaces/script";
import React, { useCallback } from "react";

const baseClass = "run-script-batch-paginated-list";

interface IRunScriptBatchPaginatedList {
  onRunScript: (script: IScript) => IScript;
  isUpdating: boolean;
  scriptIdsHaveRunSinceOpen: Set<number>;
}

const PAGE_SIZE = 6;

const RunScriptBatchPaginatedList = ({
  onRunScript,
  isUpdating,
  scriptIdsHaveRunSinceOpen,
}: IRunScriptBatchPaginatedList) => {
  const fetchPage = useCallback((pageNumber: number) => {
    // TODO
    return Promise.resolve([] as IScript[]);
  }, []);

  const hasRun = useCallback(
    (script: IScript) => {
      return scriptIdsHaveRunSinceOpen.has(script.id);
    },
    [scriptIdsHaveRunSinceOpen]
  );

  return (
    <div className={`${baseClass}`}>
      <PaginatedList<IScript>
        // ref
        fetchPage={fetchPage}
        // TODO - make name more general and not necessarily apply only to checkboxes
        isSelected={hasRun}
        onToggleItem={onRunScript}
        pageSize={PAGE_SIZE}
        // heading prop, use for ordering by name?
        disabled={isUpdating}
      />
    </div>
  );
};

export default RunScriptBatchPaginatedList;
