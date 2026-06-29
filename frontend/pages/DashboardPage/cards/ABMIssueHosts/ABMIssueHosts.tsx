import React from "react";
import PATHS from "router/paths";

import { getPathWithQueryParams } from "utilities/url";

import HostCountCard from "../HostCountCard";

const baseClass = "hosts-abm-issue";

interface IABMIssueHostsProps {
  abmIssueCount: number;
  selectedPlatformLabelId?: number;
  currentTeamId?: number;
}

export const abmIssueTooltip = (): JSX.Element => {
  return (
    <span>
      Hosts that have Apple Business (AB) profile assignment issue. Migration or
      new Mac setup won&apos;t work.
    </span>
  );
};

const ABMIssueHosts = ({
  abmIssueCount,
  selectedPlatformLabelId,
  currentTeamId,
}: IABMIssueHostsProps): JSX.Element | null => {
  // build the manage hosts URL filtered by missing and platform
  const queryParams = {
    dep_profile_error: true,
    fleet_id: currentTeamId,
  };

  const endpoint = selectedPlatformLabelId
    ? PATHS.MANAGE_HOSTS_LABEL(selectedPlatformLabelId)
    : PATHS.MANAGE_HOSTS;
  const path = getPathWithQueryParams(endpoint, queryParams);

  return (
    <HostCountCard
      iconName="abm-issue-hosts"
      count={abmIssueCount}
      title="AB issue"
      tooltip={abmIssueTooltip()}
      path={path}
      className={baseClass}
      iconPosition="left"
    />
  );
};

export default ABMIssueHosts;
