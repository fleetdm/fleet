import React from "react";
import PATHS from "router/paths";

import { PLATFORM_NAME_TO_LABEL_NAME } from "utilities/constants";
import DataError from "components/DataError";
import { SelectedPlatform } from "interfaces/platform";
import { IHostSummary } from "interfaces/host_summary";

import SummaryTile from "./SummaryTile";

const baseClass = "hosts-summary";

interface IHostSummaryProps {
  currentTeamId: number | undefined;
  macCount: number;
  windowsCount: number;
  linuxCount: number;
  chromeCount: number;
  isLoadingHostsSummary: boolean;
  builtInLabels?: IHostSummary["builtin_labels"];
  showHostsUI: boolean;
  errorHosts: boolean;
  selectedPlatform?: SelectedPlatform;
}

const HostsSummary = ({
  currentTeamId,
  macCount,
  windowsCount,
  linuxCount,
  chromeCount,
  isLoadingHostsSummary,
  builtInLabels,
  showHostsUI,
  errorHosts,
  selectedPlatform,
}: IHostSummaryProps): JSX.Element => {
  // Renders semi-transparent screen as host information is loading
  let opacity = { opacity: 0 };
  if (showHostsUI) {
    opacity = isLoadingHostsSummary ? { opacity: 0.4 } : { opacity: 1 };
  }

  const renderMacCount = (teamId?: number) => {
    const macLabelId = builtInLabels?.find((builtin) => {
      return builtin.name === PLATFORM_NAME_TO_LABEL_NAME.darwin;
    })?.id;

    if (isLoadingHostsSummary || macLabelId === undefined) {
      return <></>;
    }

    return (
      <SummaryTile
        iconName="darwin"
        circledIcon
        count={macCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title={`macOS host${macCount === 1 ? "" : "s"}`}
        path={PATHS.MANAGE_HOSTS_LABEL(macLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  const renderWindowsCount = (teamId?: number) => {
    const windowsLabelId = builtInLabels?.find(
      (builtin) => builtin.name === PLATFORM_NAME_TO_LABEL_NAME.windows
    )?.id;

    if (isLoadingHostsSummary || windowsLabelId === undefined) {
      return <></>;
    }
    return (
      <SummaryTile
        iconName="windows"
        circledIcon
        count={windowsCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title={`Windows host${windowsCount === 1 ? "" : "s"}`}
        path={PATHS.MANAGE_HOSTS_LABEL(windowsLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  const renderLinuxCount = (teamId?: number) => {
    const linuxLabelId = builtInLabels?.find(
      (builtin) => builtin.name === PLATFORM_NAME_TO_LABEL_NAME.linux
    )?.id;

    if (isLoadingHostsSummary || linuxLabelId === undefined) {
      return <></>;
    }
    return (
      <SummaryTile
        iconName="linux"
        circledIcon
        count={linuxCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title={`Linux host${linuxCount === 1 ? "" : "s"}`}
        path={PATHS.MANAGE_HOSTS_LABEL(linuxLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  const renderChromeCount = (teamId?: number) => {
    const chromeLabelId = builtInLabels?.find(
      (builtin) => builtin.name === PLATFORM_NAME_TO_LABEL_NAME.chrome
    )?.id;

    if (isLoadingHostsSummary || chromeLabelId === undefined) {
      return <></>;
    }

    return (
      <SummaryTile
        iconName="chrome"
        circledIcon
        count={chromeCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title={`Chromebook${chromeCount === 1 ? "" : "s"}`}
        path={PATHS.MANAGE_HOSTS_LABEL(chromeLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

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
