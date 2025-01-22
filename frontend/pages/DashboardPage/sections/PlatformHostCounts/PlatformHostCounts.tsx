import React, { useCallback } from "react";
import PATHS from "router/paths";

import { PLATFORM_NAME_TO_LABEL_NAME } from "pages/DashboardPage/helpers";

import { IHostSummary } from "interfaces/host_summary";
import { PlatformValueOptions } from "utilities/constants";
import DataError from "components/DataError";

import HostCountCard from "../../cards/HostCountCard";

const baseClass = "platform-host-counts";

interface IPlatformHostCountsProps {
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
  totalHostCount?: number;
}

const PlatformHostCounts = ({
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
  totalHostCount,
}: IPlatformHostCountsProps): JSX.Element => {
  // Renders semi-transparent screen as host information is loading
  let opacity = { opacity: 0 };
  if (showHostsUI) {
    opacity = isLoadingHostsSummary ? { opacity: 0.4 } : { opacity: 1 };
  }

  // Only hide card if count is 0 but there are other platform counts
  const hidePlatformCard = (platformCount: number) => {
    return platformCount === 0 && totalHostCount && totalHostCount > 0;
  };

  const getBuiltinLabelId = useCallback(
    (platformName: keyof typeof PLATFORM_NAME_TO_LABEL_NAME) =>
      builtInLabels?.find(
        (builtin) => builtin.name === PLATFORM_NAME_TO_LABEL_NAME[platformName]
      )?.id,
    [builtInLabels]
  );

  const renderMacCard = (teamId?: number) => {
    const macLabelId = getBuiltinLabelId("darwin");

    if (hidePlatformCard(macCount)) {
      return null;
    }

    if (isLoadingHostsSummary || macLabelId === undefined) {
      return <></>;
    }

    return (
      <HostCountCard
        iconName="darwin"
        count={macCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title="macOS"
        path={PATHS.MANAGE_HOSTS_LABEL(macLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  const renderWindowsCard = (teamId?: number) => {
    const windowsLabelId = getBuiltinLabelId("windows");

    if (hidePlatformCard(windowsCount)) {
      return null;
    }

    if (isLoadingHostsSummary || windowsLabelId === undefined) {
      return <></>;
    }
    return (
      <HostCountCard
        iconName="windows"
        count={windowsCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title="Windows"
        path={PATHS.MANAGE_HOSTS_LABEL(windowsLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  const renderLinuxCard = (teamId?: number) => {
    const linuxLabelId = getBuiltinLabelId("linux");

    if (hidePlatformCard(linuxCount)) {
      return null;
    }

    if (isLoadingHostsSummary || linuxLabelId === undefined) {
      return <></>;
    }
    return (
      <HostCountCard
        iconName="linux"
        count={linuxCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title="Linux"
        path={PATHS.MANAGE_HOSTS_LABEL(linuxLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  const renderChromeCard = (teamId?: number) => {
    const chromeLabelId = getBuiltinLabelId("chrome");

    if (hidePlatformCard(chromeCount)) {
      return null;
    }

    if (isLoadingHostsSummary || chromeLabelId === undefined) {
      return <></>;
    }

    return (
      <HostCountCard
        iconName="chrome"
        count={chromeCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title="Chromebooks"
        path={PATHS.MANAGE_HOSTS_LABEL(chromeLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  const renderIosCount = (teamId?: number) => {
    const iosLabelId = getBuiltinLabelId("ios");

    if (hidePlatformCard(iosCount)) {
      return null;
    }

    if (isLoadingHostsSummary || iosLabelId === undefined) {
      return <></>;
    }

    return (
      <HostCountCard
        iconName="iOS"
        count={iosCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title="iPhones"
        path={PATHS.MANAGE_HOSTS_LABEL(iosLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  const renderIpadosCount = (teamId?: number) => {
    const ipadosLabelId = getBuiltinLabelId("ipados");

    if (hidePlatformCard(ipadosCount)) {
      return null;
    }

    if (isLoadingHostsSummary || ipadosLabelId === undefined) {
      return <></>;
    }

    return (
      <HostCountCard
        iconName="iPadOS"
        count={ipadosCount}
        isLoading={isLoadingHostsSummary}
        showUI={showHostsUI}
        title="iPads"
        path={PATHS.MANAGE_HOSTS_LABEL(ipadosLabelId).concat(
          teamId !== undefined ? `?team_id=${teamId}` : ""
        )}
      />
    );
  };

  const renderCounts = (teamId?: number) => {
    switch (selectedPlatform) {
      case "darwin":
        return renderMacCard(teamId);
      case "windows":
        return renderWindowsCard(teamId);
      case "linux":
        return renderLinuxCard(teamId);
      case "chrome":
        return renderChromeCard(teamId);
      case "ios":
        return renderIosCount(teamId);
      case "ipados":
        return renderIpadosCount(teamId);
      default:
        return (
          <>
            {renderMacCard(teamId)}
            {renderWindowsCard(teamId)}
            {renderLinuxCard(teamId)}
            {renderChromeCard(teamId)}
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
    <div className={baseClass} style={opacity}>
      {renderCounts(currentTeamId)}
    </div>
  );
};

export default PlatformHostCounts;
