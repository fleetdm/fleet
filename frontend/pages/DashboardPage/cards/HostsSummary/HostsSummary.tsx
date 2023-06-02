import React from "react";
import PATHS from "router/paths";

import DataError from "components/DataError";
import { ISelectedPlatform } from "interfaces/platform";

import SummaryTile from "./SummaryTile";

const baseClass = "hosts-summary";

interface IHostSummaryProps {
  currentTeamId: number | undefined;
  macCount: number;
  windowsCount: number;
  linuxCount: number;
  chromeCount: number;
  isLoadingHostsSummary: boolean;
  showHostsUI: boolean;
  errorHosts: boolean;
  selectedPlatform?: ISelectedPlatform;
}

const HostsSummary = ({
  currentTeamId,
  macCount,
  windowsCount,
  linuxCount,
  chromeCount,
  isLoadingHostsSummary,
  showHostsUI,
  errorHosts,
  selectedPlatform,
}: IHostSummaryProps): JSX.Element => {
  // Renders semi-transparent screen as host information is loading
  let opacity = { opacity: 0 };
  if (showHostsUI) {
    opacity = isLoadingHostsSummary ? { opacity: 0.4 } : { opacity: 1 };
  }

  const renderMacCount = (teamId?: number) => (
    <SummaryTile
      iconName="darwin-purple"
      count={macCount}
      isLoading={isLoadingHostsSummary}
      showUI={showHostsUI}
      title="macOS hosts"
      path={PATHS.MANAGE_HOSTS_LABEL(7).concat(
        teamId !== undefined ? `?team_id=${teamId}` : ""
      )}
    />
  );

  const renderWindowsCount = (teamId?: number) => (
    <SummaryTile
      iconName="windows-blue"
      count={windowsCount}
      isLoading={isLoadingHostsSummary}
      showUI={showHostsUI}
      title="Windows hosts"
      path={PATHS.MANAGE_HOSTS_LABEL(10).concat(
        teamId !== undefined ? `?team_id=${teamId}` : ""
      )}
    />
  );

  const renderLinuxCount = (teamId?: number) => (
    <SummaryTile
      iconName="linux-green"
      count={linuxCount}
      isLoading={isLoadingHostsSummary}
      showUI={showHostsUI}
      title="Linux hosts"
      path={PATHS.MANAGE_HOSTS_LABEL(12).concat(
        teamId !== undefined ? `?team_id=${teamId}` : ""
      )}
    />
  );

  const renderChromeCount = (teamId?: number) => (
    <SummaryTile
      iconName="chrome"
      count={chromeCount}
      isLoading={isLoadingHostsSummary}
      showUI={showHostsUI}
      title="Chromebooks"
      // currently this is "test label"
      // TODO - ask backend what the label for chrome hosts will be
      path={PATHS.MANAGE_HOSTS_LABEL(14).concat(
        teamId !== undefined ? `?team_id=${teamId}` : ""
      )}
    />
  );
  const renderCounts = (teamId?: number) => {
    switch (selectedPlatform) {
      case "darwin":
        return renderMacCount(teamId);
      case "windows":
        return renderWindowsCount(teamId);
      case "linux":
        return renderLinuxCount(teamId);
      case "chrome":
        return renderChromeCount(teamId);
      default:
        return (
          <>
            {renderMacCount(teamId)}
            {renderWindowsCount(teamId)}
            {renderLinuxCount(teamId)}
            {renderChromeCount(teamId)}
          </>
        );
    }
  };

  if (errorHosts && !isLoadingHostsSummary) {
    return <DataError card />;
  }

  return (
    <div
      className={`${baseClass} ${
        selectedPlatform !== "all" ? "single-platform" : ""
      }`}
      style={opacity}
    >
      {renderCounts(currentTeamId)}
    </div>
  );
};

export default HostsSummary;
