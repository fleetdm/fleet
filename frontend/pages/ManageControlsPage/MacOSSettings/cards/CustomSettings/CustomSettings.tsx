import React, { useContext, useRef, useState } from "react";
import { AxiosResponse } from "axios";
import { format } from "date-fns";
import formatDistanceToNow from "date-fns/formatDistanceToNow";
import FileSaver from "file-saver";

import { IApiError } from "interfaces/errors";
import { IMdmProfile } from "interfaces/mdm";
import mdmAPI from "services/entities/mdm";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import { UPLOAD_ERROR_MESSAGES, getErrorMessage } from "./helpers";
import DeleteProfileModal from "./components/DeleteProfileModal/DeleteProfileModal";

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

  const onClickDownload = async (profile: IMdmProfile) => {
    const fileContent = await mdmAPI.downloadProfile(profile.profile_id);
    const formatDate = format(new Date(), "yyyy-MM-dd");
    const filename = `${formatDate}_${profile.name}.mobileconfig`;
    const file = new File([fileContent], filename);
    FileSaver.saveAs(file);
  };

  const onClickDelete = (profile: IMdmProfile) => {
    selectedProfile.current = profile;
    setShowDeleteProfileModal(true);
  };

  const renderProfiles = () => {
    if (!profiles || profiles.length === 0) return null;

    const profileListItems = profiles.map((profile) => {
      return (
        <li key={profile.profile_id} className={`${baseClass}__profile`}>
          <div className={`${baseClass}__profile-data`}>
            <Icon name="profile" />
            <div className={`${baseClass}__profile-info`}>
              <span className={`${baseClass}__profile-name`}>
                {profile.name}
              </span>
              <span className={`${baseClass}__profile-uploaded`}>
                {`Uploaded ${formatDistanceToNow(
                  new Date(profile.created_at)
                )} ago`}
              </span>
            </div>
          </div>
          <div className={`${baseClass}__profile-actions`}>
            <Button
              className={`${baseClass}__download-button`}
              variant="text-icon"
              onClick={() => onClickDownload(profile)}
            >
              <Icon name="download" />
            </Button>
            <Button
              className={`${baseClass}__delete-button`}
              variant="text-icon"
              onClick={() => onClickDelete(profile)}
            >
              <Icon name="trash" color="ui-fleet-black-75" />
            </Button>
          </div>
        </li>
      );
    });

    return (
      <div className={`${baseClass}__profiles`}>
        <div className={`${baseClass}__profiles-header`}>
          <span>Configuration profile</span>
          <span>Actions</span>
        </div>
        <ul className={`${baseClass}__profile-list`}>{profileListItems}</ul>
      </div>
    );
  };

  const onFileUpload = async (files: FileList | null) => {
    setShowLoading(true);

    if (!files || files.length === 0) return;

    const file = files[0];

    if (
      file.type !== "application/x-apple-aspen-config" ||
      !file.name.includes(".mobileconfig")
    ) {
      renderFlash("error", UPLOAD_ERROR_MESSAGES.wrongType.message);
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
      <h2>Custom Settings</h2>
      <p className={`${baseClass}__description`}>
        Create and upload configuration profiles to apply custom settings.{" "}
        <CustomLink
          newTab
          text="Learn how"
          url="https://fleetdm.com/docs/using-fleet/mobile-device-management#custom-settings"
        />
      </p>

      {renderProfiles()}

      <div className={`${baseClass}__profile-uploader`}>
        <Icon name="profile" />
        <p>Configuration profile (.mobileconfig)</p>
        <Button isLoading={showLoading}>
          <label htmlFor="upload-profile">Upload</label>
        </Button>
        <input
          accept=".mobileconfig,application/x-apple-aspen-config"
          id="upload-profile"
          type="file"
          onChange={(e) => onFileUpload(e.target.files)}
        />
      </div>
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
