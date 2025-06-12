import React, { useCallback } from "react";
import PATHS from "router/paths";

import { getPathWithQueryParams } from "utilities/url";
import { PLATFORM_NAME_TO_LABEL_NAME } from "pages/DashboardPage/helpers";

import { IHostSummary } from "interfaces/host_summary";
import { PlatformValueOptions } from "utilities/constants";
import DataError from "components/DataError";
import Card from "components/Card";

import HostCountCard from "../../cards/HostCountCard";

const baseClass = "platform-host-counts";

interface IPlatformHostCountsProps {
  androidDevEnabled: boolean; // TODO(android): remove when feature flag is removed
  currentTeamId: number | undefined;
  macCount: number;
  windowsCount: number;
  linuxCount: number;
  chromeCount: number;
  iosCount: number;
  ipadosCount: number;
  androidCount: number;
  builtInLabels?: IHostSummary["builtin_labels"];
  selectedPlatform?: PlatformValueOptions;
  totalHostCount?: number;
}

const PlatformHostCounts = ({
  androidDevEnabled,
  currentTeamId,
  macCount,
  windowsCount,
  linuxCount,
  chromeCount,
  iosCount,
  ipadosCount,
  androidCount,
  builtInLabels,
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
        path={getPathWithQueryParams(PATHS.MANAGE_HOSTS_LABEL(macLabelId), {
          team_id: teamId,
        })}
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
        path={getPathWithQueryParams(PATHS.MANAGE_HOSTS_LABEL(windowsLabelId), {
          team_id: teamId,
        })}
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
        path={getPathWithQueryParams(PATHS.MANAGE_HOSTS_LABEL(linuxLabelId), {
          team_id: teamId,
        })}
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
        title="ChromeOS"
        path={getPathWithQueryParams(PATHS.MANAGE_HOSTS_LABEL(chromeLabelId), {
          team_id: teamId,
        })}
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
        title="iOS"
        path={getPathWithQueryParams(PATHS.MANAGE_HOSTS_LABEL(iosLabelId), {
          team_id: teamId,
        })}
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
        title="iPadOS"
        path={getPathWithQueryParams(PATHS.MANAGE_HOSTS_LABEL(ipadosLabelId), {
          team_id: teamId,
        })}
      />
    );
  };

  const renderAndroidCount = (teamId?: number) => {
    if (!androidDevEnabled) {
      // TODO(android): remove when feature flag is removed
      return null;
    }

    const androidLabelId = getBuiltinLabelId("android");

    if (hidePlatformCard(androidCount)) {
      return null;
    }

    if (androidLabelId === undefined) {
      return <></>;
    }

    return (
      <HostCountCard
        iconName="android"
        count={androidCount}
        title="Android"
        path={PATHS.MANAGE_HOSTS_LABEL(androidLabelId).concat(
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
      case "android":
        return renderAndroidCount(teamId);
      default:
        // TODO(android): responsive layout with variable column widths (see figma for 2x2x3 grid)
        return (
          <>
            {renderMacCard(teamId)}
            {renderWindowsCard(teamId)}
            {renderLinuxCard(teamId)}
            {renderChromeCard(teamId)}
            {renderIosCount(teamId)}
            {renderIpadosCount(teamId)}
            {renderAndroidCount(teamId)}
          </>
        );
    }
  };

  return <div className={baseClass}>{renderCounts(currentTeamId)}</div>;
};

export default PlatformHostCounts;
