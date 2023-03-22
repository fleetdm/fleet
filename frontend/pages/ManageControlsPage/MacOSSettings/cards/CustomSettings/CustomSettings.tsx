import React, { useContext, useRef, useState } from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { IMdmProfile } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import CustomLink from "components/CustomLink";

import FileUploader from "../../../components/FileUploader";
import UploadList from "../../../components/UploadList";

import { UPLOAD_ERROR_MESSAGES, getErrorMessage } from "./helpers";
import DeleteProfileModal from "./components/DeleteProfileModal/DeleteProfileModal";
import ProfileListItem from "./components/ProfileListItem";
import ProfileListHeading from "./components/ProfileListHeading";

const baseClass = "custom-settings";

interface ICustomSettingsProps {
  profiles: IMdmProfile[];
  onProfileUpload: () => void;
  onProfileDelete: () => void;
}

const CustomSettings = ({
  profiles,
  onProfileUpload,
  onProfileDelete,
}: ICustomSettingsProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { currentTeam } = useContext(AppContext);

  const [showDeleteProfileModal, setShowDeleteProfileModal] = useState(false);
  const [showLoading, setShowLoading] = useState(false);

  const selectedProfile = useRef<IMdmProfile | null>(null);

  const onClickDelete = (profile: IMdmProfile) => {
    selectedProfile.current = profile;
    setShowDeleteProfileModal(true);
  };

  const onFileUpload = async (files: FileList | null) => {
    setShowLoading(true);

    if (!files || files.length === 0) {
      setShowLoading(false);
      return;
    }

    const file = files[0];

    if (
      file.type !== "application/x-apple-aspen-config" ||
      !file.name.includes(".mobileconfig")
    ) {
      renderFlash("error", UPLOAD_ERROR_MESSAGES.wrongType.message);
      setShowLoading(false);
      return;
    }

    try {
      await mdmAPI.uploadProfile(file, currentTeam?.id);
      onProfileUpload();
      renderFlash("success", "Successfully uploaded!");
    } catch (e) {
      const error = e as AxiosResponse<IApiError>;
      const errMessage = getErrorMessage(error);
      renderFlash("error", errMessage);
    } finally {
      setShowLoading(false);
    }
  };

  const onCancelDelete = () => {
    selectedProfile.current = null;
    setShowDeleteProfileModal(false);
  };

  const onDeleteProfile = async (profileId: number) => {
    try {
      await mdmAPI.deleteProfile(profileId);
      onProfileDelete();
      renderFlash("success", "Successfully deleted!");
    } catch (e) {
      renderFlash("error", "Couldn’t delete. Please try again.");
    } finally {
      selectedProfile.current = null;
      setShowDeleteProfileModal(false);
    }
  };

  return (
    <div className={baseClass}>
      <h2>Custom settings</h2>
      <p className={`${baseClass}__description`}>
        Create and upload configuration profiles to apply custom settings.{" "}
        <CustomLink
          newTab
          text="Learn how"
          url="https://fleetdm.com/docs/using-fleet/mdm-macos-settings#custom-settings"
        />
      </p>

      {profiles && (
        <UploadList
          listItems={profiles}
          HeadingComponent={ProfileListHeading}
          ListItemComponent={({ listItem }) => (
            <ProfileListItem profile={listItem} onDelete={onClickDelete} />
          )}
        />
      )}

      <FileUploader
        icon="profile"
        message="Configuration profile (.mobileconfig)"
        isLoading={showLoading}
        onFileUpload={onFileUpload}
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
