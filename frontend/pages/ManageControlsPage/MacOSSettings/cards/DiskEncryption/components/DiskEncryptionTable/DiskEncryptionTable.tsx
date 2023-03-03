import React from "react";

import { IDiskEncryptionStatusAggregate } from "interfaces/mdm";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";

import {
  generateTableHeaders,
  generateTableData,
} from "./DiskEncryptionTableConfig";

const baseClass = "disk-encryption-table";

interface IDiskEncryptionTableProps {
  aggregateData: IDiskEncryptionStatusAggregate;
}

const DEFAULT_SORT_HEADER = "hosts";
const DEFAULT_SORT_DIRECTION = "asc";

const DiskEncryptionTable = ({ aggregateData }: IDiskEncryptionTableProps) => {
  const tableHeaders = generateTableHeaders();
  const tableData = generateTableData(aggregateData);

  return (
    <div className={baseClass}>
      <TableContainer
        columns={tableHeaders}
        data={tableData}
        resultsTitle="" // TODO: make optional
        isLoading={false}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        defaultSortHeader={DEFAULT_SORT_HEADER}
        defaultSortDirection={DEFAULT_SORT_DIRECTION}
        disableTableHeader
        disablePagination
        disableCount
        emptyComponent={() => (
          <EmptyTable header="No Disk Encryption Status" info="test" />
        )}
      />
    </div>
  );
};

export default DiskEncryptionTable;
