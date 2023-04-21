import React from "react";
import { useQuery } from "react-query";

import { IBootstrapPackageAggregate } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";

import DataError from "components/DataError";
import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import {
  generateTableData,
  generateTableHeaders,
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
  const { data: bootstrapPackageAggregatem, isLoading, isError } = useQuery<
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

  const tableHeaders = generateTableHeaders();
  const tableData = generateTableData(
    bootstrapPackageAggregatem,
    currentTeamId
  );

  if (isError) return <DataError />;

  return (
    <div className={baseClass}>
      <TableContainer
        columns={tableHeaders}
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
            header="No Bootstrap Package Status"
            info="Expecting to status data? Try again in a few seconds as the system
              catches up."
          />
        )}
      />
    </div>
  );
};

export default BootstrapPackageTable;
