import React from "react";
import PATHS from "router/paths";

import { buildQueryStringFromParams } from "utilities/url";

import SummaryTile from "../HostsSummary/SummaryTile";

const baseClass = "hosts-missing";

interface IHostSummaryProps {
  missingCount: number;
  isLoadingHosts: boolean;
  showHostsUI: boolean;
  selectedPlatformLabelId?: number;
  currentTeamId?: number;
}

const MissingHosts = ({
  missingCount,
  isLoadingHosts,
  showHostsUI,
  selectedPlatformLabelId,
  currentTeamId,
}: IHostSummaryProps): JSX.Element => {
  // build the manage hosts URL filtered by missing and platform
  const queryParams = {
    status: "missing",
    team_id: currentTeamId,
  };
  const queryString = buildQueryStringFromParams(queryParams);
  const endpoint = selectedPlatformLabelId
    ? PATHS.MANAGE_HOSTS_LABEL(selectedPlatformLabelId)
    : PATHS.MANAGE_HOSTS;
  const path = `${endpoint}?${queryString}`;

  return (
    <div className={baseClass}>
      <SummaryTile
        iconName="missing-hosts"
        count={missingCount}
        isLoading={isLoadingHosts}
        showUI={showHostsUI}
        title="Missing hosts"
        tooltip="Hosts that have not been online in 30 days or more."
        path={path}
      />
    </div>
  );
};

export default MissingHosts;
