import React from "react";

const baseClass = "low-disk-space-hosts";

interface IHostSummaryProps {
  lowDiskSpaceCount: number;
  isLoadingHosts: boolean;
  showHostsUI: boolean;
}

const LowDiskSpaceHosts = ({
  lowDiskSpaceCount,
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
      <div className={`${baseClass}__tile low-disk-space-tile`}>
        <div>
          <div
            className={`${baseClass}__tile-count ${baseClass}__tile-count--low-disk-space`}
          >
            {lowDiskSpaceCount}
          </div>
          <div className={`${baseClass}__tile-description`}>Missing hosts</div>
        </div>
      </div>
    </div>
  );
};

export default LowDiskSpaceHosts;
