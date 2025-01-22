import React from "react";
import PATHS from "router/paths";

import { buildQueryStringFromParams } from "utilities/url";

import HostCountCard from "../HostCountCard";

const baseClass = "hosts-total";

interface ITotalHostsProps {
  totalCount?: number;
  isLoadingHosts: boolean;
  showHostsUI: boolean;
  selectedPlatformLabelId?: number;
  currentTeamId?: number;
}

const TOOLTIP_TEXT = "Total number of hosts.";

const TotalHosts = ({
  totalCount,
  isLoadingHosts,
  showHostsUI,
  selectedPlatformLabelId,
  currentTeamId,
}: ITotalHostsProps): JSX.Element => {
  // build the manage hosts URL filtered by low disk space only
  // currently backend cannot filter by both low disk space and label
  const queryParams = {
    team_id: currentTeamId,
  };
  const queryString = buildQueryStringFromParams(queryParams);
  const endpoint = selectedPlatformLabelId
    ? PATHS.MANAGE_HOSTS_LABEL(selectedPlatformLabelId)
    : PATHS.MANAGE_HOSTS;
  const path = `${endpoint}?${queryString}`;

  return (
    <HostCountCard
      iconName="total-hosts"
      count={totalCount || 0}
      isLoading={isLoadingHosts}
      showUI={showHostsUI}
      title="Total hosts"
      tooltip={TOOLTIP_TEXT}
      path={path}
      className={baseClass}
      iconPosition="left"
    />
  );
};

export default TotalHosts;
