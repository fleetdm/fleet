import React, { useEffect } from "react";
import PATHS from "router/paths";

import { ILabelSummary } from "interfaces/label";
import { ISelectedPlatform } from "interfaces/platform";

import { buildQueryStringFromParams } from "utilities/url";

import DataError from "components/DataError";
import SummaryTile from "./SummaryTile";

import WindowsIcon from "../../../../../assets/images/icon-windows-48x48@2x.png";
import LinuxIcon from "../../../../../assets/images/icon-linux-48x48@2x.png";
import MacIcon from "../../../../../assets/images/icon-mac-48x48@2x.png";

const baseClass = "hosts-summary";

interface IHostSummaryProps {
  currentTeamId: number | undefined;
  macCount: number;
  windowsCount: number;
  linuxCount: number;
  isLoadingHostsSummary: boolean;
  showHostsUI: boolean;
  errorHosts: boolean;
  selectedPlatform?: ISelectedPlatform;
  selectedPlatformLabelId?: number;
  labels?: ILabelSummary[];
  setActionURL?: (url: string) => void;
}

const HostsSummary = ({
  currentTeamId,
  macCount,
  windowsCount,
  linuxCount,
  isLoadingHostsSummary,
  showHostsUI,
  errorHosts,
  selectedPlatform,
  selectedPlatformLabelId,
  labels,
  setActionURL,
}: IHostSummaryProps): JSX.Element => {
  // build the manage hosts URL
  useEffect(() => {
    if (labels) {
      const queryParams = {
        team_id: currentTeamId,
      };

      const queryString = buildQueryStringFromParams(queryParams);
      const endpoint = selectedPlatformLabelId
        ? PATHS.MANAGE_HOSTS_LABEL(selectedPlatformLabelId)
        : PATHS.MANAGE_HOSTS;
      const path = `${endpoint}?${queryString}`;

      setActionURL && setActionURL(path);
    }
  }, [labels, selectedPlatformLabelId, currentTeamId]);

  // Renders semi-transparent screen as host information is loading
  let opacity = { opacity: 0 };
  if (showHostsUI) {
    opacity = isLoadingHostsSummary ? { opacity: 0.4 } : { opacity: 1 };
  }

  const renderMacCount = () => (
    <SummaryTile
      iconName="darwin-purple"
      count={macCount}
      isLoading={isLoadingHostsSummary}
      showUI={showHostsUI}
      title="macOS hosts"
      path={PATHS.MANAGE_HOSTS_LABEL(7)}
    />
  );

  const renderWindowsCount = () => (
    <SummaryTile
      iconName="windows-blue"
      count={windowsCount}
      isLoading={isLoadingHostsSummary}
      showUI={showHostsUI}
      title="Windows hosts"
      path={PATHS.MANAGE_HOSTS_LABEL(10)}
    />
  );

  const renderLinuxCount = () => (
    <SummaryTile
      iconName="linux-green"
      count={linuxCount}
      isLoading={isLoadingHostsSummary}
      showUI={showHostsUI}
      title="Linux hosts"
      path={PATHS.MANAGE_HOSTS_LABEL(12)}
    />
  );

  const renderCounts = () => {
    switch (selectedPlatform) {
      case "darwin":
        return renderMacCount();
      case "windows":
        return renderWindowsCount();
      case "linux":
        return renderLinuxCount();
      default:
        return (
          <>
            {renderMacCount()}
            {renderWindowsCount()}
            {renderLinuxCount()}
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
      {renderCounts()}
    </div>
  );
};

export default HostsSummary;
