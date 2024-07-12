import React from "react";

import { IOperatingSystemVersion } from "interfaces/operating_system";

import TableContainer from "components/TableContainer";

import { generateTableHeaders } from "./OSVersionTableConfig";
import OSVersionsEmptyState from "../OSVersionsEmptyState";
import { parseOSUpdatesCurrentVersionsQueryParams } from "../CurrentVersionSection/CurrentVersionSection";

const baseClass = "os-version-table";

interface IOSVersionTableProps {
  osVersionData: IOperatingSystemVersion[];
  currentTeamId: number;
  isLoading: boolean;
  queryParams: ReturnType<typeof parseOSUpdatesCurrentVersionsQueryParams>;
}

const OSVersionTable = ({
  osVersionData,
  currentTeamId,
  isLoading,
  queryParams,
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
        defaultSortHeader={queryParams.order_key}
        defaultSortDirection={queryParams.order_direction}
        defaultPageIndex={queryParams.page}
        disableTableHeader
        disableCount
        pageSize={queryParams.per_page}
      />
    </div>
  );
};

export default OSVersionTable;
