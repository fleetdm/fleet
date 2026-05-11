import React from "react";

import { LOW_DISK_SPACE_GB } from "pages/DashboardPage/helpers";

import { PlatformValueOptions } from "utilities/constants";
import LowDiskSpaceHosts from "../../cards/LowDiskSpaceHosts";
import MissingHosts from "../../cards/MissingHosts";
import TotalHosts from "../../cards/TotalHosts";
import ABMIssueHosts from "../../cards/ABMIssueHosts";

const baseClass = "metrics-host-counts";

interface IPlatformHostCountsProps {
  currentTeamId: number | undefined;
  selectedPlatform?: PlatformValueOptions;
  totalHostCount?: number;
  isPremiumTier?: boolean;
  missingCount: number;
  lowDiskSpaceCount: number;
  abmIssueCount: number;
  selectedPlatformLabelId?: number;
}

const MetricsHostCounts = ({
  currentTeamId,
  selectedPlatform,
  totalHostCount,
  isPremiumTier,
  missingCount,
  lowDiskSpaceCount,
  abmIssueCount,
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

  // Does not render if abmIssueCount is 0 or undefined (e.g. on non-Apple platforms views)
  // Currently all undefined is defaulted to 0 upstream
  const ABMIssueHostsCard = abmIssueCount ? (
    <ABMIssueHosts
      abmIssueCount={abmIssueCount}
      selectedPlatformLabelId={selectedPlatformLabelId}
      currentTeamId={currentTeamId}
    />
  ) : null;

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
      {ABMIssueHostsCard}
    </div>
  );
};

export default MetricsHostCounts;
