import React from "react";
import PATHS from "router/paths";

import { buildQueryStringFromParams } from "utilities/url";

import HostCountCard from "../HostCountCard";

const baseClass = "hosts-missing";

interface IMissingHostsProps {
  missingCount: number;
  selectedPlatformLabelId?: number;
  currentTeamId?: number;
}

const MissingHosts = ({
  missingCount,
  selectedPlatformLabelId,
  currentTeamId,
}: IMissingHostsProps): JSX.Element => {
  // build the manage hosts URL filtered by missing and platform
  const queryParams = {
    status: "missing",
    team_id: currentTeamId,
  };
  const queryString = buildQueryStringFromParams(queryParams);
  const endpoint = selectedPlatformLabelId
    ? PATHS.MANAGE_HOSTS_LABEL(selectedPlatformLabelId)
    : PATHS.MANAGE_HOSTS;
  const path = `${endpoint}?${queryString}`;

  return (
    <HostCountCard
      iconName="missing-hosts"
      count={missingCount}
      title="Missing hosts"
      tooltip="Hosts that have not been online in 30 days or more."
      path={path}
      className={baseClass}
      iconPosition="left"
    />
  );
};

export default MissingHosts;
