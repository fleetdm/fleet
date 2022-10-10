import React from "react";
import PATHS from "router/paths";

import SummaryTile from "../HostsSummary/SummaryTile";
import MissingHostsIcon from "../../../../../assets/images/icon-missing-hosts-28x24@2x.png";

const baseClass = "hosts-missing";

interface IHostSummaryProps {
  missingCount: number;
  isLoadingHosts: boolean;
  showHostsUI: boolean;
}

const MissingHosts = ({
  missingCount,
  isLoadingHosts,
  showHostsUI,
}: IHostSummaryProps): JSX.Element => {
  return (
    <div className={baseClass}>
      <SummaryTile
        icon={MissingHostsIcon}
        count={missingCount}
        isLoading={isLoadingHosts}
        showUI={showHostsUI}
        title="Missing hosts"
        tooltip="Hosts that have not been online in 30 days or more."
        path={`${PATHS.MANAGE_HOSTS}?status=missing`}
      />
    </div>
  );
};

export default MissingHosts;
