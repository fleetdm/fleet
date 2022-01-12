import React from "react";

const baseClass = "hosts-status";

interface IHostSummaryProps {
  onlineCount: string | undefined;
  offlineCount: string | undefined;
  newCount: string | undefined;
  isLoadingHosts: boolean;
  showHostsData: boolean;
}

const HostsStatus = ({
  onlineCount,
  offlineCount,
  newCount,
  isLoadingHosts,
  showHostsData,
}: IHostSummaryProps): JSX.Element => {
  // Renders opaque information as host information is loading
  let opacity = { opacity: 0 };
  if (showHostsData) {
    opacity = isLoadingHosts ? { opacity: 0.4 } : { opacity: 1 };
  }

  return (
    <div className={baseClass} style={opacity}>
      <div className={`${baseClass}__tile online-tile`}>
        <div>
          <div
            className={`${baseClass}__tile-count ${baseClass}__tile-count--online`}
          >
            {onlineCount}
          </div>
          <div className={`${baseClass}__tile-description`}>Online hosts</div>
        </div>
      </div>
      <div className={`${baseClass}__tile offline-tile`}>
        <div>
          <div
            className={`${baseClass}__tile-count ${baseClass}__tile-count--offline`}
          >
            {offlineCount}
          </div>
          <div className={`${baseClass}__tile-description`}>Offline hosts</div>
        </div>
      </div>
      <div className={`${baseClass}__tile new-tile`}>
        <div>
          <div
            className={`${baseClass}__tile-count ${baseClass}__tile-count--new`}
          >
            {newCount}
          </div>
          <div className={`${baseClass}__tile-description`}>New hosts</div>
        </div>
      </div>
    </div>
  );
};

export default HostsStatus;
