import React, { useContext } from "react";
import { useQuery } from "react-query";
import { AxiosResponse } from "axios";
import formatDistanceToNow from "date-fns/formatDistanceToNow";

import { IApiError } from "interfaces/errors";
import mdmAPI from "services/entities/mdm";

import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { IMdmProfile, IMdmProfilesResponse } from "interfaces/mdm";
import { NotificationContext } from "context/notification";
import { ERROR_MESSAGES, getErrorMessage } from "./helpers";

const baseClass = "custom-settings";

interface ICustomSettingsProps {
  currentTeamId?: number;
}

const CustomSettings = ({ currentTeamId }: ICustomSettingsProps) => {
  const { renderFlash } = useContext(NotificationContext);

  const {
    data: profiles,
    error: errorProfiles,
    refetch: refectchProfiles,
  } = useQuery<IMdmProfilesResponse, unknown, IMdmProfile[] | null>(
    "profiles",
    () => mdmAPI.getProfiles(currentTeamId),
    {
      select: (data) => data.profiles,
      refetchOnWindowFocus: false,
    }
  );

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
              onClick={() => {
                return null;
              }}
            >
              <Icon name="download" />
            </Button>
            <Button
              className={`${baseClass}__delete-button`}
              variant="text-icon"
              onClick={() => {
                return null;
              }}
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
    if (!files) return;

    const file = files[0];

    if (
      file.type !== "application/x-apple-aspen-config" ||
      !file.name.includes(".mobileconfig")
    ) {
      renderFlash("error", ERROR_MESSAGES.wrongType.message);
      return;
    }

    try {
      await mdmAPI.uploadProfile(file, currentTeamId);
      refectchProfiles();
      renderFlash("success", "Successfully uploaded!");
    } catch (e) {
      const error = e as AxiosResponse<IApiError>;
      const errMessage = getErrorMessage(error);
      renderFlash("error", errMessage);
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
          url="https://fleetdm.com/docs/controls#macos-settings"
        />
      </p>

      {renderProfiles()}

      <div className={`${baseClass}__profile-uploader`}>
        <Icon name="profile" />
        <p>Configuration profile (.mobileconfig)</p>
        <label htmlFor="upload-profile">Upload</label>
        <input
          accept=".mobileconfig,application/x-apple-aspen-config"
          id="upload-profile"
          type="file"
          onChange={(e) => onFileUpload(e.target.files)}
        />
      </div>
    </div>
  );
};

export default CustomSettings;
