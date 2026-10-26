import React, { useContext, useEffect } from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";

import BackButton from "components/BackButton";
import DataError from "components/DataError";
import InfoBanner from "components/InfoBanner";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import { AppContext } from "context/app";
import useGitOpsMode from "hooks/useGitOpsMode";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IMdmProfile, isAnyMDMConfigured } from "interfaces/mdm";
import { API_NO_TEAM_ID } from "interfaces/team";
import PATHS from "router/paths";
import configProfileAPI from "services/entities/config_profiles";
import mdmAPI from "services/entities/mdm";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

import AIProfileGenerator, {
  getAIPlatformOptions,
} from "./components/AIProfileGenerator";
import ProfileForm from "./components/ProfileForm";

const baseClass = "profile-form-page";

interface IProfileFormPageRouteParams {
  profile_uuid?: string;
}

type IProfileFormPageProps = RouteComponentProps<
  undefined,
  IProfileFormPageRouteParams
>;

/** Add (no profile_uuid) or edit (profile_uuid) a configuration profile as a
 * full page, with the contents editable inline. */
const ProfileFormPage = ({
  router,
  routeParams,
  location,
}: IProfileFormPageProps) => {
  const profileUUID = routeParams.profile_uuid;
  const {
    config,
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
  } = useContext(AppContext);
  const { gitOpsModeEnabled } = useGitOpsMode();

  const { currentTeamName, teamIdForApi, userTeams } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: true,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: true,
      observer: false,
      observer_plus: false,
      technician: false,
    },
  });
  // Same coercion as ManageControlsPage: free tier only has "No fleet".
  const teamId = isPremiumTier === false ? API_NO_TEAM_ID : teamIdForApi;

  const listPath = getPathWithQueryParams(PATHS.CONTROLS_CUSTOM_SETTINGS, {
    fleet_id: isPremiumTier ? teamId : undefined,
  });

  // Adding defaults to the AI view. The manual view has its own URL, so the
  // AI view's timeout can send the admin there and Back returns.
  const isManualPath =
    location.pathname === PATHS.CONTROLS_CUSTOM_SETTINGS_NEW_MANUAL;
  const showAIView =
    !profileUUID &&
    !isManualPath &&
    !gitOpsModeEnabled &&
    !config?.server_settings.ai_features_disabled &&
    getAIPlatformOptions(config?.mdm).length > 0;
  const aiTimedOut =
    isManualPath &&
    !!(location.state as { aiTimedOut?: boolean } | undefined)?.aiTimedOut;

  const {
    data: profile,
    isLoading: isLoadingProfile,
    isError: isErrorProfile,
  } = useQuery<IMdmProfile, Error>(
    ["config_profile", profileUUID],
    () => configProfileAPI.getConfigProfile(profileUUID as string),
    // The form is seeded once from this data, so a cached copy from before
    // the last save would be edited and written back. Always fetch fresh.
    { ...DEFAULT_USE_QUERY_OPTIONS, enabled: !!profileUUID, cacheTime: 0 }
  );
  const {
    data: contents,
    isLoading: isLoadingContents,
    isError: isErrorContents,
  } = useQuery<string, Error>(
    ["config_profile_contents", profileUUID],
    // as text, so JSON profiles are edited byte for byte rather than as a
    // re-serialized copy
    () => mdmAPI.downloadProfile(profileUUID as string, "text"),
    { ...DEFAULT_USE_QUERY_OPTIONS, enabled: !!profileUUID, cacheTime: 0 }
  );

  // The edit URL is shareable, so its fleet_id can disagree with the fleet
  // the profile belongs to; follow the profile so labels and the footer match.
  const profileTeamId = profile ? profile.team_id ?? API_NO_TEAM_ID : undefined;
  const isWrongFleet =
    !!isPremiumTier &&
    profileTeamId !== undefined &&
    teamId !== undefined &&
    profileTeamId !== teamId;
  // useTeamIdParam sends the user back from a fleet they can't manage, so
  // following the profile there would bounce between the two forever.
  const canManageProfileFleet =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    !!userTeams?.some((team) => team.id === profileTeamId);
  useEffect(() => {
    if (isWrongFleet && canManageProfileFleet) {
      router.replace(
        getPathWithQueryParams(location.pathname, {
          fleet_id: profileTeamId,
        })
      );
    }
  }, [
    isWrongFleet,
    canManageProfileFleet,
    location.pathname,
    profileTeamId,
    router,
  ]);

  // The roles useTeamIdParam allows above; anyone else has no fleet to land
  // on and would otherwise wait on a spinner.
  const canEditProfiles =
    isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainerOrTeamAdmin;

  // Only adding is sent away; viewing an existing profile is still useful in
  // GitOps mode or with its MDM off.
  const shouldLeaveAddPage =
    !profileUUID &&
    !!config &&
    (gitOpsModeEnabled || !canEditProfiles || !isAnyMDMConfigured(config.mdm));
  // A user with no fleet to manage has no fleet ID yet, so keep the URL's.
  const leaveAddPagePath = getPathWithQueryParams(
    PATHS.CONTROLS_CUSTOM_SETTINGS,
    { fleet_id: isPremiumTier ? teamId ?? location.query.fleet_id : undefined }
  );
  useEffect(() => {
    if (shouldLeaveAddPage) {
      router.replace(leaveAddPagePath);
    }
  }, [shouldLeaveAddPage, leaveAddPagePath, router]);

  const renderContent = () => {
    if (shouldLeaveAddPage) {
      return <Spinner />;
    }
    if (!canEditProfiles) {
      return (
        <DataError description="You don't have permission to edit profiles." />
      );
    }
    if (teamId === undefined) {
      return <Spinner />;
    }
    if (isWrongFleet && !canManageProfileFleet) {
      return (
        <DataError description="You don't have permission to edit this profile." />
      );
    }
    if (
      profileUUID &&
      (isLoadingProfile || isLoadingContents || isWrongFleet)
    ) {
      return <Spinner />;
    }
    if (profileUUID && (isErrorProfile || isErrorContents || !profile)) {
      return <DataError description="Couldn't load the profile." />;
    }
    if (showAIView) {
      return (
        <AIProfileGenerator
          router={router}
          teamId={teamId}
          teamName={currentTeamName}
          isPremiumTier={!!isPremiumTier}
          gitOpsModeEnabled={gitOpsModeEnabled}
          listPath={listPath}
          manualLocation={{
            pathname: PATHS.CONTROLS_CUSTOM_SETTINGS_NEW_MANUAL,
            query: isPremiumTier ? { fleet_id: teamId } : {},
          }}
        />
      );
    }
    return (
      <ProfileForm
        // remount when switching profiles so the form re-seeds
        key={profileUUID ?? "new"}
        router={router}
        teamId={teamId}
        teamName={currentTeamName}
        isPremiumTier={!!isPremiumTier}
        gitOpsModeEnabled={gitOpsModeEnabled}
        listPath={listPath}
        editing={profile ? { profile, contents: contents ?? "" } : undefined}
      />
    );
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__header-links`}>
        <BackButton text="Back to profiles" path={listPath} />
      </div>
      {/* The add page goes straight into the profile, per the design. */}
      {profileUUID && <h1>Edit profile</h1>}
      {aiTimedOut && (
        <InfoBanner color="yellow" icon="warning">
          Profile creation was taking too long. Upload or paste a profile
          instead.
        </InfoBanner>
      )}
      {renderContent()}
    </MainContent>
  );
};

export default ProfileFormPage;
