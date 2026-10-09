import React, { useCallback } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { Row } from "react-table";

import DataError from "components/DataError";
import EmptyState from "components/EmptyState";
import TableContainer from "components/TableContainer";
import { getBuiltinPlatformLabelId } from "interfaces/label";
import PATHS from "router/paths";
import diskEncryptionAPI, {
  IDiskEncryptionStatusAggregate,
  IDiskEncryptionSummaryResponse,
} from "services/entities/disk_encryption";
import { HOSTS_QUERY_PARAMS } from "services/entities/hosts";
import labelsAPI, { ILabelsSummaryResponse } from "services/entities/labels";
import { getPathWithQueryParams } from "utilities/url";

import {
  generateTableHeaders,
  generateTableData,
  IStatusCellValue,
} from "./DiskEncryptionTableConfig";

const baseClass = "disk-encryption-table";

// tab platforms mapped to the osquery platform of their built-in label
const PLATFORM_TO_OSQUERY_PLATFORM = {
  macos: "darwin",
  windows: "windows",
  linux: "linux",
} as const;

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

  // builtin labels are global, so the summary is fetched without a fleet
  // (Free tier rejects fleet-scoped summaries)
  const { data: labels } = useQuery<
    ILabelsSummaryResponse,
    Error,
    ILabelsSummaryResponse["labels"]
  >(["labelsSummary"], () => labelsAPI.summary(), {
    select: (res) => res.labels,
    refetchOnWindowFocus: false,
    retry: false,
  });

  const onSelectSingleRow = useCallback(
    (row: IDiskEncryptionRowProps) => {
      const { status, teamId } = row.original;

      const queryParams = {
        [HOSTS_QUERY_PARAMS.DISK_ENCRYPTION]: status?.value,
        fleet_id: teamId,
      };
      // fall back to the unfiltered hosts page when the platform label hasn't
      // loaded or is missing, rather than dropping the click
      const labelId = getBuiltinPlatformLabelId(
        labels,
        PLATFORM_TO_OSQUERY_PLATFORM[platform]
      );
      const endpoint =
        labelId !== undefined
          ? PATHS.MANAGE_HOSTS_LABEL(labelId)
          : PATHS.MANAGE_HOSTS;
      const path = getPathWithQueryParams(endpoint, queryParams);

      router.push(path);
    },
    [router, labels, platform]
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
