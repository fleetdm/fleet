import EmptyTable from "components/EmptyTable";
import TableContainer from "components/TableContainer";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import React, { useMemo } from "react";
import {
  PlatformValueOptions,
  PLATFORM_DISPLAY_NAMES,
} from "utilities/constants";

import generateTableHeaders from "./OSTableConfig";

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 8;

const baseClass = "operating-systems";

const EmptyOS = (platform: PlatformValueOptions): JSX.Element => (
  <EmptyTable
    className={`${baseClass}__os-empty-table`}
    header={`No${
      ` ${PLATFORM_DISPLAY_NAMES[platform]}` || ""
    } operating systems detected`}
    info="This report is updated every hour to protect the performance of your
  devices."
  />
);

interface IOSTableProps {
  currentTeamId?: number;
  osVersions: IOperatingSystemVersion[];
  selectedPlatform: PlatformValueOptions;
  isLoading: boolean;
}

const OSTable = ({
  currentTeamId,
  osVersions,
  selectedPlatform,
  isLoading,
}: IOSTableProps) => {
  const columnConfigs = useMemo(
    () => generateTableHeaders(currentTeamId, undefined),
    [currentTeamId]
  );

  const showPaginationControls = osVersions.length > PAGE_SIZE;

  return (
    <TableContainer
      columnConfigs={columnConfigs}
      data={osVersions}
      isLoading={isLoading}
      defaultSortHeader={DEFAULT_SORT_HEADER}
      defaultSortDirection={DEFAULT_SORT_DIRECTION}
      resultsTitle="Operating systems"
      emptyComponent={() => EmptyOS(selectedPlatform)}
      showMarkAllPages={false}
      isAllPagesSelected={false}
      disableCount
      isClientSidePagination={showPaginationControls}
      disablePagination={!showPaginationControls}
      pageSize={PAGE_SIZE}
    />
  );
};

export default OSTable;
