import React from "react";

import { IOperatingSystemVersion } from "interfaces/operating_system";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import { generateTableHeaders } from "./OSVersionTableConfig";

const baseClass = "os-version-table";

interface IOSVersionTableProps {
  osVersionData: IOperatingSystemVersion[];
  currentTeamId: number;
  isLoading: boolean;
}

const DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_SORT_DIRECTION = "desc";

const OSVersionEmptyState = () => {
  return (
    <EmptyTable
      className={`${baseClass}__empty-table`}
      header="No OS versions detected."
      info={
        <span>
          This report is updated every hour to protect
          <br /> the performance of your devices.
        </span>
      }
    />
  );
};

const OSVersionTable = ({
  osVersionData,
  currentTeamId,
  isLoading,
}: IOSVersionTableProps) => {
  const columns = generateTableHeaders();

  return (
    <div className={baseClass}>
      <TableContainer
        columns={columns}
        data={osVersionData}
        isLoading={isLoading}
        resultsTitle=""
        emptyComponent={OSVersionEmptyState}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        defaultSortHeader={DEFAULT_SORT_HEADER}
        defaultSortDirection={DEFAULT_SORT_DIRECTION}
        disableTableHeader
        disableCount
        disablePagination
      />
    </div>
  );
};

export default OSVersionTable;
