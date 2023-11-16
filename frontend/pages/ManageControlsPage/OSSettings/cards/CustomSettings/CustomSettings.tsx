import React, { useContext, useRef, useState } from "react";
import { useQuery } from "react-query";

import { IMdmProfile, IMdmProfilesResponse } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";
import { NotificationContext } from "context/notification";

import CustomLink from "components/CustomLink";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

import UploadList from "../../../components/UploadList";

import DeleteProfileModal from "./components/DeleteProfileModal/DeleteProfileModal";
import ProfileListItem from "./components/ProfileListItem";
import ProfileListHeading from "./components/ProfileListHeading";
import ProfileUploader from "./components/ProfileUploader";

const baseClass = "custom-settings";

interface ICustomSettingsProps {
  currentTeamId: number;
  /** handler that fires when a change occures on the section (e.g. disk encryption
   * enabled, profile uploaded) */
  onMutation: () => void;
}

const CustomSettings = ({
  currentTeamId,
  onMutation,
}: ICustomSettingsProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const [showDeleteProfileModal, setShowDeleteProfileModal] = useState(false);

  const selectedProfile = useRef<IMdmProfile | null>(null);

  const onClickDelete = (profile: IMdmProfile) => {
    selectedProfile.current = profile;
    setShowDeleteProfileModal(true);
  };

  const {
    data: profiles,
    isLoading: isLoadingProfiles,
    isError: isErrorProfiles,
    refetch: refetchProfiles,
  } = useQuery<IMdmProfilesResponse, unknown, IMdmProfile[] | null>(
    ["profiles", currentTeamId],
    () =>
      mdmAPI.getProfiles({
        team_id: currentTeamId,
        page: 0,
        per_page: 10,
      }),
    {
      select: (data) => data.profiles,
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

  const renderProfileList = () => {
    if (isLoadingProfiles) {
      return <Spinner />;
    }

    if (isErrorProfiles) {
      return <DataError />;
    }

    if (!profiles || profiles.length === 0) {
      return null;
    }

    return (
      <UploadList
        listItems={profiles}
        HeadingComponent={ProfileListHeading}
        ListItemComponent={({ listItem }) => (
          <ProfileListItem profile={listItem} onDelete={onClickDelete} />
        )}
      />
    );
  };

  return (
    <div className={baseClass}>
      <h2>Custom settings</h2>
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
