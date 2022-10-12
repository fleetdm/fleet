import React from "react";

import paths from "router/paths";
import SummaryTile from "../HostsSummary/SummaryTile";

const baseClass = "hosts-status";

interface IHostSummaryProps {
  onlineCount: number;
  offlineCount: number;
  isLoadingHosts: boolean;
  showHostsUI: boolean;
}

const HostsStatus = ({
  onlineCount,
  offlineCount,
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
        count={onlineCount}
        isLoading={isLoadingHosts}
        showUI={showHostsUI}
        title="Online hosts"
        path={`${paths.MANAGE_HOSTS}?status=online`}
      />
      <SummaryTile
        count={offlineCount}
        isLoading={isLoadingHosts}
        showUI={showHostsUI}
        title="Offline hosts"
        path={`${paths.MANAGE_HOSTS}?status=offline`}
      />
    </div>
  );
};

export default HostsStatus;
