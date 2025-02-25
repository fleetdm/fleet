import React from "react";

import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import { IConfig } from "interfaces/config";
import { ApplePlatform } from "interfaces/platform";

import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";

import WindowsTargetForm from "../WindowsTargetForm";
import PlatformTabs from "../PlatformTabs";
import { OSUpdatesSupportedPlatform } from "../../OSUpdates";

const baseClass = "os-updates-target-section";

type GetDefaultFnParams = {
  osType?: ApplePlatform;
  currentTeamId: number;
  appConfig: IConfig;
  teamConfig?: ITeamConfig;
};

const getDefaultOSVersion = ({
  osType,
  currentTeamId,
  appConfig,
  teamConfig,
}: GetDefaultFnParams) => {
  const mdmData =
    currentTeamId === API_NO_TEAM_ID ? appConfig?.mdm : teamConfig?.mdm;

  switch (osType) {
    case "darwin":
      return mdmData?.macos_updates.minimum_version ?? "";
    case "ios":
      return mdmData?.ios_updates.minimum_version ?? "";
    case "ipados":
      return mdmData?.ipados_updates.minimum_version ?? "";
    default:
      return "";
  }
};

const getDefaultDeadline = ({
  osType,
  currentTeamId,
  appConfig,
  teamConfig,
}: GetDefaultFnParams) => {
  const mdmData =
    currentTeamId === API_NO_TEAM_ID ? appConfig?.mdm : teamConfig?.mdm;

  switch (osType) {
    case "darwin":
      return mdmData?.macos_updates.deadline ?? "";
    case "ios":
      return mdmData?.ios_updates.deadline ?? "";
    case "ipados":
      return mdmData?.ipados_updates.deadline ?? "";
    default:
      return "";
  }
};

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
  selectedPlatform: OSUpdatesSupportedPlatform;
  teamConfig?: ITeamConfig;
  onSelectPlatform: (platform: OSUpdatesSupportedPlatform) => void;
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
    if (isAppleMdmEnabled) {
      return (
        <PlatformTabs
          currentTeamId={currentTeamId}
          defaultMacOSVersion={defaultMacOSVersion}
          defaultMacOSDeadline={defaultMacOSDeadline}
          defaultIOSVersion={defaultIOSVersion}
          defaultIOSDeadline={defaultIOSDeadline}
          defaultIPadOSVersion={defaultIPadOSOSVersion}
          defaultIPadOSDeadline={defaultIPadOSDeadline}
          defaultWindowsDeadlineDays={defaultWindowsDeadlineDays}
          defaultWindowsGracePeriodDays={defaultWindowsGracePeriodDays}
          selectedPlatform={selectedPlatform}
          onSelectPlatform={onSelectPlatform}
          refetchAppConfig={refetchAppConfig}
          refetchTeamConfig={refetchTeamConfig}
          isWindowsMdmEnabled={isWindowsMdmEnabled}
        />
      );
    }
    return (
      <WindowsTargetForm
        currentTeamId={currentTeamId}
        defaultDeadlineDays={defaultWindowsDeadlineDays}
        defaultGracePeriodDays={defaultWindowsGracePeriodDays}
        refetchAppConfig={refetchAppConfig}
        refetchTeamConfig={refetchTeamConfig}
      />
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader
        title="Target"
        wrapperCustomClass={`${baseClass}__header`}
      />
      {renderTargetForms()}
    </div>
  );
};

export default TargetSection;
