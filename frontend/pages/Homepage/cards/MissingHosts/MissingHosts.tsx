import React from "react";
import PATHS from "router/paths";

import { ISelectedPlatform } from "interfaces/platform";
import { buildQueryStringFromParams } from "utilities/url";

import SummaryTile from "../HostsSummary/SummaryTile";
import MissingHostsIcon from "../../../../../assets/images/icon-missing-hosts-28x24@2x.png";

const baseClass = "hosts-missing";

interface IHostSummaryProps {
  missingCount: number;
  isLoadingHosts: boolean;
  showHostsUI: boolean;
  selectedPlatformLabelId?: number;
}

const MissingHosts = ({
  missingCount,
  isLoadingHosts,
  showHostsUI,
  selectedPlatformLabelId,
}: IHostSummaryProps): JSX.Element => {
  // build the manage hosts URL
  const queryParams = {
    status: "missing",
  };
  const queryString = buildQueryStringFromParams(queryParams);
  const endpoint = selectedPlatformLabelId
    ? PATHS.MANAGE_HOSTS_LABEL(selectedPlatformLabelId)
    : PATHS.MANAGE_HOSTS;
  const path = `${endpoint}?${queryString}`;

  return (
    <div className={baseClass}>
      <SummaryTile
        icon={MissingHostsIcon}
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
