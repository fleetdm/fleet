import React, { useCallback } from "react";
import { useQuery } from "react-query";
import { Row } from "react-table";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";

import { getPathWithQueryParams } from "utilities/url";

import diskEncryptionAPI, {
  IDiskEncryptionStatusAggregate,
  IDiskEncryptionSummaryResponse,
} from "services/entities/disk_encryption";
import { HOSTS_QUERY_PARAMS } from "services/entities/hosts";

import TableContainer from "components/TableContainer";
import EmptyState from "components/EmptyState";
import DataError from "components/DataError";

import {
  generateTableHeaders,
  generateTableData,
  IStatusCellValue,
} from "./DiskEncryptionTableConfig";

const baseClass = "disk-encryption-table";

interface IDiskEncryptionTableProps {
  platform: keyof IDiskEncryptionStatusAggregate;
  currentTeamId?: number;
  /** macOS enforce-on/escrow-off: hosts never send Fleet a key, so status
   * tooltips drop the key phrasing. */
  isMacOSEnforceOnly?: boolean;
  router: InjectedRouter;
}
interface IDiskEncryptionRowProps extends Row {
  original: {
    id?: number;
    status?: IStatusCellValue;
    teamId?: number;
  };
}

const DiskEncryptionTable = ({
  platform,
  currentTeamId,
  isMacOSEnforceOnly = false,
  router,
}: IDiskEncryptionTableProps) => {
  const {
    data: diskEncryptionStatusData,
    error: diskEncryptionStatusError,
  } = useQuery<IDiskEncryptionSummaryResponse, Error>(
    ["disk-encryption-summary", currentTeamId],
    () => diskEncryptionAPI.getDiskEncryptionSummary(currentTeamId),
    {
      refetchOnWindowFocus: false,
      retry: false,
    }
  );

  const onSelectSingleRow = useCallback(
    (row: IDiskEncryptionRowProps) => {
      const { status, teamId } = row.original;

      const queryParams = {
        [HOSTS_QUERY_PARAMS.DISK_ENCRYPTION]: status?.value,
        fleet_id: teamId,
      };
      const path = getPathWithQueryParams(PATHS.MANAGE_HOSTS, queryParams);

      router.push(path);
    },
    [router]
  );

  const tableHeaders = generateTableHeaders();
  const tableData = generateTableData(
    platform,
    diskEncryptionStatusData,
    currentTeamId,
    isMacOSEnforceOnly
  );

  if (diskEncryptionStatusError) {
    return <DataError />;
  }

  if (!diskEncryptionStatusData) return null;

  return (
    <div className={baseClass}>
      <TableContainer
        columnConfigs={tableHeaders}
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
          <EmptyState
            header="No disk encryption status"
            info="Expecting to status data? Try again in a few seconds as the system
              catches up."
          />
        )}
        // these 2 properties allow linking on click anywhere in the row
        disableMultiRowSelect
        onSelectSingleRow={onSelectSingleRow}
        hideFooter
      />
    </div>
  );
};

export default DiskEncryptionTable;
