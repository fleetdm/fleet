import EmptyState from "components/EmptyState";
import TableContainer from "components/TableContainer";
import { IOperatingSystemVersion } from "interfaces/operating_system";
import React, { useMemo } from "react";
import {
  PlatformValueOptions,
  PLATFORM_DISPLAY_NAMES,
} from "utilities/constants";

import generateTableHeaders from "./OSTableConfig";

const DEFAULT_SORT_DIRECTION = "desc";
// Defaults to sorting by host count when viewing all platforms mixed
// together (where comparing versions across platforms isn't meaningful),
// and by version once a single platform is selected.
const DEFAULT_SORT_HEADER_ALL_PLATFORMS = "hosts_count";
const DEFAULT_SORT_HEADER_SINGLE_PLATFORM = "version";
const PAGE_SIZE = 8;

const baseClass = "operating-systems";

const EmptyOS = (platform: PlatformValueOptions): JSX.Element => (
  <EmptyState
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
  const platformHostTotals = useMemo(() => {
    const totals: Record<string, number> = {};
    osVersions.forEach(({ platform, hosts_count }) => {
      totals[platform] = (totals[platform] ?? 0) + hosts_count;
    });
    return totals;
  }, [osVersions]);

  const columnConfigs = useMemo(
    // Linux is the only platform where the distro name ("Ubuntu", "Debian",
    // ...) isn't obvious from the Version column alone, so it gets the extra
    // Name column that other platforms don't need.
    () =>
      generateTableHeaders(currentTeamId, undefined, {
        includeName: selectedPlatform === "linux",
        platformHostTotals,
      }),
    [currentTeamId, selectedPlatform, platformHostTotals]
  );

  const showPaginationControls = osVersions.length > PAGE_SIZE;

  const defaultSortHeader =
    selectedPlatform === "all"
      ? DEFAULT_SORT_HEADER_ALL_PLATFORMS
      : DEFAULT_SORT_HEADER_SINGLE_PLATFORM;

  return (
    <TableContainer
      columnConfigs={columnConfigs}
      data={osVersions}
      isLoading={isLoading}
      defaultSortHeader={defaultSortHeader}
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
