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
  builtInLabels?: IHostSummary["builtin_labels"];
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
  builtInLabels,
  errorHosts,
  selectedPlatform,
  totalHostCount,
}: IPlatformHostCountsProps): JSX.Element => {
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

    if (macLabelId === undefined) {
      return <></>;
    }

    return (
      <HostCountCard
        iconName="darwin"
        count={macCount}
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

    if (windowsLabelId === undefined) {
      return <></>;
    }
    return (
      <HostCountCard
        iconName="windows"
        count={windowsCount}
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

    if (linuxLabelId === undefined) {
      return <></>;
    }
    return (
      <HostCountCard
        iconName="linux"
        count={linuxCount}
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

    if (chromeLabelId === undefined) {
      return <></>;
    }

    return (
      <HostCountCard
        iconName="chrome"
        count={chromeCount}
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

    if (iosLabelId === undefined) {
      return <></>;
    }

    return (
      <HostCountCard
        iconName="iOS"
        count={iosCount}
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

    if (ipadosLabelId === undefined) {
      return <></>;
    }

    return (
      <HostCountCard
        iconName="iPadOS"
        count={ipadosCount}
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

  if (errorHosts) {
    return <DataError card />;
  }

  return <div className={baseClass}>{renderCounts(currentTeamId)}</div>;
};

export default PlatformHostCounts;
