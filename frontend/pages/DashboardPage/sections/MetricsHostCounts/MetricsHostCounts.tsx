import React from "react";

import { LOW_DISK_SPACE_GB } from "pages/DashboardPage/helpers";

import { PlatformValueOptions } from "utilities/constants";
import DataError from "components/DataError";
import LowDiskSpaceHosts from "../../cards/LowDiskSpaceHosts";
import MissingHosts from "../../cards/MissingHosts";
import TotalHosts from "../../cards/TotalHosts";

const baseClass = "metrics-host-counts";

interface IPlatformHostCountsProps {
  currentTeamId: number | undefined;
  isLoadingHostsSummary: boolean;
  showHostsUI: boolean;
  errorHosts: boolean;
  selectedPlatform?: PlatformValueOptions;
  totalHostCount?: number;
  isPremiumTier?: boolean;
  missingCount: number;
  lowDiskSpaceCount: number;
  selectedPlatformLabelId?: number;
}

const MetricsHostCounts = ({
  currentTeamId,
  isLoadingHostsSummary,
  showHostsUI,
  errorHosts,
  selectedPlatform,
  totalHostCount,
  isPremiumTier,
  missingCount,
  lowDiskSpaceCount,
  selectedPlatformLabelId,
}: IPlatformHostCountsProps): JSX.Element => {
  // Renders semi-transparent screen as host information is loading
  let opacity = { opacity: 0 };
  if (showHostsUI) {
    opacity = isLoadingHostsSummary ? { opacity: 0.4 } : { opacity: 1 };
  }

  if (errorHosts && !isLoadingHostsSummary) {
    return <DataError card />;
  }

  const TotalHostsCard = (
    <TotalHosts
      totalCount={totalHostCount}
      isLoadingHosts={isLoadingHostsSummary}
      showHostsUI={showHostsUI}
      selectedPlatformLabelId={selectedPlatformLabelId}
      currentTeamId={currentTeamId}
    />
  );

  const MissingHostsCard = (
    <MissingHosts
      missingCount={missingCount}
      isLoadingHosts={isLoadingHostsSummary}
      showHostsUI={showHostsUI}
      selectedPlatformLabelId={selectedPlatformLabelId}
      currentTeamId={currentTeamId}
    />
  );

  const LowDiskSpaceHostsCard = (
    <LowDiskSpaceHosts
      lowDiskSpaceGb={LOW_DISK_SPACE_GB}
      lowDiskSpaceCount={lowDiskSpaceCount}
      isLoadingHosts={isLoadingHostsSummary}
      showHostsUI={showHostsUI}
      selectedPlatformLabelId={selectedPlatformLabelId}
      currentTeamId={currentTeamId}
      notSupported={selectedPlatform === "chrome"}
    />
  );

  return (
    <div className={baseClass} style={opacity}>
      {selectedPlatform === "all" && TotalHostsCard}
      {isPremiumTier &&
        selectedPlatform !== "ios" &&
        selectedPlatform !== "ipados" && (
          <>
            {MissingHostsCard}
            {LowDiskSpaceHostsCard}
          </>
        )}
    </div>
  );
};

export default MetricsHostCounts;
