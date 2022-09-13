import React from "react";

import SummaryTile from "../HostsSummary/SummaryTile";
import MissingHostsIcon from "../../../../../assets/images/icon-missing-hosts-28x24@2x.png";
import paths from "router/paths";

const baseClass = "missing-hosts";

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
  // Renders opaque information as host information is loading
  let opacity = { opacity: 0 };
  if (showHostsUI) {
    opacity = isLoadingHosts ? { opacity: 0.4 } : { opacity: 1 };
  }

  return (
    <div className={baseClass} style={opacity}>
      <SummaryTile
        icon={MissingHostsIcon}
        count={missingCount}
        isLoading={isLoadingHosts}
        showUI={showHostsUI}
        title="Missing hosts"
        tooltip="Hosts that have not been online in 10 days or more."
        path={paths.MANAGE_HOSTS}
      />
    </div>
  );
};

export default MissingHosts;
