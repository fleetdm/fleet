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

  const showMissingAndLowDiskHosts =
    selectedPlatform !== "ios" &&
    selectedPlatform !== "ipados" &&
    selectedPlatform !== "android";

  return (
    <div className={baseClass}>
      {selectedPlatform === "all" && TotalHostsCard}
      {showMissingAndLowDiskHosts && MissingHostsCard}
      {/* Low disk space is Premium-only: `low_disk_space_count` is null for
          non-Premium callers and the linked filter is Premium-gated. */}
      {isPremiumTier && showMissingAndLowDiskHosts && LowDiskSpaceHostsCard}
      {/* ABM issue count is only populated on Premium (see DashboardPage). */}
      {ABMIssueHostsCard}
    </div>
  );
};

export default MetricsHostCounts;
