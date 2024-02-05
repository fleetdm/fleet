import React from "react";
import PATHS from "router/paths";

import { buildQueryStringFromParams } from "utilities/url";

import SummaryTile from "../HostsSummary/SummaryTile";

const baseClass = "hosts-low-space";

interface IHostSummaryProps {
  lowDiskSpaceGb: number;
  lowDiskSpaceCount: number;
  isLoadingHosts: boolean;
  showHostsUI: boolean;
  selectedPlatformLabelId?: number;
  currentTeamId?: number;
  isSandboxMode?: boolean;
  notSupported: boolean;
}

const LowDiskSpaceHosts = ({
  lowDiskSpaceGb,
  lowDiskSpaceCount,
  isLoadingHosts,
  showHostsUI,
  selectedPlatformLabelId,
  currentTeamId,
  isSandboxMode = false,
  notSupported = false, // default to supporting this feature
}: IHostSummaryProps): JSX.Element => {
  // build the manage hosts URL filtered by low disk space only
  // currently backend cannot filter by both low disk space and label
  const queryParams = {
    low_disk_space: lowDiskSpaceGb,
    team_id: currentTeamId,
  };
  const queryString = buildQueryStringFromParams(queryParams);
  const endpoint = selectedPlatformLabelId
    ? PATHS.MANAGE_HOSTS_LABEL(selectedPlatformLabelId)
    : PATHS.MANAGE_HOSTS;
  const path = `${endpoint}?${queryString}`;

  const tooltipText = notSupported
    ? "Disk space info is not available for Chromebooks."
    : `Hosts that have ${lowDiskSpaceGb} GB or less disk space available.`;

  return (
    <div className={baseClass}>
      <SummaryTile
        iconName="low-disk-space-hosts"
        count={lowDiskSpaceCount}
        isLoading={isLoadingHosts}
        showUI={showHostsUI}
        title="Low disk space hosts"
        tooltip={tooltipText}
        path={path}
        isSandboxMode={isSandboxMode}
        sandboxPremiumOnlyIcon
        notSupported={notSupported}
      />
    </div>
  );
};

export default LowDiskSpaceHosts;
