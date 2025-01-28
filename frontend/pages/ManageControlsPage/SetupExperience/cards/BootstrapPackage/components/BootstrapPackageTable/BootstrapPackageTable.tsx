import React from "react";
import { useQuery } from "react-query";

import { IBootstrapPackageAggregate } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";

import DataError from "components/DataError";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import {
  COLUMN_CONFIGS,
  generateTableData,
} from "./BootstrapPackageTableConfig";

const baseClass = "bootstrap-package-table";

interface IBootstrapPackageTableProps {
  currentTeamId: number;
}

const DEFAULT_SORT_HEADER = "hosts";
const DEFAULT_SORT_DIRECTION = "asc";

const BootstrapPackageTable = ({
  currentTeamId,
}: IBootstrapPackageTableProps) => {
  const { data: bootstrapPackageAggregate, isLoading, isError } = useQuery<
    IBootstrapPackageAggregate,
    Error,
    IBootstrapPackageAggregate
  >(
    ["bootstrap-package-summary", currentTeamId],
    () => mdmAPI.getBootstrapPackageAggregate(currentTeamId),
    {
      retry: false,
      refetchOnWindowFocus: false,
    }
  );

  const tableData = generateTableData(bootstrapPackageAggregate, currentTeamId);

  if (isError) return <DataError />;

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={COLUMN_CONFIGS}
        data={tableData}
        resultsTitle=""
        isLoading={isLoading}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        defaultSortHeader={DEFAULT_SORT_HEADER}
        defaultSortDirection={DEFAULT_SORT_DIRECTION}
        disableTableHeader
        disablePagination
        disableCount
        emptyComponent={() => (
          <EmptyTable
            header="No bootstrap package status"
            info="Expecting to status data? Try again in a few seconds as the system
              catches up."
          />
        )}
      />
    </div>
  );
};

export default BootstrapPackageTable;
