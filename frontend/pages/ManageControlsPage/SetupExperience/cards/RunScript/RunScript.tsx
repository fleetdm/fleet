import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import { AppContext } from "context/app";

import mdmAPI, {
  IGetSetupExperienceScriptResponse,
} from "services/entities/mdm";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";

import SectionHeader from "components/SectionHeader";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
import TurnOnMdmMessage from "components/TurnOnMdmMessage";

import SetupExperiencePreview from "./components/SetupExperienceScriptPreview";
import SetupExperienceScriptUploader from "./components/SetupExperienceScriptUploader";
import SetupExperienceScriptCard from "./components/SetupExperienceScriptCard";
import DeleteSetupExperienceScriptModal from "./components/DeleteSetupExperienceScriptModal";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import { getManualAgentInstallSetting } from "../BootstrapPackage/BootstrapPackage";
import { ISetupExperienceCardProps } from "../../SetupExperienceNavItems";
import { SetupEmptyState } from "../../SetupExperience";

const baseClass = "run-script";

const RunScriptContent = ({
  currentTeamId,
}: Pick<ISetupExperienceCardProps, "currentTeamId">) => {
  const [showDeleteScriptModal, setShowDeleteScriptModal] = useState(false);

  const {
    data: script,
    error: scriptError,
    isLoading,
    isError,
    refetch: refetchScript,
    remove: removeScriptFromCache,
  } = useQuery<IGetSetupExperienceScriptResponse, AxiosError>(
    ["setup-experience-script", currentTeamId],
    () => mdmAPI.getSetupExperienceScript(currentTeamId),
    { ...DEFAULT_USE_QUERY_OPTIONS, retry: false }
  );

  const { data: globalConfig, isLoading: isLoadingGlobalConfig } = useQuery<
    IConfig,
    Error
  >(["config", currentTeamId], () => configAPI.loadAll(), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: currentTeamId === API_NO_TEAM_ID,
  });

  const { data: teamConfig, isLoading: isLoadingTeamConfig } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team", currentTeamId], () => teamsAPI.load(currentTeamId), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: currentTeamId !== API_NO_TEAM_ID,
    select: (res) => res.team,
  });

  const onUpload = () => {
    refetchScript();
  };

  const onDelete = () => {
    removeScriptFromCache();
    setShowDeleteScriptModal(false);
    refetchScript();
  };

  const hasManualAgentInstall = getManualAgentInstallSetting(
    currentTeamId,
    globalConfig,
    teamConfig
  );

  if (isLoading || isLoadingGlobalConfig || isLoadingTeamConfig) {
    <Spinner />;
  }

  if (isError && scriptError.status !== 404) {
    return <DataError />;
  }

  return (
    <SetupExperienceContentContainer>
      <div className={`${baseClass}__description-container`}>
        <p className={`${baseClass}__description`}>
          Upload a script to run on macOS hosts that automatically enroll to
          Fleet.
        </p>
        <CustomLink
          className={`${baseClass}__learn-how-link`}
          newTab
          url={`${LEARN_MORE_ABOUT_BASE_LINK}/setup-assistant`}
          text="Learn how"
        />
        {!script ? (
          <SetupExperienceScriptUploader
            currentTeamId={currentTeamId}
            hasManualAgentInstall={hasManualAgentInstall}
            onUpload={onUpload}
          />
        ) : (
          <>
            <p className={`${baseClass}__run-message`}>
              Script will run during setup:
            </p>
            <SetupExperienceScriptCard
              script={script}
              onDelete={() => setShowDeleteScriptModal(true)}
            />
          </>
        )}
      </div>
      <SetupExperiencePreview />
      {showDeleteScriptModal && script && (
        <DeleteSetupExperienceScriptModal
          currentTeamId={currentTeamId}
          scriptName={script.name}
          onDeleted={onDelete}
          onExit={() => setShowDeleteScriptModal(false)}
        />
      )}
    </SetupExperienceContentContainer>
  );
};

const RunScript = ({ currentTeamId, router }: ISetupExperienceCardProps) => {
  const { config } = useContext(AppContext);

  const renderContent = () => {
    if (!config?.mdm.enabled_and_configured) {
      return (
        <TurnOnMdmMessage
          header="Manage setup experience for macOS"
          info="To install software and run scripts when Macs first boot, first turn on automatic enrollment."
          buttonText="Turn on"
          router={router}
        />
      );
    }
    if (!config?.mdm.apple_bm_enabled_and_configured) {
      return <SetupEmptyState router={router} />;
    }
    return <RunScriptContent currentTeamId={currentTeamId} />;
  };

  return (
    <section className={baseClass}>
      <SectionHeader title="Run script" />
      <>{renderContent()}</>
    </section>
  );
};

export default RunScript;
