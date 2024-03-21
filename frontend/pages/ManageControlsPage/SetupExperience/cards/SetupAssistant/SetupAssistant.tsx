import React, { useState } from "react";
import { useQuery } from "react-query";

import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";

import SetupAssistantPreview from "./components/SetupAssistantPreview";
import SetupAssistantProfileUploader from "./components/SetupAssistantProfileUploader";
import SetuAssistantProfileCard from "./components/SetupAssistantProfileCard/SetupAssistantProfileCard";
import DeleteAutoEnrollmentProfile from "./components/DeleteAutoEnrollmentProfile";
import AdvancedOptionsForm from "./components/AdvancedOptionsForm";

const baseClass = "setup-assistant";

interface ISetupAssistantProps {
  currentTeamId: number;
}

const StartupAssistant = ({ currentTeamId }: ISetupAssistantProps) => {
  const [showDeleteProfileModal, setShowDeleteProfileModal] = useState(false);

  const { data: globalConfig, isLoading: isLoadingGlobalConfig } = useQuery<
    IConfig,
    Error
  >(["config", currentTeamId], () => configAPI.loadAll(), {
    enabled: currentTeamId === API_NO_TEAM_ID,
    refetchOnWindowFocus: false,
    retry: false,
  });

  const { data: teamConfig, isLoading: isLoadingTeamConfig } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team", currentTeamId], () => teamsAPI.load(currentTeamId), {
    refetchOnWindowFocus: false,
    retry: false,
    enabled: currentTeamId !== API_NO_TEAM_ID,
    select: (res) => res.team,
  });

  const isLoading = false;

  const noPackageUploaded = true;

  const onUpload = () => {};

  const onDelete = () => {};

  return (
    <div className={baseClass}>
      <SectionHeader title="Setup assistant" />
      {isLoading ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__content`}>
          <div className={`${baseClass}__upload-container`}>
            <p className={`${baseClass}__section-description`}>
              Add an automatic enrollment profile to customize the macOS Setup
              Assistant.
              <CustomLink
                url=" https://fleetdm.com/learn-more-about/setup-assistant"
                text="Learn how"
                newTab
              />
            </p>
            {true ? (
              <SetupAssistantProfileUploader
                currentTeamId={currentTeamId}
                onUpload={() => 1}
              />
            ) : (
              <SetuAssistantProfileCard
                profileMetaData={1}
                currentTeamId={currentTeamId}
                onDelete={() => setShowDeleteProfileModal(true)}
              />
            )}
            <AdvancedOptionsForm />
          </div>
          <div className={`${baseClass}__preview-container`}>
            <SetupAssistantPreview />
          </div>
        </div>
      )}
      {showDeleteProfileModal && (
        <DeleteAutoEnrollmentProfile
          onDelete={onDelete}
          onCancel={() => setShowDeleteProfileModal(false)}
        />
      )}
    </div>
  );
};

export default StartupAssistant;
