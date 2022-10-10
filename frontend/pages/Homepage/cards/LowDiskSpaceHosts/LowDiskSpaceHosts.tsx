import React from "react";
import PATHS from "router/paths";

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
  return (
    <div className={baseClass}>
      <SummaryTile
        icon={LowDiskSpaceIcon}
        count={lowDiskSpaceCount}
        isLoading={isLoadingHosts}
        showUI={showHostsUI}
        title="Low disk space hosts"
        tooltip={`Hosts that have ${lowDiskSpaceGb} GB or less disk space available.`}
        path={`${PATHS.MANAGE_HOSTS}?low_disk_space=${lowDiskSpaceGb}`}
      />
    </div>
  );
};

export default LowDiskSpaceHosts;
