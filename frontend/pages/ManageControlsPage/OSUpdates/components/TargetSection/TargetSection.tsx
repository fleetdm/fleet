import React from "react";

import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import { IConfig } from "interfaces/config";
import { ApplePlatform } from "interfaces/platform";

import Spinner from "components/Spinner";

import WindowsTargetForm from "../WindowsTargetForm";
import PlatformTabs from "../PlatformTabs";
import { OSUpdatesTargetPlatform } from "../../OSUpdates";

const baseClass = "os-updates-target-section";

type GetDefaultFnParams = {
  osType?: ApplePlatform;
  currentTeamId: number;
  appConfig: IConfig;
  teamConfig?: ITeamConfig;
};

/** Resolves the OS update settings for an Apple platform from the team config,
 * or the app config when viewing "No team". */
const getAppleOSUpdateSettings = ({
  osType,
  currentTeamId,
  appConfig,
  teamConfig,
}: GetDefaultFnParams) => {
  const mdmData =
    currentTeamId === API_NO_TEAM_ID ? appConfig?.mdm : teamConfig?.mdm;

  switch (osType) {
    case "darwin":
      return mdmData?.macos_updates;
    case "ios":
      return mdmData?.ios_updates;
    case "ipados":
      return mdmData?.ipados_updates;
    case "tvos":
      return mdmData?.tvos_updates;
    default:
      return undefined;
  }
};

const getDefaultUpdateNewHosts = (params: GetDefaultFnParams) =>
  !!getAppleOSUpdateSettings(params)?.update_new_hosts;

/** deadline_days is only set in "latest" mode; an empty string means unset,
 * matching how the version and deadline defaults are handled. */
const getDefaultAppleDeadlineDays = (params: GetDefaultFnParams) =>
  getAppleOSUpdateSettings(params)?.deadline_days?.toString() ?? "";

const getDefaultOSVersion = (params: GetDefaultFnParams) =>
  getAppleOSUpdateSettings(params)?.minimum_version ?? "";

const getDefaultDeadline = (params: GetDefaultFnParams) =>
  getAppleOSUpdateSettings(params)?.deadline ?? "";

const getDefaultWindowsDeadlineDays = ({
  currentTeamId,
  appConfig,
  teamConfig,
}: GetDefaultFnParams) => {
  return currentTeamId === API_NO_TEAM_ID
    ? appConfig.mdm.windows_updates.deadline_days?.toString() ?? ""
    : teamConfig?.mdm?.windows_updates.deadline_days?.toString() ?? "";
};

const getDefaultWindowsGracePeriodDays = ({
  currentTeamId,
  appConfig,
  teamConfig,
}: GetDefaultFnParams) => {
  return currentTeamId === API_NO_TEAM_ID
    ? appConfig.mdm.windows_updates.grace_period_days?.toString() ?? ""
    : teamConfig?.mdm?.windows_updates.grace_period_days?.toString() ?? "";
};

interface ITargetSectionProps {
  appConfig: IConfig;
  currentTeamId: number;
  isFetching: boolean;
  selectedPlatform: OSUpdatesTargetPlatform;
  teamConfig?: ITeamConfig;
  onSelectPlatform: (platform: OSUpdatesTargetPlatform) => void;
  refetchAppConfig: () => void;
  refetchTeamConfig: () => void;
}

const TargetSection = ({
  appConfig,
  currentTeamId,
  isFetching,
  selectedPlatform,
  teamConfig,
  onSelectPlatform,
  refetchAppConfig,
  refetchTeamConfig,
}: ITargetSectionProps) => {
  if (isFetching) {
    return <Spinner />;
  }

  const isAndroidMdmEnabled = appConfig.mdm.android_enabled_and_configured;

  const isAppleMdmEnabled = appConfig.mdm.enabled_and_configured;

  const isWindowsMdmEnabled = appConfig.mdm.windows_enabled_and_configured;

  const defaultMacOSVersion = getDefaultOSVersion({
    osType: "darwin",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultMacOSDeadline = getDefaultDeadline({
    osType: "darwin",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultIOSVersion = getDefaultOSVersion({
    osType: "ios",
    currentTeamId,
    appConfig,
    teamConfig,
  });

  const defaultIOSDeadline = getDefaultDeadline({
    osType: "ios",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultIPadOSOSVersion = getDefaultOSVersion({
    osType: "ipados",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultIPadOSDeadline = getDefaultDeadline({
    osType: "ipados",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultMacOSDeadlineDays = getDefaultAppleDeadlineDays({
    osType: "darwin",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultIOSDeadlineDays = getDefaultAppleDeadlineDays({
    osType: "ios",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultIPadOSDeadlineDays = getDefaultAppleDeadlineDays({
    osType: "ipados",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultTvOSVersion = getDefaultOSVersion({
    osType: "tvos",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultTvOSDeadline = getDefaultDeadline({
    osType: "tvos",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultTvOSDeadlineDays = getDefaultAppleDeadlineDays({
    osType: "tvos",
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultMacOSUpdateNewHosts = getDefaultUpdateNewHosts({
    osType: "darwin",
    currentTeamId,
    appConfig,
    teamConfig,
  });

  const defaultWindowsDeadlineDays = getDefaultWindowsDeadlineDays({
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultWindowsGracePeriodDays = getDefaultWindowsGracePeriodDays({
    currentTeamId,
    appConfig,
    teamConfig,
  });

  const renderTargetForms = () => {
    if (isWindowsMdmEnabled && !isAppleMdmEnabled && !isAndroidMdmEnabled) {
      return (
        <WindowsTargetForm
          currentTeamId={currentTeamId}
          defaultDeadlineDays={defaultWindowsDeadlineDays}
          defaultGracePeriodDays={defaultWindowsGracePeriodDays}
          refetchAppConfig={refetchAppConfig}
          refetchTeamConfig={refetchTeamConfig}
        />
      );
    }
    return (
      <PlatformTabs
        currentTeamId={currentTeamId}
        defaultMacOSVersion={defaultMacOSVersion}
        defaultMacOSDeadline={defaultMacOSDeadline}
        defaultMacOSDeadlineDays={defaultMacOSDeadlineDays}
        defaultIOSVersion={defaultIOSVersion}
        defaultIOSDeadline={defaultIOSDeadline}
        defaultIOSDeadlineDays={defaultIOSDeadlineDays}
        defaultIPadOSVersion={defaultIPadOSOSVersion}
        defaultIPadOSDeadline={defaultIPadOSDeadline}
        defaultIPadOSDeadlineDays={defaultIPadOSDeadlineDays}
        defaultTvOSVersion={defaultTvOSVersion}
        defaultTvOSDeadline={defaultTvOSDeadline}
        defaultTvOSDeadlineDays={defaultTvOSDeadlineDays}
        defaultWindowsDeadlineDays={defaultWindowsDeadlineDays}
        defaultWindowsGracePeriodDays={defaultWindowsGracePeriodDays}
        defaultMacOSUpdateNewHosts={defaultMacOSUpdateNewHosts}
        selectedPlatform={selectedPlatform}
        onSelectPlatform={onSelectPlatform}
        refetchAppConfig={refetchAppConfig}
        refetchTeamConfig={refetchTeamConfig}
        isWindowsMdmEnabled={isWindowsMdmEnabled}
        isAndroidMdmEnabled={isAndroidMdmEnabled}
      />
    );
  };

  return <div className={baseClass}>{renderTargetForms()}</div>;
};

export default TargetSection;
