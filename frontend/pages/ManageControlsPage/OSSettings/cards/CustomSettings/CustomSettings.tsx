import React, { useCallback, useContext, useRef, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import PATHS from "router/paths";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import { IMdmProfile } from "interfaces/mdm";

import mdmAPI, { IMdmProfilesResponse } from "services/entities/mdm";

import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import Pagination from "pages/ManageControlsPage/components/Pagination";

import UploadList from "../../../components/UploadList";

import AddProfileCard from "./components/ProfileUploader/components/AddProfileCard";
import AddProfileModal from "./components/ProfileUploader/components/AddProfileModal";
import DeleteProfileModal from "./components/DeleteProfileModal/DeleteProfileModal";
import ProfileLabelsModal from "./components/ProfileLabelsModal/ProfileLabelsModal";
import ProfileListItem from "./components/ProfileListItem";
import ProfileListHeading from "./components/ProfileListHeading";

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
  const { isPremiumTier } = useContext(AppContext);

  const [showAddProfileModal, setShowAddProfileModal] = useState(false);
  const [
    profileLabelsModalData,
    setProfileLabelsModalData,
  ] = useState<IMdmProfile | null>(null);
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
  const profiles = profilesData?.profiles;
  const meta = profilesData?.meta;

  const onUploadProfile = () => {
    refetchProfiles();
    onMutation();
  };

  const onCancelDelete = () => {
    selectedProfile.current = null;
    setShowDeleteProfileModal(false);
  };

  const onDeleteProfile = async (profileId: string) => {
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

    if (!profiles?.length) {
      return null;
    }

    return (
      <>
        <UploadList
          keyAttribute="profile_uuid"
          listItems={profiles}
          HeadingComponent={() =>
            ProfileListHeading({
              onClickAddProfile: () => setShowAddProfileModal(true),
            })
          }
          ListItemComponent={({ listItem }) => (
            <ProfileListItem
              isPremium={!!isPremiumTier}
              profile={listItem}
              setProfileLabelsModalData={setProfileLabelsModalData}
              onDelete={onClickDelete}
            />
          )}
        />
        <Pagination
          className={`${baseClass}__pagination-controls`}
          disableNext={!meta?.has_next_results}
          disablePrev={!meta?.has_previous_results}
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
          url="https://fleetdm.com/learn-more-about/custom-os-settings"
        />
      </p>
      {renderProfileList()}
      {!isLoadingProfiles && !isErrorProfiles && !profiles?.length && (
        <AddProfileCard
          baseClass="add-profile"
          setShowModal={setShowAddProfileModal}
        />
      )}
      {showAddProfileModal && (
        <AddProfileModal
          baseClass="add-profile"
          currentTeamId={currentTeamId}
          isPremiumTier={!!isPremiumTier}
          onUpload={onUploadProfile}
          setShowModal={setShowAddProfileModal}
        />
      )}
      {showDeleteProfileModal && selectedProfile.current && (
        <DeleteProfileModal
          profileName={selectedProfile.current?.name}
          profileId={selectedProfile.current?.profile_uuid}
          onCancel={onCancelDelete}
          onDelete={onDeleteProfile}
        />
      )}
      {!!isPremiumTier && !!profileLabelsModalData?.labels?.length && (
        <ProfileLabelsModal
          baseClass={baseClass}
          profile={profileLabelsModalData}
          setModalData={setProfileLabelsModalData}
        />
      )}
    </div>
  );
};

export default CustomSettings;
