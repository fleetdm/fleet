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
  currentTeamId?: number;
}

const DEFAULT_SORT_HEADER = "hosts";
const DEFAULT_SORT_DIRECTION = "asc";

const BootstrapPackageTable = ({
  currentTeamId,
}: IBootstrapPackageTableProps) => {
  // TODO: hook up API to get aggregate data
  // const { data, error } = useQuery<
  //   IBootstrapPackageAggregate,
  //   Error,
  //   IBootstrapPackageAggregate
  // >(
  //   ["bootstrap-package-summary", currentTeamId],
  //   () => mdmAPI.getDiskEncryptionAggregate(currentTeamId),
  //   {
  //     refetchOnWindowFocus: false,
  //     retry: false,
  //   }
  // );

  const data: IBootstrapPackageAggregate = {
    applied: 1,
    pending: 2,
    failed: 3,
  };

  const tableHeaders = generateTableHeaders();
  const tableData = generateTableData(data, currentTeamId);

  // if (error) {
  if (false) {
    return <DataError />;
  }

  if (!data) return null;

  return (
    <div className={baseClass}>
      <TableContainer
        columns={tableHeaders}
        data={tableData}
        resultsTitle=""
        isLoading={false}
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
