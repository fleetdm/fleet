import React from "react";

import { IOperatingSystemVersion } from "interfaces/operating_system";

import TableContainer from "components/TableContainer";

import { generateTableHeaders } from "./OSVersionTableConfig";
import OSVersionsEmptyState from "../OSVersionsEmptyState";

const baseClass = "os-version-table";

interface IOSVersionTableProps {
  osVersionData: IOperatingSystemVersion[];
  currentTeamId: number;
  isLoading: boolean;
}

const DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_SORT_DIRECTION = "desc";

const OSVersionTable = ({
  osVersionData,
  currentTeamId,
  isLoading,
}: IOSVersionTableProps) => {
  const columns = generateTableHeaders(currentTeamId);

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={columns}
        data={osVersionData}
        isLoading={isLoading}
        resultsTitle=""
        emptyComponent={OSVersionsEmptyState}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        defaultSortHeader={DEFAULT_SORT_HEADER}
        defaultSortDirection={DEFAULT_SORT_DIRECTION}
        disableTableHeader
        disableCount
        pageSize={8}
        isClientSidePagination
      />
    </div>
  );
};

export default OSVersionTable;
