import React, { useContext } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import mdmAPI, { IDiskEncryptionSummaryResponse } from "services/entities/mdm";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import DataError from "components/DataError";

import {
  generateTableHeaders,
  generateTableData,
} from "./DiskEncryptionTableConfig";

const baseClass = "disk-encryption-table";

interface IDiskEncryptionTableProps {
  currentTeamId?: number;
}

const DiskEncryptionTable = ({ currentTeamId }: IDiskEncryptionTableProps) => {
  const { config } = useContext(AppContext);

  const {
    data: diskEncryptionStatusData,
    error: diskEncryptionStatusError,
  } = useQuery<IDiskEncryptionSummaryResponse, Error>(
    ["disk-encryption-summary", currentTeamId],
    () => mdmAPI.getDiskEncryptionSummary(currentTeamId),
    {
      refetchOnWindowFocus: false,
      retry: false,
    }
  );

  // TODO: WINDOWS FEATURE FLAG: remove this when windows feature flag is removed.
  // this is used to conditianlly show "View all hosts" link in table cells.
  const windowsFeatureFlagEnabled = config?.mdm_enabled ?? false;
  const tableHeaders = generateTableHeaders(windowsFeatureFlagEnabled);
  const tableData = generateTableData(
    windowsFeatureFlagEnabled,
    diskEncryptionStatusData,
    currentTeamId
  );

  if (diskEncryptionStatusError) {
    return <DataError />;
  }

  if (!diskEncryptionStatusData) return null;

  return (
    <div className={baseClass}>
      <TableContainer
        columns={tableHeaders}
        data={tableData}
        resultsTitle="" // TODO: make optional
        isLoading={false}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        manualSortBy
        disableTableHeader
        disablePagination
        disableCount
        emptyComponent={() => (
          <EmptyTable
            header="No Disk Encryption Status"
            info="Expecting to status data? Try again in a few seconds as the system
              catches up."
          />
        )}
      />
    </div>
  );
};

export default DiskEncryptionTable;
