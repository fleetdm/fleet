import React, { useContext, useState } from "react";
import { AxiosResponse } from "axios";

import { IApiError } from "interfaces/errors";
import { NotificationContext } from "context/notification";
import mdmAPI from "services/entities/mdm";

import FileUploader from "components/FileUploader";

import { UPLOAD_ERROR_MESSAGES, getErrorMessage } from "./helpers";

const baseClass = "profile-uploader";

interface IProfileUploaderProps {
  currentTeamId: number;
  onUpload: () => void;
}

const ProfileUploader = ({
  currentTeamId,
  onUpload,
}: IProfileUploaderProps) => {
  const [showLoading, setShowLoading] = useState(false);

  const { renderFlash } = useContext(NotificationContext);

  const onFileUpload = async (files: FileList | null) => {
    setShowLoading(true);

    if (!files || files.length === 0) {
      setShowLoading(false);
      return;
    }

    const file = files[0];

    if (
      // file.type might be empty on some systems as uncommon file extensions
      // would return an empty string.
      (file.type !== "" && file.type !== "application/x-apple-aspen-config") ||
      !file.name.includes(".mobileconfig")
    ) {
      renderFlash("error", UPLOAD_ERROR_MESSAGES.wrongType.message);
      setShowLoading(false);
      return;
    }

    try {
      await mdmAPI.uploadProfile(file, currentTeamId);
      renderFlash("success", "Successfully uploaded!");
      onUpload();
    } catch (e) {
      const error = e as AxiosResponse<IApiError>;
      const errMessage = getErrorMessage(error);
      renderFlash("error", errMessage);
    } finally {
      setShowLoading(false);
    }
  };

  return (
    <FileUploader
      graphicNames={["file-configuration-profile"]}
      message="Configuration profile (.mobileconfig)"
      accept=".mobileconfig,application/x-apple-aspen-config"
      isLoading={showLoading}
      onFileUpload={onFileUpload}
      className={`${baseClass}__file-uploader`}
    />
  );
};

export default ProfileUploader;
