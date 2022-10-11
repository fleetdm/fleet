import React from "react";
import PATHS from "router/paths";

import { buildQueryStringFromParams } from "utilities/url";

import SummaryTile from "../HostsSummary/SummaryTile";
import LowDiskSpaceIcon from "../../../../../assets/images/icon-low-disk-space-32x19@2x.png";

const baseClass = "hosts-low-space";

interface IHostSummaryProps {
  lowDiskSpaceGb: number;
  lowDiskSpaceCount: number;
  isLoadingHosts: boolean;
  showHostsUI: boolean;
}

const LowDiskSpaceHosts = ({
  lowDiskSpaceGb,
  lowDiskSpaceCount,
  isLoadingHosts,
  showHostsUI,
}: IHostSummaryProps): JSX.Element => {
  // build the manage hosts URL filtered by low disk space only
  // currently backend cannot filter by both low disk space and label
  const queryParams = {
    low_disk_space: lowDiskSpaceGb,
  };
  const queryString = buildQueryStringFromParams(queryParams);
  const endpoint = PATHS.MANAGE_HOSTS;
  const path = `${endpoint}?${queryString}`;

  return (
    <div className={baseClass}>
      <SummaryTile
        icon={LowDiskSpaceIcon}
        count={lowDiskSpaceCount}
        isLoading={isLoadingHosts}
        showUI={showHostsUI}
        title="Low disk space hosts"
        tooltip={`Hosts that have ${lowDiskSpaceGb} GB or less disk space available.`}
        path={path}
      />
    </div>
  );
};

export default LowDiskSpaceHosts;
