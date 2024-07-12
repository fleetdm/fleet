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
  currentVersionQueryParams: ReturnType<
    typeof parseOSUpdatesCurrentVersionsQueryParams
  >;
}

const OSVersionTable = ({
  osVersionData,
  currentTeamId,
  isLoading,
  currentVersionQueryParams,
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
        defaultSortHeader={currentVersionQueryParams.order_key}
        defaultSortDirection={currentVersionQueryParams.order_direction}
        defaultPageIndex={currentVersionQueryParams.page}
        disableTableHeader
        disableCount
        pageSize={currentVersionQueryParams.per_page}
      />
    </div>
  );
};

export default OSVersionTable;
