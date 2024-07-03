import React, { useCallback } from "react";
import PATHS from "router/paths";

import { PLATFORM_NAME_TO_LABEL_NAME } from "pages/DashboardPage/helpers";
import DataError from "components/DataError";
import { IHostSummary } from "interfaces/host_summary";
import { PlatformValueOptions } from "utilities/constants";

import SummaryTile from "./SummaryTile";

const baseClass = "hosts-summary";

interface IHostSummaryProps {
  currentTeamId: number | undefined;
  macCount: number;
  windowsCount: number;
  linuxCount: number;
  chromeCount: number;
  iosCount: number;
  ipadosCount: number;
  isLoadingHostsSummary: boolean;
  builtInLabels?: IHostSummary["builtin_labels"];
  showHostsUI: boolean;
  errorHosts: boolean;
  selectedPlatform?: PlatformValueOptions;
}

const HostsSummary = ({
  currentTeamId,
  macCount,
  windowsCount,
  linuxCount,
  chromeCount,
  iosCount,
  ipadosCount,
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

  const getBuiltinLabelId = useCallback(
    (platformName: keyof typeof PLATFORM_NAME_TO_LABEL_NAME) =>
      builtInLabels?.find(
        (builtin) => builtin.name === PLATFORM_NAME_TO_LABEL_NAME[platformName]
      )?.id,
    [builtInLabels]
  );

  const renderMacCount = (teamId?: number) => {
    const macLabelId = getBuiltinLabelId("darwin");

    if (isLoadingHostsSummary || macLabelId === undefined) {
      return <></>;
    }

    return (
      <SummaryTile
        iconName="darwin"
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
    const windowsLabelId = getBuiltinLabelId("windows");

    if (isLoadingHostsSummary || windowsLabelId === undefined) {
      return <></>;
    }
    return (
      <SummaryTile
        iconName="windows"
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
    const linuxLabelId = getBuiltinLabelId("linux");

    if (isLoadingHostsSummary || linuxLabelId === undefined) {
      return <></>;
    }
    return (
      <SummaryTile
        iconName="linux"
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
    const chromeLabelId = getBuiltinLabelId("chrome");

    if (isLoadingHostsSummary || chromeLabelId === undefined) {
      return <></>;
    }

    return (
      <SummaryTile
        iconName="chrome"
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

  const renderIosCount = (teamId?: number) => {
    const iosLabelId = getBuiltinLabelId("ios");

    if (isLoadingHostsSummary || iosLabelId === undefined) {
      return <></>;
    }

    return (
      <SummaryTile
        iconName="iOS"
        count={iosCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title={`iPhone${iosCount === 1 ? "" : "s"}`}
        path={PATHS.MANAGE_HOSTS_LABEL(iosLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  const renderIpadosCount = (teamId?: number) => {
    const ipadosLabelId = getBuiltinLabelId("ipados");

    if (isLoadingHostsSummary || ipadosLabelId === undefined) {
      return <></>;
    }

    return (
      <SummaryTile
        iconName="iPadOS"
        count={ipadosCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title={`iPad${ipadosCount === 1 ? "" : "s"}`}
        path={PATHS.MANAGE_HOSTS_LABEL(ipadosLabelId).concat(
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
      case "ios":
        return renderIosCount(teamId);
      case "ipados":
        return renderIpadosCount(teamId);
      default:
        return (
          <>
            {renderMacCount(teamId)}
            {renderWindowsCount(teamId)}
            {renderLinuxCount(teamId)}
            {renderChromeCount(teamId)}
            {renderIosCount(teamId)}
            {renderIpadosCount(teamId)}
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
