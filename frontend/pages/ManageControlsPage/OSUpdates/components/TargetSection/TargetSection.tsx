import React, { useContext } from "react";
import { useQuery } from "react-query";
import {
  Accordion,
  AccordionItem,
  AccordionItemButton,
  AccordionItemHeading,
  AccordionItemPanel,
} from "react-accessible-accordion";

import {
  API_NO_TEAM_ID,
  APP_CONTEXT_NO_TEAM_ID,
  ITeamConfig,
} from "interfaces/team";
import { IConfig } from "interfaces/config";
import { AppContext } from "context/app";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import Icon from "components/Icon";

import MacOSTargetForm from "../MacOSTargetForm";
import WindowsTargetForm from "../WindowsTargetForm";

const baseClass = "os-updates-target-section";

interface ISectionAccordionProps {
  currentTeamId: number;
  defaultMacOSVersion: string;
  defaultMacOSDeadline: string;
  defaultWindowsDeadlineDays: string;
  defaultWindowsGracePeriodDays: string;
}

const SectionAccordion = ({
  currentTeamId,
  defaultMacOSDeadline,
  defaultMacOSVersion,
  defaultWindowsDeadlineDays,
  defaultWindowsGracePeriodDays,
}: ISectionAccordionProps) => {
  return (
    <Accordion
      className={`${baseClass}__`}
      allowZeroExpanded
      preExpanded={["mac"]}
    >
      <AccordionItem uuid="mac">
        <AccordionItemHeading>
          <AccordionItemButton className={`${baseClass}__accordion-button`}>
            <span>macOS</span>
            <Icon name="chevron" direction="up" />
          </AccordionItemButton>
        </AccordionItemHeading>
        <AccordionItemPanel>
          <MacOSTargetForm
            currentTeamId={currentTeamId}
            defaultMinOsVersion={defaultMacOSVersion}
            defaultDeadline={defaultMacOSDeadline}
            key={currentTeamId}
            inAccordion
          />
        </AccordionItemPanel>
      </AccordionItem>
      <AccordionItem uuid="windows">
        <AccordionItemHeading>
          <AccordionItemButton className={`${baseClass}__accordion-button`}>
            <span>Windows</span>
            <Icon name="chevron" direction="up" />
          </AccordionItemButton>
        </AccordionItemHeading>
        <AccordionItemPanel>
          <WindowsTargetForm
            currentTeamId={currentTeamId}
            defaultDeadlineDays={defaultWindowsDeadlineDays}
            defaultGracePeriodDays={defaultWindowsGracePeriodDays}
            key={currentTeamId}
            inAccordion
          />
        </AccordionItemPanel>
      </AccordionItem>
    </Accordion>
  );
};

interface ITargetSectionProps {
  currentTeamId: number;
}

const getDefaultMacOSVersion = (
  currentTeam: number,
  appConfig: IConfig,
  teamConfig?: ITeamConfig
) => {
  return currentTeam === API_NO_TEAM_ID
    ? appConfig?.mdm.macos_updates.minimum_version ?? ""
    : teamConfig?.mdm?.macos_updates.minimum_version ?? "";
};

const getDefaultMacOSDeadline = (
  currentTeam: number,
  appConfig: IConfig,
  teamConfig?: ITeamConfig
) => {
  return currentTeam === API_NO_TEAM_ID
    ? appConfig?.mdm.macos_updates.deadline || ""
    : teamConfig?.mdm?.macos_updates.deadline || "";
};

const getDefaultWindowsDeadlineDays = (
  currentTeam: number,
  appConfig: IConfig,
  teamConfig?: ITeamConfig
) => {
  return currentTeam === API_NO_TEAM_ID
    ? appConfig.mdm.windows_updates.deadline_days?.toString() ?? ""
    : teamConfig?.mdm?.windows_updates.deadline_days?.toString() ?? "";
};

const getDefaultWindowsGracePeriodDays = (
  currentTeam: number,
  appConfig: IConfig,
  teamConfig?: ITeamConfig
) => {
  return currentTeam === API_NO_TEAM_ID
    ? appConfig.mdm.windows_updates.grace_period_days?.toString() ?? ""
    : teamConfig?.mdm?.windows_updates.grace_period_days?.toString() ?? "";
};

const TargetSection = ({ currentTeamId }: ITargetSectionProps) => {
  const { config } = useContext(AppContext);

  const { data: teamData, isLoading: isLoadingTeam, isError } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team-config", currentTeamId], () => teamsAPI.load(currentTeamId), {
    refetchOnWindowFocus: false,
    enabled: currentTeamId > APP_CONTEXT_NO_TEAM_ID,
    select: (data) => data.team,
  });

  if (!config) return null;

  const isMacMdmEnabled = config?.mdm.enabled_and_configured;
  // const isWindowsMdmEnabled = config?.mdm.windows_enabled_and_configured;
  const isWindowsMdmEnabled = true;

  const defaultWindowsDeadlineDays = "3";
  const defaultWindowsGracePeriodDays = "5";

  // Loading state rendering
  if (isLoadingTeam) {
    return <Spinner />;
  }

  const defaultMacOSVersion = getDefaultMacOSVersion(
    currentTeamId,
    config,
    teamData
  );
  const defaultMacOSDeadline = getDefaultMacOSDeadline(
    currentTeamId,
    config,
    teamData
  );

  const renderTargetForms = () => {
    if (isMacMdmEnabled && isWindowsMdmEnabled) {
      return (
        <SectionAccordion
          currentTeamId={currentTeamId}
          defaultMacOSVersion={defaultMacOSVersion}
          defaultMacOSDeadline={defaultMacOSDeadline}
          defaultWindowsDeadlineDays={defaultWindowsDeadlineDays}
          defaultWindowsGracePeriodDays={defaultWindowsGracePeriodDays}
        />
      );
    } else if (isMacMdmEnabled) {
      return (
        <MacOSTargetForm
          currentTeamId={currentTeamId}
          defaultMinOsVersion={defaultMacOSVersion}
          defaultDeadline={defaultMacOSDeadline}
        />
      );
    }
    return (
      <WindowsTargetForm
        currentTeamId={currentTeamId}
        defaultDeadlineDays={defaultWindowsDeadlineDays}
        defaultGracePeriodDays={defaultWindowsGracePeriodDays}
      />
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Target" />
      {renderTargetForms()}
    </div>
  );
};

export default TargetSection;
0;
