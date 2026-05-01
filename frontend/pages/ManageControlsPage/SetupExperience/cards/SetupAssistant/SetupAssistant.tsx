import React, { useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import { IConfig } from "interfaces/config";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import mdmAPI, {
  IAppleSetupEnrollmentProfileResponse,
  IDefaultAppleSetupEnrollmentProfileResponse,
} from "services/entities/mdm";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import PATHS from "router/paths";

import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import Button from "components/buttons/Button";

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
    select: (res) => res.fleet,
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
  const enrollmentProfileNotFound = enrollmentProfileError?.status === 404;

  const {
    data: defaultEnrollmentProfileData,
    isLoading: isLoadingDefaultEnrollmentProfile,
  } = useQuery<IDefaultAppleSetupEnrollmentProfileResponse, AxiosError>(
    ["default_enrollment_profile", currentTeamId],
    () => mdmAPI.getDefaultSetupEnrollmentProfile(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      enabled: enrollmentProfileNotFound, // only fetch the default profile if there is no team enrollment profile
    }
  );

  const getReleaseDeviceSetting = () => {
    if (currentTeamId === API_NO_TEAM_ID) {
      return (
        globalConfig?.mdm.setup_experience
          .apple_enable_release_device_manually || false
      );
    }
    return (
      teamConfig?.mdm?.setup_experience.apple_enable_release_device_manually ||
      false
    );
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
    isLoadingGlobalConfig ||
    isLoadingTeamConfig ||
    isLoadingEnrollmentProfile ||
    isLoadingDefaultEnrollmentProfile;

  const renderSetupAssistantView = () => {
    return (
      <SetupExperienceContentContainer>
        <div className={`${baseClass}__upload-container`}>
          <p className={`${baseClass}__section-description`}>
            Add an automatic enrollment profile to customize Setup Assistant.{" "}
            <CustomLink
              url="https://fleetdm.com/learn-more-about/enrollment-profiles"
              text="Learn more"
              newTab
            />
          </p>
          {enrollmentProfileNotFound || !enrollmentProfileData ? (
            <>
              {defaultEnrollmentProfileData && (
                <SetupAssistantProfileCard
                  profile={
                    defaultEnrollmentProfileData as IAppleSetupEnrollmentProfileResponse
                  }
                  defaultProfile
                />
              )}
              <SetupAssistantProfileUploader
                currentTeamId={currentTeamId}
                onUpload={onUpload}
              />
            </>
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
        <EmptyState
          variant="form"
          header="Additional configuration required"
          info="To customize, first turn on automatic enrollment."
          primaryButton={
            <Button onClick={() => router.push(PATHS.ADMIN_INTEGRATIONS_MDM)}>
              Turn on
            </Button>
          }
        />
      );
    }
    return renderSetupAssistantView();
  };

  return (
    <section className={baseClass}>
      <SectionHeader
        title="Setup Assistant"
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
