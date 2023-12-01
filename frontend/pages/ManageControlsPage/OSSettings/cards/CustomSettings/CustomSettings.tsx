import React, { useCallback, useContext, useRef, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import { IMdmProfile } from "interfaces/mdm";
import mdmAPI, { IMdmProfilesResponse } from "services/entities/mdm";
import { NotificationContext } from "context/notification";
import PATHS from "router/paths";

import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import Pagination from "pages/ManageControlsPage/components/Pagination";

import UploadList from "../../../components/UploadList";

import DeleteProfileModal from "./components/DeleteProfileModal/DeleteProfileModal";
import ProfileListItem from "./components/ProfileListItem";
import ProfileListHeading from "./components/ProfileListHeading";
import ProfileUploader from "./components/ProfileUploader";

const PROFILES_PER_PAGE = 10;

const baseClass = "custom-settings";

interface ICustomSettingsProps {
  currentTeamId: number;
  router: InjectedRouter; // v3
  currentPage: number;
  /** handler that fires when a change occures on the section (e.g. disk encryption
   * enabled, profile uploaded) */
  onMutation: () => void;
}

const CustomSettings = ({
  currentTeamId,
  router,
  currentPage,
  onMutation,
}: ICustomSettingsProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [showDeleteProfileModal, setShowDeleteProfileModal] = useState(false);

  const selectedProfile = useRef<IMdmProfile | null>(null);

  const {
    data: profilesData,
    isLoading: isLoadingProfiles,
    isError: isErrorProfiles,
    refetch: refetchProfiles,
  } = useQuery<IMdmProfilesResponse, unknown>(
    [
      {
        scope: "profiles",
        team_id: currentTeamId,
        page: currentPage,
        per_page: PROFILES_PER_PAGE,
      },
    ],
    () =>
      mdmAPI.getProfiles({
        team_id: currentTeamId,
        page: currentPage,
        per_page: PROFILES_PER_PAGE,
      }),
    {
      refetchOnWindowFocus: false,
    }
  );

  const onUploadProfile = () => {
    refetchProfiles();
    onMutation();
  };

  const onCancelDelete = () => {
    selectedProfile.current = null;
    setShowDeleteProfileModal(false);
  };

  const onDeleteProfile = async (profileId: number | string) => {
    try {
      await mdmAPI.deleteProfile(profileId);
      refetchProfiles();
      onMutation();
      renderFlash("success", "Successfully deleted!");
    } catch (e) {
      renderFlash("error", "Couldn’t delete. Please try again.");
    } finally {
      selectedProfile.current = null;
      setShowDeleteProfileModal(false);
    }
  };

  // pagination controls
  const path = PATHS.CONTROLS_CUSTOM_SETTINGS.concat(
    `?team_id=${currentTeamId}`
  );

  const onPrevPage = useCallback(() => {
    router.push(path.concat(`&page=${currentPage - 1}`));
  }, [router, path, currentPage]);

  const onNextPage = useCallback(() => {
    router.push(path.concat(`&page=${currentPage + 1}`));
  }, [router, path, currentPage]);

  const onClickDelete = (profile: IMdmProfile) => {
    selectedProfile.current = profile;
    setShowDeleteProfileModal(true);
  };

  const renderProfileList = () => {
    if (isLoadingProfiles) {
      return <Spinner />;
    }

    if (isErrorProfiles) {
      return <DataError />;
    }

    if (
      !profilesData ||
      !profilesData.profiles ||
      profilesData.profiles.length === 0
    ) {
      return null;
    }

    const { profiles, meta } = profilesData;
    return (
      <>
        <UploadList
          listItems={profiles}
          HeadingComponent={ProfileListHeading}
          ListItemComponent={({ listItem }) => (
            <ProfileListItem profile={listItem} onDelete={onClickDelete} />
          )}
        />
        <Pagination
          className={`${baseClass}__pagination-controls`}
          disableNext={!meta.has_next_results}
          disablePrev={!meta.has_previous_results}
          onNextPage={onNextPage}
          onPrevPage={onPrevPage}
        />
      </>
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Custom settings" />
      <p className={`${baseClass}__description`}>
        Create and upload configuration profiles to apply custom settings.{" "}
        <CustomLink
          newTab
          text="Learn how"
          url="https://fleetdm.com/docs/using-fleet/mdm-custom-macos-settings"
        />
      </p>
      {renderProfileList()}
      <ProfileUploader
        currentTeamId={currentTeamId}
        onUpload={onUploadProfile}
      />
      {showDeleteProfileModal && selectedProfile.current && (
        <DeleteProfileModal
          profileName={selectedProfile.current?.name}
          profileId={selectedProfile.current?.profile_id}
          onCancel={onCancelDelete}
          onDelete={onDeleteProfile}
        />
      )}
    </div>
  );
};

export default CustomSettings;
