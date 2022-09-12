import React from "react";

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
      <div className={`${baseClass}__tile missing-tile`}>
        <div>
          <div
            className={`${baseClass}__tile-count ${baseClass}__tile-count--missing`}
          >
            {missingCount}
          </div>
          <div className={`${baseClass}__tile-description`}>Missing hosts</div>
        </div>
      </div>
    </div>
  );
};

export default MissingHosts;
