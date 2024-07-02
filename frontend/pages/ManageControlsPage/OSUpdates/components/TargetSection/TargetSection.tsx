import React from "react";

import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import { IConfig } from "interfaces/config";

import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";

import MacOSTargetForm from "../MacOSTargetForm";
import WindowsTargetForm from "../WindowsTargetForm";
import PlatformTabs from "../PlatformTabs";
import { OSUpdatesSupportedPlatform } from "../../OSUpdates";

const baseClass = "os-updates-target-section";

type GetDefaultFnParams = {
  currentTeamId: number;
  appConfig: IConfig;
  teamConfig?: ITeamConfig;
};

const getDefaultMacOSVersion = ({
  currentTeamId,
  appConfig,
  teamConfig,
}: GetDefaultFnParams) => {
  return currentTeamId === API_NO_TEAM_ID
    ? appConfig?.mdm.macos_updates.minimum_version ?? ""
    : teamConfig?.mdm?.macos_updates.minimum_version ?? "";
};

const getDefaultMacOSDeadline = ({
  currentTeamId,
  appConfig,
  teamConfig,
}: GetDefaultFnParams) => {
  return currentTeamId === API_NO_TEAM_ID
    ? appConfig?.mdm.macos_updates.deadline || ""
    : teamConfig?.mdm?.macos_updates.deadline || "";
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

  const isMacMdmEnabled = appConfig.mdm.enabled_and_configured;
  const isWindowsMdmEnabled = appConfig.mdm.windows_enabled_and_configured;

  const defaultMacOSVersion = getDefaultMacOSVersion({
    currentTeamId,
    appConfig,
    teamConfig,
  });
  const defaultMacOSDeadline = getDefaultMacOSDeadline({
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
    if (!isMacMdmEnabled && !isWindowsMdmEnabled) {
      // if (isMacMdmEnabled && isWindowsMdmEnabled) {
      return (
        <PlatformTabs
          currentTeamId={currentTeamId}
          defaultMacOSVersion={defaultMacOSVersion}
          defaultMacOSDeadline={defaultMacOSDeadline}
          defaultWindowsDeadlineDays={defaultWindowsDeadlineDays}
          defaultWindowsGracePeriodDays={defaultWindowsGracePeriodDays}
          selectedPlatform={selectedPlatform}
          onSelectPlatform={onSelectPlatform}
          refetchAppConfig={refetchAppConfig}
          refetchTeamConfig={refetchTeamConfig}
        />
      );
    } else if (isMacMdmEnabled) {
      return (
        <MacOSTargetForm
          currentTeamId={currentTeamId}
          defaultMinOsVersion={defaultMacOSVersion}
          defaultDeadline={defaultMacOSDeadline}
          refetchAppConfig={refetchAppConfig}
          refetchTeamConfig={refetchTeamConfig}
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
      <SectionHeader title="Target" className={`${baseClass}__header`} />
      {renderTargetForms()}
    </div>
  );
};

export default TargetSection;
