import React from "react";

import { LOW_DISK_SPACE_GB } from "pages/DashboardPage/helpers";

import { PlatformValueOptions } from "utilities/constants";
import LowDiskSpaceHosts from "../../cards/LowDiskSpaceHosts";
import MissingHosts from "../../cards/MissingHosts";
import TotalHosts from "../../cards/TotalHosts";

const baseClass = "metrics-host-counts";

interface IPlatformHostCountsProps {
  currentTeamId: number | undefined;
  selectedPlatform?: PlatformValueOptions;
  totalHostCount?: number;
  isPremiumTier?: boolean;
  missingCount: number;
  lowDiskSpaceCount: number;
  selectedPlatformLabelId?: number;
}

const MetricsHostCounts = ({
  currentTeamId,
  selectedPlatform,
  totalHostCount,
  isPremiumTier,
  missingCount,
  lowDiskSpaceCount,
  selectedPlatformLabelId,
}: IPlatformHostCountsProps): JSX.Element => {
  const TotalHostsCard = (
    <TotalHosts
      totalCount={totalHostCount}
      selectedPlatformLabelId={selectedPlatformLabelId}
      currentTeamId={currentTeamId}
    />
  );

  const MissingHostsCard = (
    <MissingHosts
      missingCount={missingCount}
      selectedPlatformLabelId={selectedPlatformLabelId}
      currentTeamId={currentTeamId}
    />
  );

  const LowDiskSpaceHostsCard = (
    <LowDiskSpaceHosts
      lowDiskSpaceGb={LOW_DISK_SPACE_GB}
      lowDiskSpaceCount={lowDiskSpaceCount}
      selectedPlatformLabelId={selectedPlatformLabelId}
      currentTeamId={currentTeamId}
      notSupported={selectedPlatform === "chrome"}
    />
  );

  return (
    <div className={baseClass}>
      {selectedPlatform === "all" && TotalHostsCard}
      {isPremiumTier &&
        selectedPlatform !== "ios" &&
        selectedPlatform !== "ipados" &&
        selectedPlatform !== "android" && (
          <>
            {MissingHostsCard}
            {LowDiskSpaceHostsCard}
          </>
        )}
    </div>
  );
};

export default MetricsHostCounts;
