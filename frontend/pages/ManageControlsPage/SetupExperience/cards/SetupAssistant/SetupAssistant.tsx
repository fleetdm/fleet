import React, { useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import mdmAPI, {
  IAppleSetupEnrollmentProfileResponse,
} from "services/entities/mdm";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
import TurnOnMdmMessage from "components/TurnOnMdmMessage";

import SetupAssistantProfileUploader from "./components/SetupAssistantProfileUploader";
import SetupAssistantProfileCard from "./components/SetupAssistantProfileCard/SetupAssistantProfileCard";
import DeleteAutoEnrollmentProfile from "./components/DeleteAutoEnrollmentProfile";
import AdvancedOptionsForm from "./components/AdvancedOptionsForm";
import SetupExperienceContentContainer from "../../components/SetupExperienceContentContainer";
import { ISetupExperienceCardProps } from "../../SetupExperienceNavItems";

const baseClass = "setup-assistant";

const SetupAssistant = ({
  currentTeamId,
  router,
}: ISetupExperienceCardProps) => {
  const [showDeleteProfileModal, setShowDeleteProfileModal] = useState(false);

  const { data: globalConfig, isLoading: isLoadingGlobalConfig } = useQuery<
    IConfig,
    Error
  >(["config", currentTeamId], () => configAPI.loadAll(), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    retry: false,
  });

  const { data: teamConfig, isLoading: isLoadingTeamConfig } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team", currentTeamId], () => teamsAPI.load(currentTeamId), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    refetchOnWindowFocus: false,
    retry: false,
    enabled: currentTeamId !== API_NO_TEAM_ID,
    select: (res) => res.team,
  });

  const {
    data: enrollmentProfileData,
    isLoading: isLoadingEnrollmentProfile,
    error: enrollmentProfileError,
    refetch: refetchEnrollmentProfile,
  } = useQuery<IAppleSetupEnrollmentProfileResponse, AxiosError>(
    ["enrollment_profile", currentTeamId],
    () => mdmAPI.getSetupEnrollmentProfile(currentTeamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
    }
  );

  const getReleaseDeviceSetting = () => {
    if (currentTeamId === API_NO_TEAM_ID) {
      return (
        globalConfig?.mdm.macos_setup.enable_release_device_manually || false
      );
    }
    return teamConfig?.mdm?.macos_setup.enable_release_device_manually || false;
  };

  const onUpload = () => {
    refetchEnrollmentProfile();
  };

  const onDelete = () => {
    setShowDeleteProfileModal(false);
    refetchEnrollmentProfile();
  };

  const defaultReleaseDeviceSetting = getReleaseDeviceSetting();

  const isLoading =
    isLoadingGlobalConfig || isLoadingTeamConfig || isLoadingEnrollmentProfile;
  const enrollmentProfileNotFound = enrollmentProfileError?.status === 404;

  const renderSetupAssistantView = () => {
    return (
      <SetupExperienceContentContainer>
        <div className={`${baseClass}__upload-container`}>
          <p className={`${baseClass}__section-description`}>
            Add an automatic enrollment profile to customize the macOS Setup
            Assistant.
          </p>
          {enrollmentProfileNotFound || !enrollmentProfileData ? (
            <SetupAssistantProfileUploader
              currentTeamId={currentTeamId}
              onUpload={onUpload}
            />
          ) : (
            <SetupAssistantProfileCard
              profile={enrollmentProfileData}
              onDelete={() => setShowDeleteProfileModal(true)}
            />
          )}
          <AdvancedOptionsForm
            key={String(defaultReleaseDeviceSetting)}
            currentTeamId={currentTeamId}
            defaultReleaseDevice={defaultReleaseDeviceSetting}
          />
        </div>
      </SetupExperienceContentContainer>
    );
  };

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }
    if (
      !(
        globalConfig?.mdm.enabled_and_configured &&
        globalConfig?.mdm.apple_bm_enabled_and_configured
      )
    ) {
      return (
        <TurnOnMdmMessage
          header="Additional configuration required"
          info="Supported on macOS. To customize, first turn on automatic enrollment."
          buttonText="Turn on"
          router={router}
        />
      );
    }
    return renderSetupAssistantView();
  };

  return (
    <section className={baseClass}>
      <SectionHeader
        title="Setup assistant"
        details={
          <CustomLink
            url="https://fleetdm.com/learn-more-about/setup-assistant"
            text="Preview end user experience"
            newTab
          />
        }
      />
      {renderContent()}
      {showDeleteProfileModal && (
        <DeleteAutoEnrollmentProfile
          currentTeamId={currentTeamId}
          onDelete={onDelete}
          onCancel={() => setShowDeleteProfileModal(false)}
        />
      )}
    </section>
  );
};

export default SetupAssistant;
