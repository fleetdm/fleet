import React from "react";
import PATHS from "router/paths";

import { buildQueryStringFromParams } from "utilities/url";

import HostCountCard from "../HostCountCard";

const baseClass = "hosts-low-space";

interface ILowDiskSpaceHostsProps {
  lowDiskSpaceGb: number;
  lowDiskSpaceCount: number;
  selectedPlatformLabelId?: number;
  currentTeamId?: number;
  notSupported: boolean;
}

const LowDiskSpaceHosts = ({
  lowDiskSpaceGb,
  lowDiskSpaceCount,
  selectedPlatformLabelId,
  currentTeamId,
  notSupported = false, // default to supporting this feature
}: ILowDiskSpaceHostsProps): JSX.Element => {
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
    <HostCountCard
      iconName="low-disk-space-hosts"
      count={lowDiskSpaceCount}
      title="Low disk space hosts"
      tooltip={tooltipText}
      path={path}
      notSupported={notSupported}
      className={baseClass}
      iconPosition="left"
    />
  );
};

export default LowDiskSpaceHosts;
