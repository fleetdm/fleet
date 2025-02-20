import React, { useCallback } from "react";
import { useQuery } from "react-query";
import { Row } from "react-table";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";

import { buildQueryStringFromParams } from "utilities/url";

import diskEncryptionAPI, {
  IDiskEncryptionSummaryResponse,
} from "services/entities/disk_encryption";
import { HOSTS_QUERY_PARAMS } from "services/entities/hosts";

import TableContainer from "components/TableContainer";
import EmptyTable from "components/EmptyTable";
import DataError from "components/DataError";

import {
  generateTableHeaders,
  generateTableData,
  IStatusCellValue,
} from "./DiskEncryptionTableConfig";

const baseClass = "disk-encryption-table";

interface IDiskEncryptionTableProps {
  currentTeamId?: number;
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
  currentTeamId,
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
        team_id: teamId,
      };
      const endpoint = PATHS.MANAGE_HOSTS;
      const path = `${endpoint}?${buildQueryStringFromParams(queryParams)}`;
      router.push(path);
    },
    [router]
  );

  const tableHeaders = generateTableHeaders();
  const tableData = generateTableData(diskEncryptionStatusData, currentTeamId);

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
          <EmptyTable
            header="No disk encryption status"
            info="Expecting to status data? Try again in a few seconds as the system
              catches up."
          />
        )}
        // these 2 properties allow linking on click anywhere in the row
        disableMultiRowSelect
        onSelectSingleRow={onSelectSingleRow}
      />
    </div>
  );
};

export default DiskEncryptionTable;
