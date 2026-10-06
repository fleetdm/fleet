import React, { useContext, useEffect } from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";

import BackButton from "components/BackButton";
import DataError from "components/DataError";
import MainContent from "components/MainContent";
import Spinner from "components/Spinner";
import { AppContext } from "context/app";
import useGitOpsMode from "hooks/useGitOpsMode";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IMdmProfile } from "interfaces/mdm";
import { API_NO_TEAM_ID } from "interfaces/team";
import PATHS from "router/paths";
import configProfileAPI from "services/entities/config_profiles";
import mdmAPI from "services/entities/mdm";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

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

  const renderContent = () => {
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
      <h1>{profileUUID ? "Edit profile" : "Add profile"}</h1>
      {renderContent()}
    </MainContent>
  );
};

export default ProfileFormPage;
