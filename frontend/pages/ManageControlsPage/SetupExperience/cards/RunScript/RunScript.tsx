import React, { useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
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

import SetupExperiencePreview from "./components/SetupExperienceScriptPreview";
import SetupExperienceScriptUploader from "./components/SetupExperienceScriptUploader";
import SetupExperienceScriptCard from "./components/SetupExperienceScriptCard";
import DeleteSetupExperienceScriptModal from "./components/DeleteSetupExperienceScriptModal";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import { getManualAgentInstallSetting } from "../BootstrapPackage/BootstrapPackage";

const baseClass = "run-script";

interface IRunScriptProps {
  currentTeamId: number;
}

const RunScript = ({ currentTeamId }: IRunScriptProps) => {
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

  const renderContent = () => {
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
            Upload a script to run on hosts that automatically enroll to Fleet.
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
      </SetupExperienceContentContainer>
    );
  };

  return (
    <section className={baseClass}>
      <SectionHeader title="Run script" />
      <>{renderContent()}</>
      {showDeleteScriptModal && script && (
        <DeleteSetupExperienceScriptModal
          currentTeamId={currentTeamId}
          scriptName={script.name}
          onDeleted={onDelete}
          onExit={() => setShowDeleteScriptModal(false)}
        />
      )}
    </section>
  );
};

export default RunScript;
